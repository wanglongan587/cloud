import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Squad } from '@/api/generated.schemas'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useMembers } from '@/features/members/api'
import { useSquadAgents, useSquadCommand } from '@/features/squads/api'

/** Selects existing typed identities; this action never grants workspace admission. */
export function SquadAdd({ tenantId, squad }: { tenantId: string; squad: Squad }) {
  const { t } = useTranslation()
  const users = useMembers(tenantId)
  const agents = useSquadAgents(tenantId)
  const command = useSquadCommand(tenantId, squad.id, squad.version)
  const [kind, setKind] = useState<'user' | 'agent'>('user')
  const [memberId, setMemberId] = useState('')
  const [responsibility, setResponsibility] = useState('')
  const options =
    kind === 'user'
      ? users.data?.items
          .filter((user) => user.status === 'active')
          .map((user) => ({ id: user.userId, name: user.displayName }))
      : agents.data?.items.map((agent) => ({ id: agent.id, name: agent.displayName }))
  return (
    <section className="space-y-3 rounded-lg border p-4">
      <div className="flex flex-wrap gap-2">
        <Button
          variant={kind === 'user' ? 'default' : 'outline'}
          onClick={() => {
            setKind('user')
            setMemberId('')
          }}
        >
          {t('squad.addUser')}
        </Button>
        <Button
          variant={kind === 'agent' ? 'default' : 'outline'}
          onClick={() => {
            setKind('agent')
            setMemberId('')
          }}
        >
          {t('squad.addAgent')}
        </Button>
      </div>
      {(users.isError || agents.isError) && <p role="alert">{t('squad.failed')}</p>}
      <select
        aria-label={t(kind === 'user' ? 'squad.addUser' : 'squad.addAgent')}
        className="w-full rounded border p-2"
        value={memberId}
        onChange={(e) => setMemberId(e.target.value)}
      >
        <option value="">{t('squad.select')}</option>
        {options?.map((item) => (
          <option key={item.id} value={item.id}>
            {item.name}
          </option>
        ))}
      </select>
      <Input
        aria-label={t('squad.responsibility')}
        value={responsibility}
        maxLength={2000}
        onChange={(e) => setResponsibility(e.target.value)}
      />
      <Button
        disabled={!memberId || command.isPending}
        onClick={() => command.mutate({ action: 'add', kind, memberId, responsibility })}
      >
        {t('squad.add')}
      </Button>
      {command.isError && <p role="alert">{t('squad.failure')}</p>}
      <LeaderChoice tenantId={tenantId} squad={squad} />
    </section>
  )
}

function LeaderChoice({ tenantId, squad }: { tenantId: string; squad: Squad }) {
  const { t } = useTranslation()
  const agents = useSquadAgents(tenantId)
  const command = useSquadCommand(tenantId, squad.id, squad.version)
  const [agentId, setAgentId] = useState(squad.leaderAgentId ?? '')
  return (
    <div className="space-y-2 border-t pt-3">
      <label className="block">
        {t('squad.leader')}
        <select
          className="ml-2 rounded border p-2"
          value={agentId}
          onChange={(e) => setAgentId(e.target.value)}
        >
          <option value="">{t('squad.noLeader')}</option>
          {agents.data?.items.map((agent) => (
            <option value={agent.id} key={agent.id}>
              {agent.displayName}
            </option>
          ))}
        </select>
      </label>
      <Button
        disabled={command.isPending}
        onClick={() => command.mutate({ action: 'leader', agentId })}
      >
        {t('squad.changeLeader')}
      </Button>
      {command.isError && <p role="alert">{t('squad.failure')}</p>}
    </div>
  )
}
