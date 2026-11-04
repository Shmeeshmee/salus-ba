import { describe, expect, it } from 'vitest'
import { fakeFetch } from '../test/fakeFetch'
import { newApiClient } from './client'

describe('newApiClient', () => {
  it('sends X-Salus-CSRF: 1 on writes and not on reads', async () => {
    const { fetch, requests } = fakeFetch({
      'GET /auth/me': () =>
        Response.json({ id: 'a1000000-0000-4000-8000-000000000001', email: 'x@example.test', role: 'admin', mfaEnrolled: true }),
      'POST /auth/logout': () => new Response(null, { status: 204 }),
    })
    const api = newApiClient(fetch)

    await api.GET('/auth/me')
    await api.POST('/auth/logout')

    expect(requests[0].headers.get('X-Salus-CSRF')).toBeNull()
    expect(requests[1].headers.get('X-Salus-CSRF')).toBe('1')
  })
})
