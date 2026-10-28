import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { renderAtRoute } from '@/test/render'
import { server } from '@/test/msw-server'
import { installCloudSpaceHandlers } from '@/test/cloud-handlers'
import { SquadDetailPage } from './squad-detail-page'

describe('Squad detail authority', () => {
  it.each(['squad-platform', 'unknown-squad'])(
    'does not resolve %s from demo data',
    async (squadId) => {
      installCloudSpaceHandlers('member')
      server.use(
        http.get('/api/v1/tenants/:tid/squads/:id', () =>
          HttpResponse.json({ code: 'not_found' }, { status: 404 }),
        ),
      )
      renderAtRoute(
        '/w/:workspaceSlug/squads/:squadId',
        <SquadDetailPage slug="cloud-dev" />,
        `/w/cloud-dev/squads/${squadId}`,
      )
      expect(await screen.findByRole('alert')).toHaveTextContent('无法读取小队')
      expect(screen.queryByText('平台组')).not.toBeInTheDocument()
    },
  )
  it('retains archived history and hides all mutation controls', async () => {
    installCloudSpaceHandlers('member')
    server.use(
      http.get('/api/v1/tenants/:tid/squads/:id', () =>
        HttpResponse.json({
          id: 'archived',
          name: 'Past team',
          description: '',
          instructions: '',
          version: 3,
          archivedAt: '2026-10-28',
          canMaintain: true,
          canInvite: true,
        }),
      ),
      http.get('/api/v1/tenants/:tid/squads/:id/members', () =>
        HttpResponse.json({
          items: [
            {
              id: 'old',
              displayName: 'Former member',
              kind: 'user',
              sourceKind: 'manual',
              endedAt: '2026-10-28',
              endedReason: 'squad_archived',
            },
          ],
          nextCursor: '',
        }),
      ),
    )
    renderAtRoute(
      '/w/:workspaceSlug/squads/:squadId',
      <SquadDetailPage slug="cloud-dev" />,
      '/w/cloud-dev/squads/archived',
    )
    expect(await screen.findByRole('heading', { name: 'Past team' })).toBeInTheDocument()
    expect(await screen.findByText(/Former member/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '编辑小队' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '邀请新人' })).not.toBeInTheDocument()
  })
})
