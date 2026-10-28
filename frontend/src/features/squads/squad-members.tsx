import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Squad, SquadMember } from '@/api/generated.schemas'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useSquadCommand, useSquadMembers } from '@/features/squads/api'

/** Displays current grants separately from retained history, with explicit maintenance edits. */
export function SquadMembers({ tenantId, squad }: { tenantId: string; squad: Squad }) {
  const { t } = useTranslation()
  const members = useSquadMembers(tenantId, squad.id)
  return (
    <section className="space-y-3">
      <h2 className="font-semibold">{t('squad.members')}</h2>
      {members.isPending && <p role="status">{t('squad.loading')}</p>}
      {members.isError && (
        <p role="alert">
          {t('squad.failed')}{' '}
          <Button onClick={() => void members.refetch()}>{t('squad.retry')}</Button>
        </p>
      )}
      {members.data?.items
        .filter((member) => !member.endedAt)
        .map((member) => (
          <MemberRow
            key={`${member.id}:${member.version}`}
            tenantId={tenantId}
            squad={squad}
            member={member}
          />
        ))}
      <details>
        <summary>{t('squad.history')}</summary>
        <ul>
          {members.data?.items
            .filter((member) => member.endedAt)
            .map((member) => (
              <li key={member.id} className="py-2 text-sm">
                {member.displayName} · {member.kind} · {member.sourceKind} · {member.endedAt} ·{' '}
                {member.endedReason}
              </li>
            ))}
        </ul>
      </details>
    </section>
  )
}

function MemberRow({
  tenantId,
  squad,
  member,
}: {
  tenantId: string
  squad: Squad
  member: SquadMember
}) {
  const { t } = useTranslation()
  const command = useSquadCommand(tenantId, squad.id, squad.version)
  const [responsibility, setResponsibility] = useState(member.responsibility)
  const [maintainer, setMaintainer] = useState(member.maintainer)
  const [clearLeader, setClearLeader] = useState(false)
  const editable = squad.canMaintain && !squad.archivedAt
  return (
    <article className="space-y-2 rounded-md border p-3">
      <p className="font-medium">
        {member.displayName} · {member.kind}
        {member.maintainer ? ` · ${t('squad.maintainer')}` : ''}
      </p>
      {editable ? (
        <div className="flex flex-wrap items-center gap-2">
          <Input
            aria-label={`${t('squad.responsibility')} ${member.displayName}`}
            value={responsibility}
            maxLength={2000}
            onChange={(e) => setResponsibility(e.target.value)}
          />
          {member.kind === 'user' && (
            <label>
              <input
                type="checkbox"
                checked={maintainer}
                onChange={(e) => setMaintainer(e.target.checked)}
              />{' '}
              {t('squad.maintainer')}
            </label>
          )}
          <Button
            disabled={command.isPending}
            onClick={() =>
              command.mutate({ action: 'member', id: member.id, responsibility, maintainer })
            }
          >
            {t('squad.save')}
          </Button>
          {squad.leaderAgentId && squad.leaderAgentId === member.agentId && (
            <label>
              <input
                type="checkbox"
                checked={clearLeader}
                onChange={(e) => setClearLeader(e.target.checked)}
              />{' '}
              {t('squad.clearLeader')}
            </label>
          )}
          <Button
            variant="outline"
            disabled={command.isPending}
            onClick={() => command.mutate({ action: 'remove', id: member.id, clearLeader })}
          >
            {t('squad.remove')}
          </Button>
        </div>
      ) : (
        <p className="text-sm">{member.responsibility}</p>
      )}
      {command.isError && <p role="alert">{t('squad.failure')}</p>}
    </article>
  )
}
