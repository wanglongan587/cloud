package contract

import (
	"strings"

	"github.com/wanglongan587/cloud/internal/api/router"
)

func squadSchemas(s obj) {
	s["SquadAgent"] = resource("id displayName pluginId status", "")
	s["Squad"] = object(obj{
		"id": uuid(), "tenantId": uuid(), "spaceId": uuid(), "name": str(), "description": str(), "instructions": str(),
		"createdBy": uuid(), "createdAt": timestamp(), "updatedAt": timestamp(), "archivedAt": optional(timestamp()),
		"leaderAgentId": optional(uuid()), "version": number(), "humanCount": number(), "agentCount": number(),
		"canMaintain": boolean(), "canInvite": boolean(), "executionAvailable": boolean(),
	}, "id", "tenantId", "spaceId", "name", "description", "instructions", "createdBy", "createdAt", "updatedAt", "archivedAt", "leaderAgentId", "version")
	s["SquadMember"] = object(obj{
		"id": uuid(), "tenantId": uuid(), "squadId": uuid(), "kind": enumeration("user", "agent"),
		"userId": optional(uuid()), "agentId": optional(uuid()), "displayName": str(), "pluginId": optional(str()),
		"responsibility": str(), "maintainer": boolean(), "joinedBy": uuid(), "sourceKind": str(), "sourceId": optional(uuid()),
		"joinedAt": timestamp(), "endedAt": optional(timestamp()), "endedReason": optional(str()), "version": number(),
	}, "id", "tenantId", "squadId", "kind", "userId", "agentId", "displayName", "responsibility", "maintainer", "joinedBy", "sourceKind", "sourceId", "joinedAt", "endedAt", "endedReason", "version")
	for _, name := range []string{"Invitation", "JoinLink", "JoinRequest"} {
		properties(s, name)["targetSquadIds"] = array(uuid())
	}
	for _, name := range []string{"Invitation", "JoinLink"} {
		properties(s, name)["invalidatedAt"] = optional(timestamp())
	}
	properties(s, "JoinLinkPreview")["alreadyJoinedTargets"] = boolean()
	properties(s, "JoinLinkPreview")["applicationStatus"] = enumeration("pending", "conflict")
	properties(s, "JoinLinkPreview")["pendingTargetSquads"] = array(object(obj{"id": uuid(), "name": str()}, "id", "name"))
	properties(s, "JoinRequest")["targetSquads"] = array(object(obj{"id": uuid(), "name": str(), "archived": boolean()}, "id", "name", "archived"))
	properties(s, "JoinLinkPreview")["inviterName"] = str()
	properties(s, "JoinLinkPreview")["targetSquads"] = array(object(obj{"id": uuid(), "name": str()}, "id", "name"))
}

func squadResponse(r router.Route) (obj, string) {
	if strings.HasSuffix(r.Path, "/squad-agents") {
		return object(obj{"items": array(ref("SquadAgent")), "nextCursor": str()}, "items", "nextCursor"), "200"
	}
	if r.Method == "GET" && strings.HasSuffix(r.Path, "/members") {
		return object(obj{"items": array(ref("SquadMember")), "nextCursor": str()}, "items", "nextCursor"), "200"
	}
	if r.Method == "GET" && strings.HasSuffix(r.Path, "/squads") {
		return object(obj{"items": array(ref("Squad")), "nextCursor": str()}, "items", "nextCursor"), "200"
	}
	if r.Method == "POST" && strings.HasSuffix(r.Path, "/squads") {
		return ref("Squad"), "201"
	}
	return ref("Squad"), "200"
}
