import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderInLanguage } from './i18n/renderInLanguage'
import { AppointmentDetail } from './AppointmentDetail'
import { newApiClient, type Account } from './api/client'
import { newFhirClient, type Patient } from './api/fhir'
import { formatDate } from './i18n/format'
import type { Language } from './i18n/languages'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, fhirJSON, outcome, paths, type Route } from './test/fakeFetch'
import { admin, appointment, clinicRoute, cy, dee, meta, org, patientId, practitioner } from './test/fixtures'

afterEach(cleanup)

const id = 'e1000000-0000-4000-8000-000000000001'
// Monday 9 November 2026, 09:00 to 09:30 in Europe/Sarajevo.
const booked = appointment(id, '2026-11-09T08:00:00Z', '2026-11-09T08:30:00Z', {}, 'Ada Example')
const before = () => new Date('2026-11-09T07:59:00Z')
const after = () => new Date('2026-11-09T08:01:00Z')

const patient: Patient = {
  resourceType: 'Patient',
  id: patientId,
  meta,
  name: [{ given: ['Ada'], family: 'Example' }],
  birthDate: '1990-01-01',
  telecom: [{ system: 'phone', value: '+38760000001', use: 'mobile' }],
}

const read = `GET /fhir/R4/Appointment/${id}`
const patientRead = `GET /fhir/R4/Patient/${patientId}`
const statusPost = `POST /appointments/${id}/status`
const cancelPost = `POST /appointments/${id}/cancel`
const statusGet = `GET /appointments/${id}/consultation`
const questionnaireSearch = 'GET /fhir/R4/QuestionnaireResponse'

function answer(status: string) {
  return Response.json({ id, organizationId: org, practitionerRoleId: cy.roleId, slotId: id, start: booked.start, end: booked.end, status, versionId: 2 })
}

function renderDetail(account: Account, routes: Record<string, Route>, now = after, language: Language = 'en') {
  const fake = fakeFetch({
    ...clinicRoute,
    [statusGet]: () => apiError(404, 'not_found'),
    [questionnaireSearch]: () => fhirJSON({ resourceType: 'Bundle', type: 'searchset' }),
    ...routes,
  })
  renderInLanguage(<AppointmentDetail api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} account={account} id={id} now={now} />, language)
  return fake
}

