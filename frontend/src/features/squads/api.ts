import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  getApiV1TenantsTidSquads,
  getApiV1TenantsTidSquadsSquadId,
  getApiV1TenantsTidSquadsSquadIdMembers,
  getApiV1TenantsTidSquadAgents,
  postApiV1TenantsTidSquads,
  patchApiV1TenantsTidSquadsSquadId,
  postApiV1TenantsTidSquadsSquadIdArchive,
  postApiV1TenantsTidSquadsSquadIdLeave,
  postApiV1TenantsTidSquadsSquadIdLeader,
  postApiV1TenantsTidSquadsSquadIdMembers,
  patchApiV1TenantsTidSquadsSquadIdMembersSquadMemberId,
  deleteApiV1TenantsTidSquadsSquadIdMembersSquadMemberId,
} from '@/api/squads/squads'
import { readAllPages } from '@/lib/pagination'

const key = (tenantId: string) => [`/api/v1/tenants/${tenantId}/squads`]

/** Reads active squads; archival history remains available by its stable URL. */
export function useSquads(tenantId: string) {
  return useQuery({
    queryKey: key(tenantId),
    queryFn: ({ signal }) =>
      readAllPages((after) =>
        getApiV1TenantsTidSquads(tenantId, after ? { after } : undefined, undefined, signal),
      ),
  })
}

/** Loads current authority and configuration, never an executable Team target. */
export function useSquad(tenantId: string, id: string) {
  return useQuery({
    queryKey: [...key(tenantId), id],
    queryFn: ({ signal }) => getApiV1TenantsTidSquadsSquadId(tenantId, id, undefined, signal),
  })
}

/** Retains ended relationships so a new grant cannot erase membership history. */
export function useSquadMembers(tenantId: string, id: string) {
  return useQuery({
    queryKey: [...key(tenantId), id, 'members'],
    queryFn: ({ signal }) =>
      readAllPages((after) =>
        getApiV1TenantsTidSquadsSquadIdMembers(
          tenantId,
          id,
          after ? { after } : undefined,
          undefined,
          signal,
        ),
      ),
  })
}

/** Lists active Space Agents, not marketplace or demo personas. */
export function useSquadAgents(tenantId: string) {
  return useQuery({
    queryKey: [`/api/v1/tenants/${tenantId}/squad-agents`],
    queryFn: ({ signal }) =>
      readAllPages((after) =>
        getApiV1TenantsTidSquadAgents(tenantId, after ? { after } : undefined, undefined, signal),
      ),
  })
}

/** Creates a squad and the creator's first maintainer relationship atomically. */
export function useCreateSquad(tenantId: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (input: { name: string; description: string; instructions: string }) =>
      postApiV1TenantsTidSquads(tenantId, input),
    onSuccess: () => client.invalidateQueries({ queryKey: key(tenantId) }),
  })
}

type Command =
  | { action: 'edit'; name: string; description: string; instructions: string }
  | { action: 'archive' | 'leave' }
  | { action: 'leader'; agentId: string }
  | { action: 'add'; kind: 'user' | 'agent'; memberId: string; responsibility: string }
  | { action: 'member'; id: string; responsibility: string; maintainer: boolean }
  | { action: 'remove'; id: string; clearLeader: boolean }

/** Carries the displayed squad version; a conflict refetches current authority. */
export function useSquadCommand(tenantId: string, id: string, version: number) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (command: Command) => {
      switch (command.action) {
        case 'edit':
          return patchApiV1TenantsTidSquadsSquadId(tenantId, id, {
            name: command.name,
            description: command.description,
            instructions: command.instructions,
            version,
          })
        case 'archive':
          return postApiV1TenantsTidSquadsSquadIdArchive(tenantId, id, { version })
        case 'leave':
          return postApiV1TenantsTidSquadsSquadIdLeave(tenantId, id, { version })
        case 'leader':
          return postApiV1TenantsTidSquadsSquadIdLeader(tenantId, id, {
            agentId: command.agentId,
            version,
          })
        case 'add':
          return postApiV1TenantsTidSquadsSquadIdMembers(tenantId, id, {
            kind: command.kind,
            memberId: command.memberId,
            responsibility: command.responsibility,
            version,
          })
        case 'member':
          return patchApiV1TenantsTidSquadsSquadIdMembersSquadMemberId(tenantId, id, command.id, {
            responsibility: command.responsibility,
            maintainer: command.maintainer,
            version,
          })
        case 'remove':
          return deleteApiV1TenantsTidSquadsSquadIdMembersSquadMemberId(tenantId, id, command.id, {
            clearLeader: command.clearLeader,
            version,
          })
        default:
          throw new Error('Unsupported squad command')
      }
    },
    onSettled: () => client.invalidateQueries({ queryKey: key(tenantId) }),
  })
}
