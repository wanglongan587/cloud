package core

import "strings"

// Squad identity and member references do not grant execution or resource access.
// All callers first pass the existing current tenant-membership authorization.
func squadRow(t *transaction, tid, id string) Object {
	require(validID(id), 404, "not_found")
	row := t.one("SELECT * FROM squads WHERE id=$1 AND tenant_id=$2", id, tid)
	require(row != nil, 404, "not_found")
	return row
}

func canMaintainSquad(t *transaction, row Object, uid string) bool {
	return membership(t, row.S("tenantId"), uid, false).S("role") == "admin" ||
		t.one("SELECT id FROM squad_members WHERE squad_id=$1 AND user_id=$2 AND maintainer AND ended_at IS NULL", row.S("id"), uid) != nil
}

func squadRead(t *transaction, r *PublicRequest, uid string) Object {
	if strings.HasSuffix(r.Path, "/squad-agents") {
		return page(t, "SELECT id,display_name,plugin_id,status FROM space_agents WHERE tenant_id=$1 AND status='active'", []any{r.TenantID}, "id", r)
	}
	if r.SquadID == "" {
		return page(t, `SELECT s.*,
 (SELECT count(*) FROM squad_members m WHERE m.squad_id=s.id AND m.kind='user' AND m.ended_at IS NULL) AS human_count,
 (SELECT count(*) FROM squad_members m WHERE m.squad_id=s.id AND m.kind='agent' AND m.ended_at IS NULL) AS agent_count
 FROM squads s WHERE s.tenant_id=$1 AND s.archived_at IS NULL`, []any{r.TenantID}, "s.id", r)
	}
	row := squadRow(t, r.TenantID, r.SquadID)
	if strings.HasSuffix(r.Path, "/members") {
		return page(t, `SELECT m.*,COALESCE(u.display_name,a.display_name) AS display_name,a.plugin_id
 FROM squad_members m LEFT JOIN users u ON u.id=m.user_id LEFT JOIN space_agents a ON a.id=m.agent_id
 WHERE m.squad_id=$1`, []any{r.SquadID}, "m.id", r)
	}
	row["canMaintain"] = canMaintainSquad(t, row, uid)
	row["canInvite"] = membership(t, r.TenantID, uid, false).S("role") == "admin"
	row["executionAvailable"] = false
	return row
}

