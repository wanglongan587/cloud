package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/wanglongan587/cloud/internal/core"
)

func TestSquadLeaderRetirementAndTenantIsolation(t *testing.T) {
	f := setup(t)
	squad := createTestSquad(t, f, "Agents")
	agent := uuid.NewString()
	_, err := f.store.Pool.Exec(`INSERT INTO space_agents(id,tenant_id,space_id,plugin_id,display_name,status) VALUES($1,$2,$3,'test/agent','Agent','active')`, agent, f.tid, f.fixtureSpace().S("id"))
	must(t, err)
	path := f.path("/squads/" + squad.S("id"))
	leader := f.call("POST", path+"/leader", core.Object{"agentId": agent, "version": squad.N("version")}, "set-leader", 200)
	if leader.S("leaderAgentId") != agent || f.scalar("SELECT count(*) FROM squad_members WHERE squad_id=$1 AND agent_id=$2 AND ended_at IS NULL", squad.S("id"), agent) != 1 {
		t.Fatal("leader and agent grant were not committed together")
	}
	var memberID string
	must(t, f.store.Pool.QueryRow("SELECT id FROM squad_members WHERE squad_id=$1 AND agent_id=$2 AND ended_at IS NULL", squad.S("id"), agent).Scan(&memberID))
	f.call("DELETE", path+"/members/"+memberID, core.Object{"version": leader.N("version")}, "uncleared-leader", 409)
	_, err = f.store.Pool.Exec("UPDATE space_agents SET status='retired' WHERE id=$1", agent)
	must(t, err)
	_, err = f.store.Pool.Exec("UPDATE space_agents SET status='active' WHERE id=$1", agent)
	must(t, err)
	after := f.call("GET", path, nil, "", 200)
	for _, method := range []string{"PATCH", "DELETE"} {
		f.call(method, path+"/members/not-a-uuid", core.Object{"version": after.N("version")}, "invalid-"+method, 404)
	}
	if after.S("leaderAgentId") != "" || f.scalar("SELECT count(*) FROM squad_members WHERE agent_id=$1 AND ended_at IS NULL", agent) != 0 {
		t.Fatal("agent reinstallation revived old leader or membership")
	}
	f.call("PATCH", path, core.Object{"name": "Stale", "version": leader.N("version")}, "", 409)
	other := joinUser("other-tenant-owner")
	created := joinCall(t, f, other, "POST", "/api/v1/tenants", "foreign", core.Object{"name": "Other", "slug": "other-squad-test"}, 201)
	foreignPath := "/api/v1/tenants/" + created.O("tenant").S("id") + "/squads/" + squad.S("id")
	joinCall(t, f, other, "GET", foreignPath, "", nil, 404)
	joinCall(t, f, other, "POST", "/api/v1/tenants/"+created.O("tenant").S("id")+"/invitations", "cross-target", core.Object{"token": joinToken('X'), "targetSquadIds": []string{squad.S("id")}}, 404)
	if f.scalar("SELECT count(*) FROM issue_runs") != 0 {
		t.Fatal("squad configuration initiated execution")
	}
}

func TestGlobalUserDisableClosesEveryTenantGrantAndIssuerLink(t *testing.T) {
	f := setup(t)
	user, uid := f.addUser(t, "global", "Global")
	f.call("PUT", f.path("/members/"+uid), core.Object{"role": "admin", "status": "active", "version": 1}, "", 200)
	other := joinCall(t, f, user, "POST", "/api/v1/tenants", "global-second", core.Object{"name": "Second", "slug": "global-second"}, 201)
	_, err := f.store.Pool.Exec("INSERT INTO tenant_memberships(tenant_id,user_id,role,status) VALUES($1,$2,'admin','active')", other.O("tenant").S("id"), f.uid)
	must(t, err)
	for index, tid := range []string{f.tid, other.O("tenant").S("id")} {
		base := "/api/v1/tenants/" + tid
		joinCall(t, f, user, "POST", base+"/squads", "global-squad", core.Object{"name": "Global"}, 201)
		joinCall(t, f, user, "POST", base+"/invitations", "global-link", core.Object{"token": joinToken(byte('a' + index))}, 201)
	}
	_, err = f.store.Pool.Exec("UPDATE users SET status='disabled' WHERE id=$1", uid)
	must(t, err)
	_, err = f.store.Pool.Exec("UPDATE users SET status='active' WHERE id=$1", uid)
	must(t, err)
	if f.scalar("SELECT count(*) FROM tenant_invitations WHERE created_by=$1 AND invalidated_at IS NOT NULL", uid) != 2 || f.scalar("SELECT count(*) FROM squad_members WHERE user_id=$1 AND ended_at IS NULL", uid) != 0 {
		t.Fatal("global disable missed a tenant or reactivation revived a grant")
	}
}

