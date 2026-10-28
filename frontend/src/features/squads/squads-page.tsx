import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { useCurrentSpace } from '@/features/spaces/current-space'
import { useCreateSquad, useSquads } from '@/features/squads/api'
import { SquadEditor } from '@/features/squads/squad-editor'

/** Projects only the current workspace's real squad records. */
export function SquadsPage({ slug }: { slug: string }) {
  const { tenantId, space } = useCurrentSpace()
  if (!tenantId || space?.slug !== slug) return null
  return <SquadList key={tenantId} tenantId={tenantId} slug={slug} />
}

function SquadList({ tenantId, slug }: { tenantId: string; slug: string }) {
  const { t } = useTranslation()
  const squads = useSquads(tenantId)
  const create = useCreateSquad(tenantId)
  const [editing, setEditing] = useState(false)
  return (
    <main className="mx-auto max-w-5xl space-y-5 p-4 md:p-6">
      <div className="flex items-center justify-between gap-3">
        <h1 className="text-xl font-semibold">{t('squad.title')}</h1>
        <Button onClick={() => setEditing(true)}>{t('squad.create')}</Button>
      </div>
      <p className="text-sm text-muted-foreground">{t('squad.noExecution')}</p>
      {editing && (
        <SquadEditor
          busy={create.isPending}
          onSave={(value) => create.mutate(value, { onSuccess: () => setEditing(false) })}
          onCancel={() => setEditing(false)}
        />
      )}
      {create.isError && <p role="alert">{t('squad.failure')}</p>}
      {squads.isPending && <p role="status">{t('squad.loading')}</p>}
      {squads.isError && (
        <p role="alert">
          {t('squad.failed')}{' '}
          <Button onClick={() => void squads.refetch()}>{t('squad.retry')}</Button>
        </p>
      )}
      {squads.data?.items.length === 0 && <p>{t('squad.empty')}</p>}
      <ul className="grid gap-4 sm:grid-cols-2">
        {squads.data?.items.map((squad) => (
          <li className="rounded-lg border p-4" key={squad.id}>
            <Link className="font-medium underline" to={`/w/${slug}/squads/${squad.id}`}>
              {squad.name}
            </Link>
            <p className="my-2 text-sm">{squad.description}</p>
            <p className="text-sm text-muted-foreground">
              {t('squad.count', { humans: squad.humanCount ?? 0, agents: squad.agentCount ?? 0 })}
            </p>
          </li>
        ))}
      </ul>
    </main>
  )
}
