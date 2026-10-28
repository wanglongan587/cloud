import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import type { Squad } from '@/api/generated.schemas'
import { Button } from '@/components/ui/button'
import { PageHeader } from '@/components/layout/page-header'
import { useCurrentSpace } from '@/features/spaces/current-space'
import { useSquad, useSquadCommand } from '@/features/squads/api'
import { SquadEditor } from '@/features/squads/squad-editor'
import { SquadMembers } from '@/features/squads/squad-members'
import { SquadAdd } from '@/features/squads/squad-add'
import { SquadInvites } from '@/features/squads/squad-invites'
import { squadFailureMessage } from '@/features/squads/squad-feedback'

/** Resolves a stable squad URL against current tenant authority, including archived history. */
export function SquadDetailPage({ slug }: { slug: string }) {
  const { tenantId, space } = useCurrentSpace()
  const { squadId = '' } = useParams<{ squadId: string }>()
  if (!tenantId || space?.slug !== slug) return null
  return <SquadDetail key={`${tenantId}:${squadId}`} tenantId={tenantId} id={squadId} slug={slug} />
}

function SquadDetail({ tenantId, id, slug }: { tenantId: string; id: string; slug: string }) {
  const { t } = useTranslation()
  const detail = useSquad(tenantId, id)
  return (
    <main className="mx-auto max-w-5xl space-y-5 p-4 md:p-6">
      <PageHeader
        title={t('squad.title')}
        actions={<Link to={`/w/${slug}/squads`}>{t('squad.back')}</Link>}
      />
      {detail.isPending && <p role="status">{t('squad.loading')}</p>}
      {detail.isError && (
        <p role="alert">
          {t(squadFailureMessage(detail.error))}{' '}
          <Button onClick={() => void detail.refetch()}>{t('squad.retry')}</Button>
        </p>
      )}
      {detail.data && <SquadContent tenantId={tenantId} squad={detail.data} />}
    </main>
  )
}

function SquadContent({ tenantId, squad }: { tenantId: string; squad: Squad }) {
  const { t } = useTranslation()
  const command = useSquadCommand(tenantId, squad.id, squad.version)
  const [editing, setEditing] = useState<Squad>()
  const [archiving, setArchiving] = useState(false)
  return (
    <>
      <h1 className="text-xl font-semibold">{squad.name}</h1>
      <p>{squad.description}</p>
      <p className="whitespace-pre-wrap text-sm">{squad.instructions}</p>
      <p className="rounded-md bg-muted p-3 text-sm">{t('squad.noExecution')}</p>
      {squad.archivedAt ? (
        <p role="status">{t('squad.archived')}</p>
      ) : (
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" onClick={() => command.mutate({ action: 'leave' })}>
            {t('squad.leave')}
          </Button>
          {squad.canMaintain && (
            <>
              <Button onClick={() => setEditing(editing ? undefined : squad)}>
                {t('squad.edit')}
              </Button>
              <Button variant="outline" onClick={() => setArchiving(true)}>
                {t('squad.archive')}
              </Button>
            </>
          )}
        </div>
      )}
      {archiving && !squad.archivedAt && (
        <Button disabled={command.isPending} onClick={() => command.mutate({ action: 'archive' })}>
          {t('squad.confirmArchive')}
        </Button>
      )}
      {command.isError && <p role="alert">{t('squad.failure')}</p>}
      {editing && squad.canMaintain && !squad.archivedAt && (
        <SquadEdit tenantId={tenantId} snapshot={editing} onClose={() => setEditing(undefined)} />
      )}
      <SquadMembers tenantId={tenantId} squad={squad} />
      {squad.canMaintain && !squad.archivedAt && (
        <SquadAdd key={`${squad.id}:${squad.version}`} tenantId={tenantId} squad={squad} />
      )}
      {squad.canInvite && !squad.archivedAt && (
        <SquadInvites tenantId={tenantId} squadId={squad.id} />
      )}
    </>
  )
}

function SquadEdit({
  tenantId,
  snapshot,
  onClose,
}: {
  tenantId: string
  snapshot: Squad
  onClose: () => void
}) {
  const { t } = useTranslation()
  const command = useSquadCommand(tenantId, snapshot.id, snapshot.version)
  return (
    <>
      <SquadEditor
        initial={snapshot}
        busy={command.isPending}
        onSave={(value) => command.mutate({ action: 'edit', ...value }, { onSuccess: onClose })}
        onCancel={onClose}
      />
      {command.isError && <p role="alert">{t('squad.failure')}</p>}
    </>
  )
}
