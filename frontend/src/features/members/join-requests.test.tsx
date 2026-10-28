import { screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { useJoinRequests } from '@/features/members/api'
import { JoinRequests } from '@/features/members/join-requests'
import { renderWithProviders } from '@/test/render'
import { server } from '@/test/msw-server'

function Requests() {
  const requests = useJoinRequests('tenant')
  return <JoinRequests tenantId="tenant" requests={requests} linkFilter={undefined} />
}

describe('Approval target confirmation', () => {
  it('shows frozen targets and forbids approving an archived target', async () => {
    server.use(
      http.get('/api/v1/tenants/tenant/join-requests', () =>
        HttpResponse.json({
          items: [
            {
              id: 'request',
              userId: 'person',
              displayName: 'Applicant',
              linkId: 'link',
              createdAt: '2026-10-28T00:00:00Z',
              version: 1,
              status: 'pending',
              targetSquadIds: ['squad'],
              targetSquads: [{ id: 'squad', name: 'Frozen team', archived: true }],
            },
          ],
          nextCursor: '',
        }),
      ),
    )
    renderWithProviders(<Requests />)
    expect(await screen.findByText(/Frozen team/)).toHaveTextContent('已归档')
    expect(screen.getByRole('button', { name: '批准' })).toBeDisabled()
    expect(screen.getByRole('button', { name: '拒绝' })).toBeEnabled()
  })
})
