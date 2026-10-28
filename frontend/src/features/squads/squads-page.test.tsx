import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '@/test/render'
import { SquadsPage } from './squads-page'
import { installCloudSpaceHandlers } from '@/test/cloud-handlers'
import { server } from '@/test/msw-server'

describe('Real squad list', () => {
  it('renders an empty server list and creates a real squad without an execution action', async () => {
    installCloudSpaceHandlers('member')
    let created = false
    server.use(
      http.get('/api/v1/tenants/:tid/squads', () =>
        HttpResponse.json({
          items: created
            ? [{ id: 'real', name: 'Design', description: '', humanCount: 1, agentCount: 0 }]
            : [],
          nextCursor: '',
        }),
      ),
      http.post('/api/v1/tenants/:tid/squads', async ({ request }) => {
        expect(await request.json()).toEqual({ name: 'Design', description: '', instructions: '' })
        created = true
        return HttpResponse.json({ id: 'real' }, { status: 201 })
      }),
    )
    renderWithProviders(<SquadsPage slug="cloud-dev" />, { slug: 'cloud-dev' })
    expect(await screen.findByText(/还没有小队/)).toBeInTheDocument()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: '创建小队' }))
    await user.type(screen.getByLabelText('小队名称'), 'Design')
    await user.click(screen.getByRole('button', { name: '保存' }))
    expect(await screen.findByRole('link', { name: 'Design' })).toHaveAttribute(
      'href',
      '/w/cloud-dev/squads/real',
    )
    expect(screen.getByText('1 位成员 · 0 个智能体')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /执行|Run/ })).not.toBeInTheDocument()
  })

  it('shows failures rather than falling back to demo squads', async () => {
    installCloudSpaceHandlers('member')
    server.use(
      http.get('/api/v1/tenants/:tid/squads', () =>
        HttpResponse.json({ code: 'unavailable' }, { status: 503 }),
      ),
    )
    renderWithProviders(<SquadsPage slug="cloud-dev" />, { slug: 'cloud-dev' })
    expect(await screen.findByRole('alert')).toHaveTextContent('无法读取小队')
    expect(screen.queryByText('平台组')).not.toBeInTheDocument()
  })
})
