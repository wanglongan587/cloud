package integration

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"github.com/wanglongan587/cloud/internal/core"
)

func TestSquadArchiveAndRedemptionHaveOneAtomicOrder(t *testing.T) {
	f := setup(t)
	squad := createTestSquad(t, f, "Concurrent")
	token := joinToken('C')
	invite := targetedInvite(t, f, token, squad.S("id"))
	user := joinUser("racing-user")
	identity := joinCall(t, f, user, "GET", "/api/v1/me", "", nil, 200)
	type outcome struct {
		kind   string
		status int
		err    error
	}
	start, results := make(chan struct{}), make(chan outcome, 2)
	for _, action := range []string{"redeem", "archive"} {
		go func() {
			<-start
			caller, path, body := user, "/api/v1/join/invitations/redeem", core.Object{"token": token}
			if action == "archive" {
				caller, path, body = f.user, f.path("/squads/"+squad.S("id")+"/archive"), core.Object{"version": squad.N("version")}
			}
			gateway := core.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "gateway-a"}}
			_, status, err := f.client.Call(context.Background(), "POST", path, "gateway", gateway, &caller, action, body)
			results <- outcome{action, status, err}
		}()
	}
	close(start)
	observed := map[string]int{}
	for range 2 {
		result := <-results
		must(t, result.err)
		observed[result.kind] = result.status
	}
	if observed["redeem"] == 200 && observed["archive"] == 409 {
		current := f.call("GET", f.path("/squads/"+squad.S("id")), nil, "", 200)
		f.call("POST", f.path("/squads/"+squad.S("id")+"/archive"), core.Object{"version": current.N("version")}, "archive-current", 200)
	} else if observed["redeem"] != 404 || observed["archive"] != 200 {
		t.Fatalf("non-atomic race outcome: %v", observed)
	}
	if f.scalar("SELECT count(*) FROM squad_members WHERE squad_id=$1 AND ended_at IS NULL", squad.S("id")) != 0 {
		t.Fatal("archived squad retained active grants")
	}
	consumed := f.scalar("SELECT count(*) FROM tenant_invitations WHERE id=$1 AND consumed_at IS NOT NULL", invite.S("id"))
	sources := f.scalar("SELECT count(*) FROM membership_join_events WHERE user_id=$1", identity.S("id"))
	if sources != consumed || (observed["redeem"] == 404 && sources != 0) {
		t.Fatal("race left partial admission or provenance")
	}
}
