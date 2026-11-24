import { act, cleanup, fireEvent, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ClinicExport } from './ClinicExport'
import { newApiClient } from './api/client'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from './test/fakeFetch'

afterEach(() => {
  cleanup()
  vi.useRealTimers()
})

const id = '44444444-4444-4444-8444-444444444444'
const queued = { id, status: 'queued', requestedAt: '2026-11-24T09:00:00Z' }
const ready = { ...queued, status: 'ready', expiresAt: '2026-12-01T09:00:00Z' }

function renderExport(routes: Record<string, Route>) {
  const fake = fakeFetch(routes)
  renderInLanguage(<ClinicExport api={newApiClient(fake.fetch)} zone="Europe/Sarajevo" pollMs={10} />, 'en')
  return fake
}

describe('ClinicExport', () => {
  it('asks for the export, waits while it is built, then offers the download until it expires', async () => {
    let reads = 0
    const fake = renderExport({
      'POST /clinic/export': () => Response.json(queued, { status: 202 }),
      [`GET /clinic/export/${id}`]: () => Response.json(reads++ === 0 ? queued : ready),
    })
    fireEvent.click(screen.getByRole('button', { name: 'Export the clinic’s data' }))
    expect(await screen.findByText('The archive is being built. This can take a few minutes.')).toBeTruthy()
    const link = await screen.findByRole('link', { name: 'Download the archive' })
    expect(link.getAttribute('href')).toBe(`${globalThis.location.origin}/clinic/export/${id}/archive`)
    expect(screen.getByText('Available until 1 Dec 2026, 10:00.')).toBeTruthy()
    expectCsrfOnWrites(fake.requests)
  })

  it('says when the build failed, and when the export has expired', async () => {
    renderExport({
      'POST /clinic/export': () => Response.json(queued, { status: 202 }),
      [`GET /clinic/export/${id}`]: () => Response.json({ ...queued, status: 'failed' }),
    })
    fireEvent.click(screen.getByRole('button', { name: 'Export the clinic’s data' }))
    expect(await screen.findByText('The archive could not be built. Ask for a new export.')).toBeTruthy()
    cleanup()
    renderExport({
      'POST /clinic/export': () => Response.json({ ...ready, status: 'expired' }, { status: 202 }),
    })
    fireEvent.click(screen.getByRole('button', { name: 'Export the clinic’s data' }))
    expect(await screen.findByText('This export has expired. Ask for a new one.')).toBeTruthy()
  })

  it('says when asking failed or is not allowed', async () => {
    renderExport({ 'POST /clinic/export': () => apiError(403, 'forbidden') })
    fireEvent.click(screen.getByRole('button', { name: 'Export the clinic’s data' }))
    expect(await screen.findByText('Only a clinic admin can export the clinic’s data.')).toBeTruthy()
    cleanup()
    renderExport({ 'POST /clinic/export': () => apiError(500, 'internal') })
    fireEvent.click(screen.getByRole('button', { name: 'Export the clinic’s data' }))
    expect(await screen.findByText('The export could not be started. Try again.')).toBeTruthy()
  })

  it('stops asking for the status once it leaves the page', async () => {
    let reads = 0
    renderExport({
      'POST /clinic/export': () => Response.json(queued, { status: 202 }),
      [`GET /clinic/export/${id}`]: () => {
        reads++
        return Response.json(queued)
      },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Export the clinic’s data' }))
    await screen.findByText('The archive is being built. This can take a few minutes.')
    cleanup()
    const seen = reads
    await act(() => new Promise((r) => setTimeout(r, 50)))
    expect(reads).toBe(seen)
  })
})
