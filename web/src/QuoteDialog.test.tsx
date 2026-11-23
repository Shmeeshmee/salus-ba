import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { newApiClient } from './api/client'
import { renderInLanguage } from './i18n/renderInLanguage'
import type { Language } from './i18n/languages'
import { QuoteDialog, VisitQuotes } from './QuoteDialog'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from './test/fakeFetch'
import { org, patientId } from './test/fixtures'

afterEach(cleanup)

const visit = 'e1000000-0000-4000-8000-000000000001'
const exam = 'c1000000-0000-4000-8000-000000000001'
const lab = 'c1000000-0000-4000-8000-000000000002'
const services = [
  { id: exam, name: 'Placeholder examination', priceMinor: 5000, currency: 'BAM', modes: ['in-person'], active: true, translations: [], versionId: 1 },
  { id: lab, name: 'Placeholder laboratory item', priceMinor: 1170, currency: 'BAM', modes: ['in-person'], active: true, translations: [], versionId: 1 },
]
const quote = (id: string, state: 'sent' | 'expired', appointmentId = visit) => ({
  id, organizationId: org, patientId, appointmentId, issuedAt: '2026-11-09T08:40:00Z', validUntil: '2026-12-09T08:40:00Z', state,
  note: '', currency: 'BAM', totalMinor: 9500, lines: [],
})

function renderDialog(routes: Record<string, Route>, done: { sent?: unknown; closed?: boolean } = {}, language: Language = 'en') {
  const fake = fakeFetch({ 'GET /clinic/services': () => Response.json(services), ...routes })
  renderInLanguage(
    <QuoteDialog
      api={newApiClient(fake.fetch)}
      appointmentId={visit}
      patientId={patientId}
      onSent={(q) => (done.sent = q)}
      onClose={() => (done.closed = true)}
    />,
    language,
  )
  return fake
}

describe('QuoteDialog (clinic-cms X1)', () => {
  it('sends the visit patient a quote of price-list lines with a discount and a note, and says it is no receipt', async () => {
    let body: unknown
    const done: { sent?: unknown } = {}
    const fake = renderDialog(
      {
        'POST /clinic/quotes': async (r) => {
          body = await r.json()
          return Response.json(quote('q1', 'sent'), { status: 201 })
        },
      },
      done,
    )
    const dialog = within(await screen.findByRole('dialog', { name: 'New quote' }))
    expect(dialog.getByText(/A quote is not a receipt: nothing is charged/)).toBeTruthy()
    fireEvent.change(await dialog.findByLabelText('Service'), { target: { value: exam } })
    fireEvent.change(dialog.getByLabelText('Quantity'), { target: { value: '2' } })
    fireEvent.change(dialog.getByLabelText('Discount (BAM)'), { target: { value: '5' } })
    fireEvent.click(dialog.getByRole('button', { name: 'Add line' }))
    fireEvent.change(dialog.getAllByLabelText('Service')[1], { target: { value: lab } })
    fireEvent.change(dialog.getByLabelText('Note for the patient'), { target: { value: 'Placeholder note' } })
    expect(dialog.getByText(/Total: .*106\.70/)).toBeTruthy()
    fireEvent.click(dialog.getByRole('button', { name: 'Send quote' }))

    await screen.findByRole('dialog')
    await new Promise((r) => setTimeout(r, 0))
    expect(body).toEqual({
      patientId,
      appointmentId: visit,
      note: 'Placeholder note',
      lines: [
        { serviceId: exam, quantity: 2, discountMinor: 500 },
        { serviceId: lab, quantity: 1, discountMinor: 0 },
      ],
    })
    expect(done.sent).toMatchObject({ id: 'q1' })
    expectCsrfOnWrites(fake.requests)
  })

  it('sends nothing while a line is incomplete, and names a refusal', async () => {
    let posts = 0
    renderDialog({
      'POST /clinic/quotes': () => {
        posts++
        return apiError(404, 'not_found')
      },
    })
    const dialog = within(await screen.findByRole('dialog', { name: 'New quote' }))
    await dialog.findByLabelText('Service')
    expect(dialog.getByRole('button', { name: 'Send quote' })).toHaveProperty('disabled', true)
    fireEvent.change(dialog.getByLabelText('Service'), { target: { value: exam } })
    fireEvent.change(dialog.getByLabelText('Discount (BAM)'), { target: { value: '51' } })
    expect(dialog.getByRole('button', { name: 'Send quote' })).toHaveProperty('disabled', true)
    fireEvent.change(dialog.getByLabelText('Discount (BAM)'), { target: { value: '' } })
    fireEvent.click(dialog.getByRole('button', { name: 'Send quote' }))
    expect(await dialog.findByText('A quote cannot be sent to this patient from this visit.')).toBeTruthy()
    expect(posts).toBe(1)
  })

  it('speaks every staff language', async () => {
    renderDialog({}, {}, 'bs')
    expect(await screen.findByRole('dialog', { name: 'Nova ponuda' })).toBeTruthy()
  })
})

describe('VisitQuotes (clinic-cms X1)', () => {
  it("lists the visit's quotes with their state, asking for that visit's only", async () => {
    let asked = ''
    const fake = fakeFetch({
      'GET /clinic/quotes': (r) => {
        asked = new URL(r.url).searchParams.get('appointmentId') ?? ''
        return Response.json([quote('q2', 'expired'), quote('q1', 'sent')])
      },
    })
    renderInLanguage(<VisitQuotes api={newApiClient(fake.fetch)} appointmentId={visit} zone="Europe/Sarajevo" />, 'en')
    const items = await screen.findAllByRole('listitem')
    expect(asked).toBe(visit)
    expect(items).toHaveLength(2)
    expect(items[0].textContent).toContain('Expired')
    expect(items[1].textContent).toContain('Sent')
    expect(items[1].textContent).toMatch(/9 November 2026.*95\.00.*valid until Wednesday, 9 December 2026/)
  })

  it('says when there are none or they cannot be loaded', async () => {
    const empty = fakeFetch({ 'GET /clinic/quotes': () => Response.json([]) })
    renderInLanguage(<VisitQuotes api={newApiClient(empty.fetch)} appointmentId={visit} zone="Europe/Sarajevo" />, 'en')
    expect(await screen.findByText('No quotes from this visit.')).toBeTruthy()
    cleanup()
    const failing = fakeFetch({ 'GET /clinic/quotes': () => apiError(500, 'internal') })
    renderInLanguage(<VisitQuotes api={newApiClient(failing.fetch)} appointmentId={visit} zone="Europe/Sarajevo" />, 'en')
    expect(await screen.findByText('The quotes could not be loaded. Reload the page to try again.')).toBeTruthy()
  })
})
