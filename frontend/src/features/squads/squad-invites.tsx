import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { CreateLinkDialog } from '@/features/members/create-link-dialog'
import { HuaweiAdd } from '@/features/members/directory-add'
import { useInvitations, useJoinRequests } from '@/features/members/api'
import { useLoginProviders } from '@/features/auth/providers'
import { linkState, useJoinClock } from '@/features/members/browser'

/** Mounts only for administrators; outside newcomers keep the ordinary admission boundary. */
export function SquadInvites({ tenantId, squadId }: { tenantId: string; squadId: string }) {
  const { data: providers } = useLoginProviders()
  if (!providers) return null
  if (providers.includes('huawei-idaas'))
    return <HuaweiAdd tenantId={tenantId} targetSquadIds={[squadId]} />
  return <PublicInvites tenantId={tenantId} squadId={squadId} />
}

function PublicInvites({ tenantId, squadId }: { tenantId: string; squadId: string }) {
  const { t } = useTranslation()
  const [kind, setKind] = useState<'invite' | 'apply'>()
  const invitations = useInvitations(tenantId)
  const requests = useJoinRequests(tenantId)
  const now = useJoinClock()
  const pendingInvites = invitations.data?.items.filter(
    (row) => row.targetSquadIds?.includes(squadId) && linkState(row, now) === 'active',
  )
  const pendingRequests = requests.data?.items.filter(
    (row) => row.targetSquadIds?.includes(squadId) && row.status === 'pending',
  )
  return (
    <section className="space-y-3 rounded-lg border p-4">
      <h2 className="font-semibold">{t('squad.invite')}</h2>
      <div className="flex gap-2">
        <Button onClick={() => setKind('invite')}>{t('entry.invite')}</Button>
        <Button variant="outline" onClick={() => setKind('apply')}>
          {t('entry.apply')}
        </Button>
      </div>
      {kind && (
        <CreateLinkDialog
          tenantId={tenantId}
          kind={kind}
          targetSquadIds={[squadId]}
          onClose={() => setKind(undefined)}
        />
      )}
      <h3>{t('squad.pending')}</h3>
      {(invitations.isPending || requests.isPending) && <p role="status">{t('squad.loading')}</p>}
      {(invitations.isError || requests.isError) && <p role="alert">{t('squad.failed')}</p>}
      <ul>
        {pendingInvites?.map((row) => (
          <li key={row.id}>
            {t('squad.invitation')} · {row.note} · {row.expiresAt}
          </li>
        ))}
      </ul>
      {pendingRequests && <p>{t('squad.pendingRequests', { count: pendingRequests.length })}</p>}
    </section>
  )
}
