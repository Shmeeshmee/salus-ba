import { cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { Desk } from './Desk'
import { newApiClient, type Account } from './api/client'
import { newFhirClient } from './api/fhir'
import type { Language } from './i18n/languages'
import { renderInLanguage } from './i18n/renderInLanguage'
import { anaVisit, boVisit, cleoVisit, deskDay, exam, freeInPerson, now, scheduleRoute, services, slotsRoute } from './test/desk'
import { apiError, expectCsrfOnWrites, fakeFetch, paths, type Route } from './test/fakeFetch'
import { admin, clinicRoute, staffRoute } from './test/fixtures'

afterEach(cleanup)

const reception: Account = { ...admin, id: 'a1000000-0000-4000-8000-000000000009', email: 'desk@example.test', role: 'reception' }

function renderDesk(routes: Record<string, Route>, account: Account = admin, language: Language = 'en') {
  const fake = fakeFetch({
    ...clinicRoute,
    ...staffRoute,
    'GET /clinic/desk/today': () => Response.json(deskDay),
    'GET /clinic/services': () => Response.json(services),
    ...routes,
  })
  renderInLanguage(<Desk api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} account={account} now={now} />, language)
  return fake
}

function rowOf(name: string) {
  return screen.getByText(name).closest('li')!
}

