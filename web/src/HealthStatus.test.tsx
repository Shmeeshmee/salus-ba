import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { HealthStatus } from './HealthStatus'
import { newApiClient } from './api/client'

afterEach(cleanup)

function clientAnswering(respond: (url: string) => Response | Promise<Response>) {
  return newApiClient((input: Request) => Promise.resolve(respond(input.url)))
}

describe('HealthStatus', () => {
  it('shows that the API is checked while the request runs', () => {
    render(<HealthStatus client={clientAnswering(() => new Promise<Response>(() => {}))} />)

    expect(screen.getByText('API status: checking')).toBeTruthy()
  })

  it('calls /healthz and shows ok', async () => {
    let called = ''
    const client = clientAnswering((url) => {
      called = url
      return Response.json({ status: 'ok' })
    })

    render(<HealthStatus client={client} />)

    expect(await screen.findByText('API status: ok')).toBeTruthy()
    expect(new URL(called).pathname).toBe('/healthz')
  })

  it('shows unavailable when the API answers an error', async () => {
    render(<HealthStatus client={clientAnswering(() => new Response(null, { status: 503 }))} />)

    expect(await screen.findByText('API status: unavailable')).toBeTruthy()
  })

  it('shows unavailable when the API cannot be reached', async () => {
    render(<HealthStatus client={clientAnswering(() => Promise.reject(new TypeError('network')))} />)

    expect(await screen.findByText('API status: unavailable')).toBeTruthy()
  })
})
