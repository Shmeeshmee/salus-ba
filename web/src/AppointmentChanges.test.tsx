import { cleanup, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { AppointmentChanges } from './AppointmentChanges'
import { newApiClient } from './api/client'
import type { Language } from './i18n/languages'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, fakeFetch, type Route } from './test/fakeFetch'

afterEach(cleanup)

const id = 'e1000000-0000-4000-8000-000000000001'
const changesGet = `GET /appointments/${id}/changes`

const changes = {
  changes: [
    { at: '2026-11-02T09:00:00Z', kind: 'confirmed', via: 'link' },
    {
      at: '2026-11-03T10:15:00Z',
      kind: 'rescheduled',
      via: 'app',
      from_start: '2026-11-09T08:00:00Z',
      from_end: '2026-11-09T08:30:00Z',
      to_start: '2026-11-10T13:00:00Z',
      to_end: '2026-11-10T13:30:00Z',
    },
    { at: '2026-11-04T11:00:00Z', kind: 'status_set', via: 'staff', from_status: 'booked', to_status: 'cancelled' },
  ],
}

function renderChanges(route: Route, language: Language = 'en') {
  const fake = fakeFetch({ [changesGet]: route })
  renderInLanguage(<AppointmentChanges api={newApiClient(fake.fetch)} id={id} zone="Europe/Sarajevo" />, language)
  return fake
}

describe('AppointmentChanges', () => {
  it('lists the changes oldest first, each with how and when it was made, in the clinic zone', async () => {
    renderChanges(() => Response.json(changes))

    const items = await screen.findAllByRole('listitem')
    expect(items).toHaveLength(3)
    expect(items[0].textContent).toContain('Confirmed by the patient')
    expect(items[0].textContent).toContain('Through the reminder link')
    expect(items[0].textContent).toContain('10:00')
    expect(items[1].textContent).toContain('Moved from')
    expect(items[1].textContent).toContain('09:00')
    expect(items[1].textContent).toContain('14:00')
    expect(items[1].textContent).toContain('In the app')
    expect(items[2].textContent).toContain('Cancelled')
    expect(items[2].textContent).toContain('By the clinic')
  })

  it('says when there is no change yet', async () => {
    renderChanges(() => Response.json({ changes: [] }))

    expect(await screen.findByText('No changes yet.')).toBeTruthy()
  })

  it('says when the changes cannot be loaded', async () => {
    renderChanges(() => apiError(500, 'internal'))

    expect(await screen.findByRole('alert')).toBeTruthy()
  })

  it('describes a correction of the outcome by the clinic (SAL-36)', async () => {
    renderChanges(() =>
      Response.json({
        changes: [
          { at: '2026-11-04T11:00:00Z', kind: 'status_set', via: 'system', from_status: 'booked', to_status: 'noshow' },
          { at: '2026-11-05T08:00:00Z', kind: 'status_corrected', via: 'staff', from_status: 'noshow', to_status: 'fulfilled' },
          { at: '2026-11-06T08:00:00Z', kind: 'status_corrected', via: 'staff', from_status: 'fulfilled', to_status: 'noshow' },
        ],
      }),
    )

    const items = await screen.findAllByRole('listitem')
    expect(items.map((li) => li.querySelector('span')!.textContent)).toEqual(['Marked no-show', 'Corrected to fulfilled', 'Corrected to no-show'])
    expect(items[1].textContent).toContain('By the clinic')
  })

  it('leaves out a kind it does not know rather than guess', async () => {
    renderChanges(() => Response.json({ changes: [{ at: '2026-11-04T11:00:00Z', kind: 'merged', via: 'staff' }] }))

    expect(await screen.findByText('No changes yet.')).toBeTruthy()
  })

  it('writes the changes in the language', async () => {
    renderChanges(() => Response.json(changes), 'hr')

    expect(await screen.findByText('Pacijent je potvrdio')).toBeTruthy()
    expect(screen.getByText('Otkazan')).toBeTruthy()
  })
})