describe('Desk', () => {
  it("lists today's visits by practitioner and time, with the patient, visit type, service and status", async () => {
    const fake = renderDesk({}, reception)
    const cy = within(await screen.findByRole('list', { name: 'Cy Example' }))
    const rows = cy.getAllByRole('listitem').map((r) => r.textContent)
    expect(rows).toHaveLength(2)
    expect(rows[0]).toContain('09:00')
    expect(rows[0]).toContain('Ana Example')
    expect(rows[0]).toContain('2 March 1980')
    expect(rows[0]).toContain('+387 61 000 001') // grouped as people write it (wave 4 walk, finding 6)
    expect(rows[0]).toContain('At the clinic')
    expect(rows[0]).toContain('Placeholder examination')
    expect(rows[0]).toContain('Booked')
    expect(rows[1]).toContain('10:00')
    expect(rows[1]).toContain('Bo Sample')
    expect(rows[1]).toContain('Video')
    const dee = within(screen.getByRole('list', { name: 'Dee Sample' })).getAllByRole('listitem')
    expect(dee[0].textContent).toContain('Cleo Test')
    expect(dee[0].textContent).toContain('Arrived 09:25')
    expect(screen.getByRole('heading', { name: /Monday, 23 November 2026/ })).toBeTruthy()
    const asked = new URL(fake.requests.find((r) => r.url.includes('/clinic/desk/today'))!.url)
    expect(asked.search).toBe('')
  })

  it('offers "Arrived" only on a booked visit at the clinic', async () => {
    renderDesk({})
    await screen.findByText('Ana Example')
    expect(within(rowOf('Ana Example')).queryByRole('button', { name: 'Arrived' })).toBeTruthy()
    expect(within(rowOf('Bo Sample')).queryByRole('button', { name: 'Arrived' })).toBeNull()
    expect(within(rowOf('Cleo Test')).queryByRole('button', { name: 'Arrived' })).toBeNull()
  })

  it('checks a patient in and shows the visit as arrived', async () => {
    const fake = renderDesk({
      [`POST /clinic/appointments/${anaVisit.appointment.id}/arrive`]: () =>
        Response.json({ ...anaVisit.appointment, status: 'arrived', arrivedAt: '2026-11-23T08:20:00Z', versionId: 2 }),
    })
    await screen.findByText('Ana Example')
    fireEvent.click(within(rowOf('Ana Example')).getByRole('button', { name: 'Arrived' }))

    await waitFor(() => expect(rowOf('Ana Example').textContent).toContain('Arrived 09:20'))
    expect(within(rowOf('Ana Example')).queryByRole('button', { name: 'Arrived' })).toBeNull()
    expect(paths(fake.requests)).toContain(`POST /clinic/appointments/${anaVisit.appointment.id}/arrive`)
    expectCsrfOnWrites(fake.requests)
  })

  it.each([
    ['not_in_person', 409, 'A video visit is not checked in at the desk.'],
    ['appointment_closed', 409, 'This visit is no longer booked. Reload the page to see it as it is now.'],
    ['forbidden', 403, 'The check-in could not be saved. Try again.'],
  ])('says why a check-in was refused (%s)', async (code, status, text) => {
    renderDesk({ [`POST /clinic/appointments/${anaVisit.appointment.id}/arrive`]: () => apiError(status, code) })
    await screen.findByText('Ana Example')
    fireEvent.click(within(rowOf('Ana Example')).getByRole('button', { name: 'Arrived' }))

    expect(await within(rowOf('Ana Example')).findByText(text)).toBeTruthy()
    expect(rowOf('Ana Example').textContent).toContain('Booked')
  })

  it('issues the receipt from the row, starting from the booked service', async () => {
    let body: unknown
    renderDesk({
      'POST /clinic/receipts': async (r) => {
        body = await r.json()
        return Response.json({ id: 'r0000000-0000-4000-8000-000000000001', number: 12 }, { status: 201 })
      },
    })
    await screen.findByText('Ana Example')
    fireEvent.click(within(rowOf('Ana Example')).getByRole('button', { name: 'Issue receipt' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'New receipt' }))
    await waitFor(() => expect((dialog.getByLabelText('Service') as HTMLSelectElement).value).toBe(exam))
    fireEvent.click(dialog.getByRole('button', { name: 'Issue receipt' }))

    expect(await screen.findByText('Receipt no. 12 was issued.')).toBeTruthy()
    expect(body).toEqual({ kind: 'receipt', paymentMethod: 'cash', appointmentId: anaVisit.appointment.id, lines: [{ serviceId: exam, quantity: 1, discountMinor: 0 }] })
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it("shows a visit's receipts with their PDF, whoever issued them, and marks a reversed one", async () => {
    const day = {
      ...deskDay,
      visits: [{ ...anaVisit, receipts: [{ id: 'r0000000-0000-4000-8000-000000000007', number: 7, reversed: true }, { id: 'r0000000-0000-4000-8000-000000000008', number: 8, reversed: false }] }, boVisit, cleoVisit],
    }
    renderDesk({ 'GET /clinic/desk/today': () => Response.json(day) }, reception)
    await screen.findByText('Ana Example')
    const row = within(rowOf('Ana Example'))
    expect(row.getByText('Receipt no. 7')).toBeTruthy()
    expect(row.getByText('Reversed')).toBeTruthy()
    const pdfs = row.getAllByRole('link', { name: 'Open PDF' }).map((a) => a.getAttribute('href'))
    expect(pdfs).toEqual([expect.stringMatching(/\/clinic\/receipts\/r0000000-0000-4000-8000-000000000007\/pdf$/), expect.stringMatching(/\/clinic\/receipts\/r0000000-0000-4000-8000-000000000008\/pdf$/)])
    expect(row.queryByRole('button', { name: 'Issue receipt' })).toBeNull()
    expect(row.getByRole('button', { name: 'Issue another receipt' })).toBeTruthy()
    expect(within(rowOf('Cleo Test')).getByRole('button', { name: 'Issue receipt' })).toBeTruthy()
  })

  it('shows the receipt on the row once issued, and asks before a second one', async () => {
    renderDesk({ 'POST /clinic/receipts': () => Response.json({ id: 'r0000000-0000-4000-8000-000000000012', number: 12 }, { status: 201 }) }, reception)
    await screen.findByText('Ana Example')
    fireEvent.click(within(rowOf('Ana Example')).getByRole('button', { name: 'Issue receipt' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'New receipt' }))
    await waitFor(() => expect((dialog.getByLabelText('Service') as HTMLSelectElement).value).toBe(exam))
    fireEvent.click(dialog.getByRole('button', { name: 'Issue receipt' }))
    await screen.findByText('Receipt no. 12 was issued.')

    const row = within(rowOf('Ana Example'))
    expect(row.getByText('Receipt no. 12')).toBeTruthy()
    expect(row.getByRole('link', { name: 'Open PDF' }).getAttribute('href')).toMatch(/\/clinic\/receipts\/r0000000-0000-4000-8000-000000000012\/pdf$/)
    fireEvent.click(row.getByRole('button', { name: 'Issue another receipt' }))
    const ask = within(await screen.findByRole('dialog', { name: 'New receipt' }))
    expect(ask.getByText('This visit already has receipt no. 12. Issue another receipt?')).toBeTruthy()
    expect(ask.queryByLabelText('Service')).toBeNull()
    fireEvent.click(ask.getByRole('button', { name: 'Issue another receipt' }))
    expect(await within(screen.getByRole('dialog', { name: 'New receipt' })).findByLabelText('Service')).toBeTruthy()
  })

  it('offers no receipt for a cancelled visit or a no-show', async () => {
    renderDesk({
      'GET /clinic/desk/today': () =>
        Response.json({
          ...deskDay,
          visits: [
            { ...anaVisit, appointment: { ...anaVisit.appointment, status: 'cancelled', cancellationReason: 'pat' } },
            { ...boVisit, appointment: { ...boVisit.appointment, status: 'noshow' } },
            cleoVisit,
          ],
        }),
    })
    await screen.findByText('Ana Example')
    expect(rowOf('Ana Example').textContent).toContain('Cancelled')
    // Only the patient's name, which opens their card (R2; an app patient's read-only, wave 5 walk finding 6).
    expect(within(rowOf('Ana Example')).getAllByRole('button').map((b) => b.textContent)).toEqual(['Ana Example'])
    expect(within(rowOf('Bo Sample')).getAllByRole('button').map((b) => b.textContent)).toEqual(['Bo Sample'])
    expect(within(rowOf('Cleo Test')).getByRole('button', { name: 'Issue receipt' })).toBeTruthy()
  })

  it("opens the card of a patient the clinic registered from their name (R2), and an app patient's read-only", async () => {
    renderDesk({ 'GET /auth/me': () => Response.json(reception) }, reception)
    fireEvent.click(await screen.findByRole('button', { name: 'Bo Sample' }))
    expect(await screen.findByRole('dialog', { name: 'Patient card' })).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Close' }))
    fireEvent.click(screen.getByRole('button', { name: 'Ana Example' }))
    const card = within(await screen.findByRole('dialog', { name: 'Patient card' }))
    expect(card.queryByRole('textbox')).toBeNull()
  })

  it('names a practitioner who left the clinic', async () => {
    renderDesk({
      'GET /clinic/desk/today': () =>
        Response.json({ ...deskDay, visits: [{ ...anaVisit, appointment: { ...anaVisit.appointment, practitionerRoleId: 'a4000000-0000-4000-8000-0000000000ff' } }] }),
    })
    expect(await screen.findByRole('list', { name: 'Former practitioner' })).toBeTruthy()
  })

  it('says when there are no visits today, and when the day cannot be loaded', async () => {
    renderDesk({ 'GET /clinic/desk/today': () => Response.json({ date: '2026-11-23', visits: [] }) })
    expect(await screen.findByText('No visits today.')).toBeTruthy()
    cleanup()
    renderDesk({ 'GET /clinic/desk/today': () => apiError(403, 'forbidden') })
    expect(await screen.findByText("Today's visits could not be loaded. Reload the page to try again.")).toBeTruthy()
  })

  it("says on a booked visit's row whether a reminder will go, and why not (R3)", async () => {
    const day = {
      ...deskDay,
      visits: deskDay.visits.map((v) =>
        v.patient.id === anaVisit.patient.id ? { ...v, reminder: true } : v.patient.id === boVisit.patient.id ? { ...v, reminder: false, reminderOff: 'no_phone' as const } : { ...v, reminder: false, reminderOff: 'sms_off' as const },
      ),
    }
    renderDesk({ 'GET /clinic/desk/today': () => Response.json(day) }, reception)
    await screen.findByRole('list', { name: 'Cy Example' })
    expect(rowOf('Ana Example').textContent).toContain('Reminder will be sent')
    expect(within(rowOf('Bo Sample')).getByText('No reminder: no phone number')).toBeTruthy()
    // Cleo has arrived: a reminder no longer matters on her row.
    expect(screen.queryByText('No reminder: SMS is off for the clinic')).toBeNull()
  })

  it('opens the walk-in from the toolbar', async () => {
    renderDesk({ 'GET /clinic/patients': () => Response.json([]) })
    fireEvent.click(await screen.findByRole('button', { name: 'Walk-in' }))
    expect(await screen.findByRole('dialog', { name: 'Walk-in' })).toBeTruthy()
  })

  it("opens the search of the clinic's patients from the toolbar, for a patient who has no visit today (finding 5)", async () => {
    renderDesk({ 'GET /clinic/patients': () => Response.json([]) }, reception)
    fireEvent.click(await screen.findByRole('button', { name: 'Patients' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'Patients' }))
    fireEvent.click(dialog.getByRole('button', { name: 'Close' }))
    expect(screen.queryByRole('dialog', { name: 'Patients' })).toBeNull()
  })

  it('shows the desk in Bosnian', async () => {
    renderDesk({}, admin, 'bs')
    await screen.findByText('Ana Example')
    expect(screen.getByRole('button', { name: 'Pacijent bez termina' })).toBeTruthy()
    expect(rowOf('Ana Example').textContent).toContain('2. mart 1980')
    expect(within(rowOf('Ana Example')).getByRole('button', { name: 'Prijavi dolazak' })).toBeTruthy()
  })

  it('moves a booked visit from its row and reads the day again (M1)', async () => {
    let days = 0
    const fake = renderDesk(
      {
        'GET /clinic/desk/today': () => {
          days++
          return Response.json(deskDay)
        },
        'GET /fhir/R4/Schedule': scheduleRoute,
        'GET /fhir/R4/Slot': slotsRoute,
        [`POST /clinic/appointments/${anaVisit.appointment.id}/reschedule`]: () => Response.json({ ...anaVisit.appointment, slotId: freeInPerson, versionId: 2 }),
      },
      reception,
    )
    await screen.findByText('Ana Example')
    expect(within(rowOf('Cleo Test')).queryByRole('button', { name: 'Move' })).toBeNull() // arrived
    fireEvent.click(within(rowOf('Ana Example')).getByRole('button', { name: 'Move' }))
    const dialog = within(await screen.findByRole('dialog', { name: 'Move visit' }))
    const time = (await dialog.findByLabelText('New time')) as HTMLSelectElement
    await waitFor(() => expect(time.options).toHaveLength(2))
    fireEvent.change(time, { target: { value: freeInPerson } })
    fireEvent.click(dialog.getByRole('button', { name: 'Move visit' }))

    expect(await screen.findByText('The visit was moved.')).toBeTruthy()
    expect(screen.queryByRole('dialog')).toBeNull()
    await waitFor(() => expect(days).toBe(2))
    expect(paths(fake.requests)).toContain(`POST /clinic/appointments/${anaVisit.appointment.id}/reschedule`)
  })
})

// Placeholder sites of the clinic (U2 of docs/clinic-cms/WAVE6.md, package LC2).
const site = { id: 'f1000000-0000-4000-8000-000000000001', organizationId: 'org', name: 'Placeholder main site', timeZone: 'Europe/Sarajevo', active: true, versionId: 1 }
const annex = { ...site, id: 'f1000000-0000-4000-8000-000000000002', name: 'Placeholder annex' }
const closed = { ...site, id: 'f1000000-0000-4000-8000-000000000003', name: 'Placeholder closed site', active: false }

describe('Desk by location', () => {
  it('reads the day of one location, with its video visits, once one is picked', async () => {
    const fake = renderDesk(
      {
        'GET /clinic/locations': () => Response.json([site, annex, closed]),
        'GET /clinic/desk/today': (r) => Response.json(new URL(r.url).searchParams.get('locationId') === annex.id ? { ...deskDay, visits: [boVisit] } : deskDay),
      },
      reception,
    )
    const pick = (await screen.findByLabelText('Location')) as HTMLSelectElement
    expect([...pick.options].map((o) => o.textContent)).toEqual(['All locations', 'Placeholder main site', 'Placeholder annex'])
    await screen.findByText('Ana Example')
    fireEvent.change(pick, { target: { value: annex.id } })
    await waitFor(() => expect(screen.queryByText('Ana Example')).toBeNull())
    expect(screen.getByText('Bo Sample')).toBeTruthy()
    const asked = fake.requests.filter((r) => r.url.includes('/clinic/desk/today')).map((r) => new URL(r.url).search)
    expect(asked).toEqual(['', `?locationId=${annex.id}`])
    fireEvent.change(pick, { target: { value: '' } })
    expect(await screen.findByText('Ana Example')).toBeTruthy()
  })

  it("names each in-person visit's location while every location is shown, and none once one is picked", async () => {
    const atAnnex = { ...anaVisit, appointment: { ...anaVisit.appointment, locationId: annex.id } }
    renderDesk(
      {
        'GET /clinic/locations': () => Response.json([site, annex, closed]),
        'GET /clinic/desk/today': () => Response.json({ ...deskDay, visits: [atAnnex, boVisit] }),
      },
      reception,
    )
    const row = (await screen.findByText('Ana Example')).closest('li') as HTMLElement
    expect(within(row).getByText(/Placeholder annex/)).toBeTruthy()
    const video = screen.getByText('Bo Sample').closest('li') as HTMLElement
    expect(within(video).queryByText(/Placeholder/)).toBeNull()
    fireEvent.change(screen.getByLabelText('Location'), { target: { value: annex.id } })
    await waitFor(() => expect(within(screen.getByText('Ana Example').closest('li') as HTMLElement).queryByText(/Placeholder annex/)).toBeNull())
  })

  it('names a visit at a location taken out of use, also when one location is left in use (RD6)', async () => {
    const atClosed = { ...anaVisit, appointment: { ...anaVisit.appointment, locationId: closed.id } }
    for (const locations of [
      [site, annex, closed],
      [site, closed],
    ]) {
      renderDesk({ 'GET /clinic/locations': () => Response.json(locations), 'GET /clinic/desk/today': () => Response.json({ ...deskDay, visits: [atClosed, boVisit] }) }, reception)
      const row = (await screen.findByText('Ana Example')).closest('li') as HTMLElement
      expect(within(row).getByText(/Placeholder closed site \(not in use\)/)).toBeTruthy()
      cleanup()
    }
  })

  it('offers no choice to a clinic with one location in use, or when the locations cannot be read', async () => {
    renderDesk({ 'GET /clinic/locations': () => Response.json([site, closed]) })
    await screen.findByText('Ana Example')
    expect(screen.queryByLabelText('Location')).toBeNull()
    cleanup()
    renderDesk({ 'GET /clinic/locations': () => apiError(500, 'internal') })
    await screen.findByText('Ana Example')
    expect(screen.queryByLabelText('Location')).toBeNull()
  })
})
