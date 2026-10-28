package integration

import (
	"crypto/sha256"
	"testing"

	"github.com/google/uuid"
)

func TestSquadUpgradePreservesKnownFactsAndInvalidatesOnlyUnauthorizedIssuers(t *testing.T) {
	pool, _ := testSchema(t, "squad_upgrade_")
	applyMigrationsUpTo(t, pool, migrationsBefore(t, "0074_squad_targeted_joining.sql"))
	seed := seedOwnershipChain(t, pool)
	unauthorized, inviteGood, inviteBad, consumed := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	_, err := pool.Exec("INSERT INTO users(id,display_name,status) VALUES($1,'Member issuer','active')", unauthorized)
	must(t, err)
	_, err = pool.Exec("INSERT INTO tenant_memberships(tenant_id,user_id,role,status) VALUES($1,$2,'member','active')", seed.tenantID, unauthorized)
	must(t, err)
	for i, row := range []struct{ id, creator string }{{inviteGood, seed.userID}, {inviteBad, unauthorized}, {consumed, unauthorized}} {
		digest := sha256.Sum256([]byte{byte(i)})
		_, err = pool.Exec("INSERT INTO tenant_invitations(id,tenant_id,token_hash,created_by,expires_at) VALUES($1,$2,$3,$4,now()+interval '7 days')", row.id, seed.tenantID, digest[:], row.creator)
		must(t, err)
	}
	_, err = pool.Exec("UPDATE tenant_invitations SET consumed_by=$2,consumed_at=now() WHERE id=$1", consumed, seed.userID)
	must(t, err)
	var before, after string
	must(t, pool.QueryRow("SELECT row_to_json(i)::text FROM tenant_invitations i WHERE id=$1", inviteGood).Scan(&before))
	store := newStoreOnSchema(t, pool)
	migratedTwice(t, store)
	must(t, store.CheckSchema(t.Context()))
	// Compare every pre-existing column; the upgrade may only append policy data.
	must(t, pool.QueryRow("SELECT (to_jsonb(i)-'target_squad_ids'-'invalidated_at')::text FROM tenant_invitations i WHERE id=$1", inviteGood).Scan(&after))
	var equal bool
	must(t, pool.QueryRow("SELECT $1::jsonb=$2::jsonb", before, after).Scan(&equal))
	if !equal {
		t.Fatal("upgrade changed legitimate existing token metadata")
	}
	var invalidated, squads, sources, activation int
	must(t, pool.QueryRow("SELECT count(*) FROM tenant_invitations WHERE invalidated_at IS NOT NULL AND id=$1", inviteBad).Scan(&invalidated))
	must(t, pool.QueryRow("SELECT count(*) FROM squads").Scan(&squads))
	must(t, pool.QueryRow("SELECT count(*) FROM membership_join_events").Scan(&sources))
	must(t, pool.QueryRow("SELECT count(*) FROM join_policy_activation").Scan(&activation))
	if invalidated != 1 || squads != 0 || sources != 0 || activation != 1 {
		t.Fatalf("upgrade fabricated history or missed activation: invalidated=%d squads=%d sources=%d activation=%d", invalidated, squads, sources, activation)
	}
	var retained int
	must(t, pool.QueryRow("SELECT count(*) FROM tenant_invitations WHERE id=$1 AND consumed_by=$2 AND invalidated_at IS NULL AND target_squad_ids='[]'", consumed, seed.userID).Scan(&retained))
	if retained != 1 {
		t.Fatal("upgrade rewrote an already consumed invitation")
	}
}
