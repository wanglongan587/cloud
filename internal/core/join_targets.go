package core

import (
	"sort"

	"github.com/google/uuid"
)

// joinTargets canonicalizes a bounded intent before it is frozen on the link or
// request. Redemption never takes targets from the accepting client's body.
func joinTargets(body Object) []string {
	ids := []string{}
	if body["targetSquadIds"] == nil {
		return ids
	}
	values, ok := body["targetSquadIds"].([]any)
	require(ok, 400, "invalid_squad_targets")
	seen := map[string]bool{}
	for _, value := range values {
		id, ok := value.(string)
		require(ok && validID(id), 400, "invalid_squad_targets")
		parsed, err := uuid.Parse(id)
		require(err == nil, 400, "invalid_squad_targets")
		id = parsed.String()
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	require(len(ids) <= 20, 400, "too_many_squad_targets")
	sort.Strings(ids)
	return ids
}

func requireJoinTargets(t *transaction, tid string, ids []string) {
	for _, id := range ids {
		require(t.one("SELECT id FROM squads WHERE id=$1 AND tenant_id=$2 AND archived_at IS NULL", id, tid) != nil, 404, "join_target_unavailable")
	}
}

func addJoinTargets(t *transaction, tid, uid, actor, source, sourceID string, ids []string) bool {
	changed := false
	for _, id := range ids {
		if addSquadMember(t, squadRow(t, tid, id), "user", uid, false, "", actor, source, sourceID) {
			changed = true
			t.exec("UPDATE squads SET version=version+1,updated_at=clock_timestamp() WHERE id=$1", id)
		}
	}
	return changed
}

func missingJoinTargets(t *transaction, tid, uid string, ids []string) bool {
	for _, id := range ids {
		if t.one("SELECT id FROM squad_members WHERE tenant_id=$1 AND squad_id=$2 AND user_id=$3 AND ended_at IS NULL", tid, id, uid) == nil {
			return true
		}
	}
	return false
}