func squadWrite(t *transaction, r *PublicRequest, uid string) (Object, int) {
	if r.SquadID == "" {
		id := newID()
		t.exec(`INSERT INTO squads(id,tenant_id,space_id,name,description,instructions,created_by)
 VALUES($1,$2,$3,$4,$5,$6,$7)`, id, r.TenantID, soleSpace(t, r.TenantID).S("id"), validText(r.Body.S("name"), 200), squadText(r.Body.S("description"), 2000), squadText(r.Body.S("instructions"), 8000), uid)
		row := squadRow(t, r.TenantID, id)
		addSquadMember(t, row, "user", uid, true, "", uid, "created", "")
		return row, 201
	}
	row := squadRow(t, r.TenantID, r.SquadID)
	require(row["archivedAt"] == nil, 409, "squad_archived")
	version(row, r.Body.N("version"))
	if strings.HasSuffix(r.Path, "/leave") {
		t.exec("UPDATE squad_members SET ended_at=clock_timestamp(),ended_reason='left',version=version+1 WHERE squad_id=$1 AND user_id=$2 AND ended_at IS NULL", r.SquadID, uid)
	} else {
		require(canMaintainSquad(t, row, uid), 403, "squad_maintainer_required")
		switch {
		case strings.HasSuffix(r.Path, "/archive"):
			t.exec("UPDATE squads SET archived_at=clock_timestamp(),leader_agent_id=NULL WHERE id=$1", r.SquadID)
		case strings.HasSuffix(r.Path, "/leader"):
			leader := r.Body.S("agentId")
			if leader != "" {
				addSquadMember(t, row, "agent", leader, false, "", uid, "leader", "")
			}
			t.exec("UPDATE squads SET leader_agent_id=NULLIF($2,'')::uuid WHERE id=$1", r.SquadID, leader)
		case strings.HasSuffix(r.Path, "/members"):
			addSquadMember(t, row, r.Body.S("kind"), r.Body.S("memberId"), r.Body.B("maintainer"), squadText(r.Body.S("responsibility"), 2000), uid, "manual", "")
		case r.SquadMemberID != "":
			member := t.one("SELECT * FROM squad_members WHERE id=$1 AND squad_id=$2 AND ended_at IS NULL", r.SquadMemberID, r.SquadID)
			require(member != nil, 404, "not_found")
			if r.Method == "DELETE" {
				if row.S("leaderAgentId") != "" && row.S("leaderAgentId") == member.S("agentId") {
					require(r.Body.B("clearLeader"), 409, "squad_leader_clear_required")
					t.exec("UPDATE squads SET leader_agent_id=NULL WHERE id=$1", r.SquadID)
				}
				t.exec("UPDATE squad_members SET ended_at=clock_timestamp(),ended_reason='removed',version=version+1 WHERE id=$1", r.SquadMemberID)
			} else {
				require(member.S("kind") == "user" || !r.Body.B("maintainer"), 400, "agent_cannot_maintain")
				t.exec("UPDATE squad_members SET responsibility=$2,maintainer=$3,version=version+1 WHERE id=$1", r.SquadMemberID, squadText(r.Body.S("responsibility"), 2000), r.Body.B("maintainer"))
			}
		default:
			t.exec("UPDATE squads SET name=$2,description=$3,instructions=$4 WHERE id=$1", r.SquadID, validText(r.Body.S("name"), 200), squadText(r.Body.S("description"), 2000), squadText(r.Body.S("instructions"), 8000))
		}
	}
	t.exec("UPDATE squads SET version=version+1,updated_at=clock_timestamp() WHERE id=$1", r.SquadID)
	t.exec("INSERT INTO squad_changes(id,squad_id,actor_user_id,reason,details) VALUES($1,$2,$3,$4,$5)", newID(), r.SquadID, uid, r.Method+" "+r.Path, jsonText(r.Body))
	return squadRow(t, r.TenantID, r.SquadID), 200
}

func squadText(value string, limit int) string {
	value = strings.TrimSpace(value)
	require(len(value) <= limit, 400, "invalid_input")
	return value
}

// addSquadMember preserves prior relationships and returns whether a new grant
// was made. Existing grants are never silently promoted or rewritten by joining.
func addSquadMember(t *transaction, squad Object, kind, id string, maintainer bool, responsibility, actor, source, sourceID string) bool {
	require(validID(id), 404, "not_found")
	require(squad["archivedAt"] == nil, 409, "squad_archived")
	var user, agent any
	switch kind {
	case "user":
		require(t.one(`SELECT m.user_id FROM tenant_memberships m JOIN users u ON u.id=m.user_id
 WHERE m.tenant_id=$1 AND m.user_id=$2 AND m.status='active' AND u.status='active' AND u.deleted_at IS NULL`, squad.S("tenantId"), id) != nil, 404, "not_found")
		user = id
	case "agent":
		require(!maintainer, 400, "agent_cannot_maintain")
		require(t.one("SELECT id FROM space_agents WHERE id=$1 AND tenant_id=$2 AND space_id=$3 AND status='active'", id, squad.S("tenantId"), squad.S("spaceId")) != nil, 404, "not_found")
		agent = id
	default:
		reject(400, "invalid_squad_member_kind")
	}
	if t.one("SELECT id FROM squad_members WHERE squad_id=$1 AND kind=$2 AND (user_id=$3 OR agent_id=$4) AND ended_at IS NULL", squad.S("id"), kind, user, agent) != nil {
		return false
	}
	t.exec(`INSERT INTO squad_members(id,squad_id,tenant_id,kind,user_id,agent_id,maintainer,responsibility,joined_by,source_kind,source_id)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,'')::uuid)`, newID(), squad.S("id"), squad.S("tenantId"), kind, user, agent, maintainer, responsibility, actor, source, sourceID)
	return true
}
