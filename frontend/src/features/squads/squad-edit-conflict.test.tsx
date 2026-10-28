import { screen, waitFor } from '@testing-library/react'
import { useQueryClient } from '@tanstack/react-query'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { SquadDetailPage } from '@/features/squads/squad-detail-page'
import { installCloudSpaceHandlers } from '@/test/cloud-handlers'
import { renderAtRoute } from '@/test/render'
import { server } from '@/test/msw-server'

function RemoteRefresh({ update }: { update: () => void }) {
  const client = useQueryClient()
  return (
    <button
      onClick={() => {
        update()
        void client.invalidateQueries()
      }}
    >
      Remote edit
    </button>
  )
}

describe('Squad edit conflict', () => {
  it('keeps the draft version when a live refresh changes the displayed squad', async () => {
    installCloudSpaceHandlers('member')
    let version = 1
    let submitted: unknown
    server.use(
      http.get('/api/v1/tenants/:tid/squads/:id', () =>
        HttpResponse.json({
          id: 'squad',
          name: version === 1 ? 'Original' : 'Remote',
          description: '',
          instructions: '',
          version,
          archivedAt: null,
          canMaintain: true,
          canInvite: false,
        }),
      ),
      http.get('/api/v1/tenants/:tid/squads/:id/members', () =>
        HttpResponse.json({ items: [], nextCursor: '' }),
      ),
      http.get('/api/v1/tenants/:tid/squad-agents', () =>
        HttpResponse.json({ items: [], nextCursor: '' }),
      ),
      http.get('/api/v1/tenants/:tid/members', () =>
        HttpResponse.json({ items: [], nextCursor: '' }),
      ),
      http.patch('/api/v1/tenants/:tid/squads/:id', async ({ request }) => {
        submitted = await request.json()
        return HttpResponse.json({ code: 'version_conflict' }, { status: 409 })
      }),
    )
    renderAtRoute(
      '/w/:workspaceSlug/squads/:squadId',
      <>
        <RemoteRefresh
          update={() => {
            version = 2
          }}
        />
        <SquadDetailPage slug="cloud-dev" />
      </>,
      '/w/cloud-dev/squads/squad',
    )
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: '编辑小队' }))
    await user.clear(screen.getByLabelText('小队名称'))
    await user.type(screen.getByLabelText('小队名称'), 'Draft')
    await user.click(screen.getByRole('button', { name: 'Remote edit' }))
    expect(await screen.findByRole('heading', { name: 'Remote' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '保存' }))
    await waitFor(() =>
      expect(submitted).toEqual({ name: 'Draft', description: '', instructions: '', version: 1 }),
    )
    expect(await screen.findByRole('alert')).toHaveTextContent('操作失败')
  })
})