describe('AppointmentDetail', () => {
  it('shows the date and times in the clinic zone, and the status', async () => {
    renderDetail(practitioner, { [read]: () => fhirJSON(booked), [patientRead]: () => outcome(404, 'not-found') })

    expect(await screen.findByText('Monday, 9 November 2026, 09:00–09:30')).toBeTruthy()
    expect(screen.getByText('Booked')).toBeTruthy()
  })

  it('shows the patient profile behind a disclosure when the Patient read answers', async () => {
    renderDetail(practitioner, { [read]: () => fhirJSON(booked), [patientRead]: () => fhirJSON(patient) })

    expect(await screen.findByText('Patient details')).toBeTruthy()
    expect(screen.getByRole('heading', { name: 'Ada Example' })).toBeTruthy()
    expect(screen.getByText('1 January 1990')).toBeTruthy()
    expect(screen.getByText('+38760000001')).toBeTruthy()
  })

  it('hides the profile on a 404 and shows the name the appointment gives', async () => {
    renderDetail(practitioner, { [read]: () => fhirJSON(booked), [patientRead]: () => outcome(404, 'not-found') })

    expect(await screen.findByRole('heading', { name: 'Ada Example' })).toBeTruthy()
    await vi.waitFor(() => expect(screen.queryByText('Loading')).toBeNull())
    expect(screen.queryByText('Patient details')).toBeNull()
    expect(screen.queryByText('1 January 1990')).toBeNull()
  })

  it('shows no name at all when neither the Patient read nor the appointment gives one', async () => {
    const nameless = appointment(id, booked.start, booked.end)
    renderDetail(practitioner, { [read]: () => fhirJSON(nameless), [patientRead]: () => outcome(404, 'not-found') })

    expect(await screen.findByRole('heading', { name: 'Appointment' })).toBeTruthy()
    await vi.waitFor(() => expect(screen.queryByText('Loading')).toBeNull())
    expect(screen.queryByText('Patient details')).toBeNull()
  })

  it('does not read the Patient for an admin, who sees the name only, and offers cancel but no outcome', async () => {
    const { requests } = renderDetail(admin, { [read]: () => fhirJSON(booked) })

    expect(await screen.findByRole('heading', { name: 'Ada Example' })).toBeTruthy()
    expect(paths(requests)).not.toContain(patientRead)
    expect(screen.getByRole('button', { name: 'Cancel appointment' })).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Mark fulfilled' })).toBeNull()
  })

  it('disables fulfilled and no-show before the start', async () => {
    renderDetail(practitioner, { [read]: () => fhirJSON(booked), [patientRead]: () => outcome(404, 'not-found') }, before)

    expect(((await screen.findByRole('button', { name: 'Mark fulfilled' })) as HTMLButtonElement).disabled).toBe(true)
    expect((screen.getByRole('button', { name: 'Mark no-show' }) as HTMLButtonElement).disabled).toBe(true)
    expect(screen.getByText('You can record the outcome once the appointment has started.')).toBeTruthy()
    expect((screen.getByRole('button', { name: 'Cancel appointment' }) as HTMLButtonElement).disabled).toBe(false)
  })

  it.each([
    ['Mark fulfilled', 'fulfilled', 'Fulfilled'],
    ['Mark no-show', 'noshow', 'No-show'],
  ])('%s once the start has passed, with the CSRF header', async (button, status, label) => {
    const { requests } = renderDetail(practitioner, {
      [read]: () => fhirJSON(booked),
      [patientRead]: () => outcome(404, 'not-found'),
      [statusPost]: () => answer(status),
    })

    fireEvent.click(await screen.findByRole('button', { name: button }))

    expect(await screen.findByText(label)).toBeTruthy()
    const post = requests.find((r) => r.method === 'POST')!
    expect(await post.json()).toEqual({ status })
    expectCsrfOnWrites(requests)
    expect(screen.queryByRole('button', { name: 'Mark fulfilled' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Cancel appointment' })).toBeNull()
  })

  it('cancels after a confirmation, with the CSRF header', async () => {
    const { requests } = renderDetail(admin, { [read]: () => fhirJSON(booked), [cancelPost]: () => answer('cancelled') })

    fireEvent.click(await screen.findByRole('button', { name: 'Cancel appointment' }))
    expect(requests.filter((r) => r.method === 'POST')).toHaveLength(0)
    fireEvent.click(screen.getByRole('button', { name: 'Yes, cancel' }))

    expect(await screen.findByText('Cancelled')).toBeTruthy()
    expect(paths(requests)).toContain(cancelPost)
    expectCsrfOnWrites(requests)
  })

  it('keeps the appointment when the cancellation is not confirmed', async () => {
    const { requests } = renderDetail(admin, { [read]: () => fhirJSON(booked) })

    fireEvent.click(await screen.findByRole('button', { name: 'Cancel appointment' }))
    fireEvent.click(screen.getByRole('button', { name: 'Keep it' }))

    expect(screen.getByRole('button', { name: 'Cancel appointment' })).toBeTruthy()
    expect(requests.filter((r) => r.method === 'POST')).toHaveLength(0)
  })

  it('shows no actions on an appointment that is no longer booked', async () => {
    renderDetail(practitioner, { [read]: () => fhirJSON({ ...booked, status: 'fulfilled' }), [patientRead]: () => outcome(404, 'not-found') })

    expect(await screen.findByText('Fulfilled')).toBeTruthy()
    expect(screen.queryByRole('button')).toBeNull()
  })

  it.each([
    [409, 'appointment_closed', 'This appointment is already closed. Reload the page to see its status.'],
    [409, 'appointment_not_started', 'The appointment has not started yet.'],
    [403, 'forbidden', 'You cannot change this appointment.'],
    [500, 'internal error', 'Something went wrong. Try again.'],
  ])('maps %i %s to a message', async (status, code, message) => {
    renderDetail(practitioner, {
      [read]: () => fhirJSON(booked),
      [patientRead]: () => outcome(404, 'not-found'),
      [statusPost]: () => apiError(status, code),
    })

    fireEvent.click(await screen.findByRole('button', { name: 'Mark fulfilled' }))

    await vi.waitFor(() => expect(screen.getByRole('alert').textContent).toBe(message))
    expect(screen.getByText('Booked')).toBeTruthy()
  })

  it('says when the appointment cannot be found', async () => {
    renderDetail(practitioner, { [read]: () => outcome(404, 'not-found') })

    expect((await screen.findByRole('alert')).textContent).toBe('This appointment could not be found.')
  })

  describe('the join button', () => {
    const routes = { [read]: () => fhirJSON(booked), [patientRead]: () => outcome(404, 'not-found') }
    const joinButton = () => screen.queryByRole('button', { name: 'Join consultation' })

    it('opens the consultation inside the window', async () => {
      renderDetail(practitioner, routes)

      fireEvent.click(await screen.findByRole('button', { name: 'Join consultation' }))

      expect(location.pathname).toBe(`/appointments/${id}/call`)
      history.replaceState(null, '', '/')
    })

    it('says when joining opens, before the window', async () => {
      renderDetail(practitioner, routes, () => new Date('2026-11-09T07:49:59Z'))

      expect(await screen.findByText('You can join from 10 minutes before the start.')).toBeTruthy()
      expect(joinButton()).toBeNull()
    })

    it('is shown from 10 minutes before the start', async () => {
      renderDetail(practitioner, routes, () => new Date('2026-11-09T07:50:00Z'))

      expect(await screen.findByRole('button', { name: 'Join consultation' })).toBeTruthy()
      expect(screen.queryByText('You can join from 10 minutes before the start.')).toBeNull()
    })

    it('is gone 30 minutes after the end', async () => {
      renderDetail(practitioner, routes, () => new Date('2026-11-09T09:00:00Z'))

      await screen.findByRole('button', { name: 'Mark fulfilled' })
      expect(joinButton()).toBeNull()
      expect(screen.queryByText('You can join from 10 minutes before the start.')).toBeNull()
    })

    it('is not offered to an admin, who cannot join', async () => {
      renderDetail(admin, { [read]: () => fhirJSON(booked) })

      await screen.findByRole('button', { name: 'Cancel appointment' })
      expect(joinButton()).toBeNull()
    })
  })

  describe('the consultation', () => {
    const routes = (status: unknown) => ({
      [read]: () => fhirJSON(booked),
      [patientRead]: () => outcome(404, 'not-found'),
      [statusGet]: () => Response.json(status),
    })
    const section = async () => within(await screen.findByRole('region', { name: 'Consultation' }))

    it('shows that it has not started', async () => {
      renderDetail(practitioner, routes({ state: 'not_started' }))

      const s = await section()
      expect(s.getByText('Not started')).toBeTruthy()
      expect(s.queryByText(/joined|did not join/)).toBeNull()
    })

    it('shows who joined, in the clinic zone, while it is open', async () => {
      renderDetail(practitioner, routes({ state: 'open', practitionerJoinedAt: '2026-11-09T07:58:00Z' }))

      const s = await section()
      expect(s.getByText('Open')).toBeTruthy()
      expect(s.getByText('Practitioner joined at 08:58')).toBeTruthy()
      expect(s.queryByText(/Patient/)).toBeNull()
    })

    it('shows to the admin who did not join once it has ended', async () => {
      renderDetail(admin, {
        [read]: () => fhirJSON(booked),
        [statusGet]: () => Response.json({ state: 'ended', patientJoinedAt: '2026-11-09T08:01:00Z', endedAt: '2026-11-09T09:00:00Z', endReason: 'practitioner_absent' }),
      })

      const s = await section()
      expect(s.getByText('Ended')).toBeTruthy()
      expect(s.getByText('Patient joined at 09:01')).toBeTruthy()
      expect(s.getByText('Practitioner did not join')).toBeTruthy()
    })

    it('shows that neither joined', async () => {
      renderDetail(practitioner, routes({ state: 'ended', endedAt: '2026-11-09T09:00:00Z', endReason: 'practitioner_absent' }))

      const s = await section()
      expect(s.getByText('Patient did not join')).toBeTruthy()
      expect(s.getByText('Practitioner did not join')).toBeTruthy()
    })

    it('is left out when the status cannot be read', async () => {
      renderDetail(practitioner, { [read]: () => fhirJSON(booked), [patientRead]: () => outcome(404, 'not-found') })

      await screen.findByRole('button', { name: 'Mark fulfilled' })
      await vi.waitFor(() => expect(screen.queryByText('Loading')).toBeNull())
      expect(screen.queryByRole('region', { name: 'Consultation' })).toBeNull()
    })
  })

  describe('the questionnaires', () => {
    const routes = { [read]: () => fhirJSON(booked), [patientRead]: () => outcome(404, 'not-found') }
    const section = async () => within(await screen.findByRole('region', { name: 'Questionnaires' }))

    it("are shown to the appointment's practitioner, who may add one before the start", async () => {
      renderDetail(practitioner, routes, before)

      expect(await (await section()).findByRole('button', { name: 'Add questionnaire' })).toBeTruthy()
    })

    it('are shown to an admin, who may not add one', async () => {
      renderDetail(admin, { [read]: () => fhirJSON(booked) }, before)

      await (await section()).findByText('No questionnaires requested.')
      expect(screen.queryByRole('button', { name: 'Add questionnaire' })).toBeNull()
    })

    it('cannot be added by a practitioner who is not the appointment\'s', async () => {
      renderDetail({ ...practitioner, practitionerId: dee.practitionerId }, routes, before)

      await (await section()).findByText('No questionnaires requested.')
      expect(screen.queryByRole('button', { name: 'Add questionnaire' })).toBeNull()
    })
  })
})

describe('AppointmentDetail in a language', () => {
  it('shows the appointment, the profile and the actions in the language', async () => {
    renderDetail(practitioner, { [read]: () => fhirJSON(booked), [patientRead]: () => fhirJSON(patient) }, after, 'sr-Cyrl')

    const date = formatDate(new Date(booked.start), 'Europe/Sarajevo', 'sr-Cyrl', 'long')
    expect(date).toMatch(/понедјељак/i)
    expect(await screen.findByText(`${date}, 09:00–09:30`)).toBeTruthy()
    expect(screen.getByText('Заказан')).toBeTruthy()
    expect(screen.getByText('Подаци о пацијенту')).toBeTruthy()
    expect(screen.getByText(/јануар 1990/)).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Означи као обављен' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Откажи термин' })).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Календар' })).toBeTruthy()
  })

  it('shows the consultation status in the language', async () => {
    renderDetail(admin, {
      [read]: () => fhirJSON(booked),
      [statusGet]: () => Response.json({ state: 'ended', patientJoinedAt: '2026-11-09T08:01:00Z', endedAt: '2026-11-09T09:00:00Z', endReason: 'practitioner_absent' }),
    }, after, 'hr')

    const s = within(await screen.findByRole('region', { name: 'Konzultacija' }))
    expect(s.getByText('Završena')).toBeTruthy()
    expect(s.getByText('Pacijent se pridružio u 09:01')).toBeTruthy()
    expect(s.getByText('Liječnik se nije pridružio')).toBeTruthy()
  })

  it('asks before cancelling, in the language', async () => {
    renderDetail(admin, { [read]: () => fhirJSON(booked) }, after, 'bs')

    fireEvent.click(await screen.findByRole('button', { name: 'Otkaži termin' }))

    expect(screen.getByText('Otkazati ovaj termin? Pacijent će biti obaviješten e-mailom, a vrijeme ponovo postaje slobodno.')).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Da, otkaži' })).toBeTruthy()
  })
})