func TestSquadPendingIntentConflictAndApprovalRollback(t *testing.T) {
	f := setup(t)
	a, b := createTestSquad(t, f, "First"), createTestSquad(t, f, "Second")
	user := joinUser("requester")
	for _, item := range []struct{ squad, token string }{{a.S("id"), joinToken('p')}, {b.S("id"), joinToken('q')}} {
		f.call("POST", f.path("/join-links"), core.Object{"token": item.token, "targetSquadIds": []string{item.squad}}, "link-"+item.squad, 201)
	}
	r := joinCall(t, f, user, "POST", "/api/v1/join/requests", "request-a", core.Object{"token": joinToken('p')}, 201)
	joinCall(t, f, user, "POST", "/api/v1/join/requests", "request-b", core.Object{"token": joinToken('q')}, 409)
	preview := joinCall(t, f, user, "POST", core.JoinLinkPreviewPath, "", core.Object{"kind": "apply", "token": joinToken('q')}, 200)
	if preview.S("applicationStatus") != "conflict" || len(preview["pendingTargetSquads"].([]any)) != 1 {
		t.Fatalf("different intent was misrepresented as pending: %v", preview)
	}
	list := f.call("GET", f.path("/join-requests"), nil, "", 200)
	if len(core.Object(list["items"].([]any)[0].(map[string]any))["targetSquads"].([]any)) != 1 {
		t.Fatal("approver cannot inspect frozen targets")
	}
	targetedInvite(t, f, joinToken('z'), a.S("id"))
	joinCall(t, f, user, "POST", "/api/v1/join/invitations/redeem", "join-before-decision", core.Object{"token": joinToken('z')}, 200)
	repeated := joinCall(t, f, user, "POST", "/api/v1/join/requests", "request-a-again", core.Object{"token": joinToken('p')}, 201)
	if repeated.S("id") != r.S("id") {
		t.Fatal("same intent did not return existing pending request")
	}
	current := f.call("GET", f.path("/squads/"+a.S("id")), nil, "", 200)
	f.call("POST", f.path("/squads/"+a.S("id")+"/archive"), core.Object{"version": current.N("version")}, "archive-request-target", 200)
	f.call("POST", f.path("/join-requests/"+r.S("id")+"/approve"), core.Object{"version": r.N("version")}, "approve-dead-target", 404)
	if f.scalar("SELECT count(*) FROM tenant_join_requests WHERE id=$1 AND status='pending'", r.S("id")) != 1 || f.scalar("SELECT count(*) FROM squad_members WHERE user_id=$1 AND ended_at IS NULL", r.S("userId")) != 0 {
		t.Fatal("failed approval changed intent or partially admitted the applicant")
	}
	_, err := f.store.Pool.Exec("UPDATE tenant_join_requests SET target_squad_ids='[]' WHERE id=$1", r.S("id"))
	wantPGError(t, err, "23514")
}

func TestSquadDirectoryJoinUsesTheSameAtomicSources(t *testing.T) {
	f := setup(t)
	squad := createTestSquad(t, f, "Directory")
	person := core.DirectoryPerson{GlobalUserID: "750123", Name: "Employee", Employed: true}
	r := &core.PublicRequest{Method: "POST", Path: f.path("/members/huawei"), TenantID: f.tid, Key: "directory-target", Identity: &f.user, Person: &person,
		Body: core.Object{"globalUserId": person.GlobalUserID, "targetSquadIds": []any{squad.S("id")}}}
	out, status, err := f.store.Public(context.Background(), r)
	must(t, err)
	if status != 200 || f.scalar("SELECT count(*) FROM squad_members WHERE user_id=$1 AND ended_at IS NULL", out.S("userId")) != 1 || f.scalar("SELECT count(*) FROM membership_join_events WHERE user_id=$1 AND method='directory'", out.S("userId")) != 1 {
		t.Fatal("directory admission did not share the qualification/source transaction")
	}
}
