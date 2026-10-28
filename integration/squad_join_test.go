package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"github.com/wanglongan587/cloud/internal/core"
)

func createTestSquad(t *testing.T, f *fixture, name string) core.Object {
	t.Helper()
	return f.call("POST", f.path("/squads"), core.Object{"name": name}, "create-"+name, 201)
}

func targetedInvite(t *testing.T, f *fixture, token string, squads ...string) core.Object {
	t.Helper()
	return f.call("POST", f.path("/invitations"), core.Object{"token": token, "targetSquadIds": squads}, "issue-"+token, 201)
}

func TestSquadTargetedInvitationIsAtomicAndSeparatesProvenance(t *testing.T) {
	f := setup(t)
	a, b := createTestSquad(t, f, "A"), createTestSquad(t, f, "B")
	user, uid := f.addUser(t, "existing", "Existing")
	token := joinToken('T')
	targetedInvite(t, f, token, strings.ToUpper(a.S("id")), b.S("id"), a.S("id"))
	before := f.scalar("SELECT count(*) FROM membership_join_events WHERE user_id=$1", uid)
	preview := joinCall(t, f, user, "POST", core.JoinLinkPreviewPath, "", core.Object{"kind": "invite", "token": token}, 200)
	if !preview.B("alreadyMember") || preview.B("alreadyJoinedTargets") || len(preview["targetSquads"].([]any)) != 2 {
		t.Fatalf("target confirmation missing: %v", preview)
	}
	joinCall(t, f, user, "POST", "/api/v1/join/invitations/redeem", "targeted", core.Object{"token": token}, 200)
	if f.scalar("SELECT count(*) FROM squad_members WHERE user_id=$1 AND ended_at IS NULL", uid) != 2 || f.scalar("SELECT count(*) FROM membership_join_events WHERE user_id=$1", uid) != before {
		t.Fatal("pure squad admission changed workspace provenance or missed a target")
	}
	noOp := joinToken('U')
	invite := targetedInvite(t, f, noOp, a.S("id"), b.S("id"))
	notices, cancel := f.store.Events.Subscribe(f.fixtureSpace().S("id"))
	t.Cleanup(cancel)
	joinCall(t, f, user, "POST", "/api/v1/join/invitations/redeem", "all-present", core.Object{"token": noOp}, 200)
	if f.scalar("SELECT count(*) FROM tenant_invitations WHERE id=$1 AND consumed_at IS NOT NULL", invite.S("id")) != 0 {
		t.Fatal("target no-op consumed token")
	}
	select {
	case event := <-notices:
		t.Fatalf("target no-op notified: %v", event)
	default:
	}
	current := f.call("GET", f.path("/squads/"+b.S("id")), nil, "", 200)
	f.call("POST", f.path("/squads/"+b.S("id")+"/archive"), core.Object{"version": current.N("version")}, "archive-b", 200)
	newUser := joinUser("outsider")
	joinCall(t, f, newUser, "POST", "/api/v1/join/invitations/redeem", "archived-target", core.Object{"token": noOp}, 404)
	if f.scalar("SELECT count(*) FROM tenant_memberships m JOIN user_identities i ON i.user_id=m.user_id WHERE i.subject='outsider'") != 0 {
		t.Fatal("invalid target left partial workspace admission")
	}
	gw := core.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "gateway-a"}}
	out, status, err := f.client.Call(context.Background(), "POST", core.JoinLinkPreviewPath, "gateway", gw, nil, "", core.Object{"kind": "invite", "token": noOp})
	must(t, err)
	if status != 200 || len(out) != 2 || out.S("state") != "unavailable" {
		t.Fatalf("anonymous invalid target leaked detail: %d %v", status, out)
	}
}

func TestIssuerRevocationNeverRevivesAndPendingApprovalIsIndependent(t *testing.T) {
	f := setup(t)
	other, otherID := f.addUser(t, "other-admin", "Other")
	f.call("PUT", f.path("/members/"+otherID), core.Object{"role": "admin", "status": "active", "version": 1}, "", 200)
	squad := createTestSquad(t, f, "Team")
	token := joinToken('I')
	joinCall(t, f, other, "POST", f.path("/invitations"), "other-invite", core.Object{"token": token, "targetSquadIds": []string{squad.S("id")}}, 201)
	apply := joinToken('J')
	joinCall(t, f, other, "POST", f.path("/join-links"), "other-application", core.Object{"token": apply, "targetSquadIds": []string{squad.S("id")}}, 201)
	request := joinCall(t, f, joinUser("applicant"), "POST", "/api/v1/join/requests", "apply", core.Object{"token": apply}, 201)
	f.call("PUT", f.path("/members/"+otherID), core.Object{"role": "member", "status": "active", "version": 2}, "", 200)
	f.call("PUT", f.path("/members/"+otherID), core.Object{"role": "admin", "status": "active", "version": 3}, "", 200)
	for _, value := range []struct{ kind, token string }{{"invite", token}, {"apply", apply}} {
		preview := joinCall(t, f, f.user, "POST", core.JoinLinkPreviewPath, "", core.Object{"kind": value.kind, "token": value.token}, 200)
		if preview.S("state") != "unavailable" {
			t.Fatalf("restoring admin revived old link: %v", preview)
		}
	}
	joinCall(t, f, joinUser("late-user"), "POST", "/api/v1/join/invitations/redeem", "late", core.Object{"token": token}, 404)
	f.call("POST", f.path("/join-requests/"+request.S("id")+"/approve"), core.Object{"version": request.N("version")}, "independent-approval", 200)
	if f.scalar("SELECT count(*) FROM squad_members WHERE user_id=$1 AND ended_at IS NULL", request.S("userId")) != 1 {
		t.Fatal("current administrator could not approve frozen pending intent")
	}
}

func TestSquadMaintainerRevocationAndReentryNeverRestoreOldGrants(t *testing.T) {
	f := setup(t)
	user, uid := f.addUser(t, "creator", "Creator")
	squad := joinCall(t, f, user, "POST", f.path("/squads"), "created", core.Object{"name": "Created"}, 201)
	path := f.path("/squads/" + squad.S("id"))
	f.call("PUT", f.path("/members/"+uid), core.Object{"role": "member", "status": "disabled", "version": 1}, "", 200)
	if f.scalar("SELECT count(*) FROM squad_members WHERE user_id=$1 AND ended_at IS NULL", uid) != 0 {
		t.Fatal("workspace removal left active squad grants")
	}
	joinCall(t, f, user, "GET", path, "", nil, 403)
	token := joinToken('V')
	targetedInvite(t, f, token, squad.S("id"))
	joinCall(t, f, user, "POST", "/api/v1/join/invitations/redeem", "rejoin", core.Object{"token": token}, 200)
	row := joinCall(t, f, user, "GET", path, "", nil, 200)
	if row.B("canMaintain") || row.B("executionAvailable") {
		t.Fatalf("reentry restored creator privilege or execution: %v", row)
	}
	joinCall(t, f, user, "PATCH", path, "", core.Object{"name": "Forbidden", "version": row.N("version")}, 403)
	f.call("PATCH", path, core.Object{"name": "Admin takeover", "version": row.N("version")}, "", 200)
	if f.scalar("SELECT count(*) FROM squad_members WHERE user_id=$1", uid) != 2 {
		t.Fatal("reentry overwrote historical grant")
	}
}
