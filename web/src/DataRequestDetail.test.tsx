import { cleanup, fireEvent, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { DataRequestDetail } from './DataRequestDetail'
import { newApiClient } from './api/client'
import { newFhirClient } from './api/fhir'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, paths, type Route } from './test/fakeFetch'
import { requests } from './test/dataRequests'
import { admin, clinicRoute } from './test/fixtures'

afterEach(cleanup)

const open = { ...requests[0], history: [{ status: 'received' as const, changedAt: '2026-11-02T09:00:00Z' }] }
const id = open.id
const patient = open.patientId!

function renderDetail(routes: Record<string, Route> = {}) {
  const fake = fakeFetch({ ...clinicRoute, [`GET /clinic/data-requests/${id}`]: () => Response.json(open), ...routes })
  renderInLanguage(<DataRequestDetail api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} account={admin} id={id} />, 'en')
  return fake
}

describe('DataRequestDetail', () => {
  it('shows the request with its due date, what it needs and its history', async () => {
    renderDetail()
    expect(await screen.findByRole('heading', { name: 'Erasure' })).toBeTruthy()
    expect(screen.getByText('Due Wednesday, 2 December 2026')).toBeTruthy()
    expect(screen.getByText(/Erase what has no legal basis to stay/)).toBeTruthy()
    expect(screen.getByRole('list', { name: 'History' }).textContent).toContain('Received')
    expect(screen.getByRole('link', { name: 'All requests' }).getAttribute('href')).toBe('/data-requests')
    expect(screen.getByRole('link', { name: 'Open the hand-over' }).getAttribute('href')).toBe(`${location.origin}/clinic/patients/${patient}/record`)
  })

  it('answers with a code, at the version read, and shows the answer', async () => {
    const fake = renderDetail({
      [`POST /clinic/data-requests/${id}/answer`]: () => Response.json({ ...open, status: 'fulfilled', answerCode: 'done_partly_kept', answeredAt: '2026-11-05T09:00:00Z', version: 2 }),
    })
    fireEvent.change(await screen.findByLabelText('Answer'), { target: { value: 'done_partly_kept' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send the answer' }))

    expect(await screen.findByText('Done, the medical record kept')).toBeTruthy()
    const post = fake.requests.find((r) => r.method === 'POST')!
    expect(post.headers.get('If-Match')).toBe('W/"1"')
    expect(await post.json()).toEqual({ status: 'fulfilled', code: 'done_partly_kept' })
    expect(screen.queryByRole('button', { name: 'Send the answer' })).toBeNull()
    expectCsrfOnWrites(fake.requests)
  })

  it('extends once with a reason, and refuses as the API does', async () => {
    const fake = renderDetail({
      [`POST /clinic/data-requests/${id}/extend`]: () => Response.json({ ...open, extendedUntil: '2027-02-02T09:00:00Z', extensionReason: 'many_requests', version: 2 }),
    })
    fireEvent.change(await screen.findByLabelText('Reason'), { target: { value: 'many_requests' } })
    fireEvent.click(screen.getByRole('button', { name: 'Extend by two months' }))
    expect(await screen.findByText('Due Tuesday, 2 February 2027')).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Extend by two months' })).toBeNull()
    expect(await fake.requests.find((r) => r.method === 'POST')!.json()).toEqual({ reason: 'many_requests' })
    cleanup()

    renderDetail({ [`POST /clinic/data-requests/${id}/answer`]: () => apiError(412, 'version_conflict') })
    fireEvent.click(await screen.findByRole('button', { name: 'Send the answer' }))
    expect((await screen.findByRole('alert')).textContent).toBe('The request changed meanwhile. Reload the page.')
    cleanup()

    renderDetail({ [`POST /clinic/data-requests/${id}/extend`]: () => apiError(409, 'request_overdue') })
    fireEvent.click(await screen.findByRole('button', { name: 'Extend by two months' }))
    expect((await screen.findByRole('alert')).textContent).toBe('Its due date has passed, so it can no longer be extended.')
  })

  it("restricts the patient's record for the request and lifts the restriction", async () => {
    const fake = renderDetail({
      [`PUT /clinic/patients/${patient}/restriction`]: () => new Response(null, { status: 204 }),
      [`DELETE /clinic/patients/${patient}/restriction`]: () => new Response(null, { status: 204 }),
    })
    fireEvent.click(await screen.findByRole('button', { name: 'Restrict the record' }))
    expect((await screen.findByRole('status')).textContent).toBe('The record is restricted.')
    const put = fake.requests.find((r) => r.method === 'PUT')!
    expect(await put.json()).toEqual({ dataRequestId: id })

    fireEvent.click(screen.getByRole('button', { name: 'Lift the restriction' }))
    expect(await screen.findByText('The restriction is lifted.')).toBeTruthy()
    expect(paths(fake.requests)).toContain(`DELETE /clinic/patients/${patient}/restriction`)
    expectCsrfOnWrites(fake.requests)
  })

  it('lifts a restriction after the request is answered, and says when the erasure keeps it', async () => {
    const answered = { ...open, status: 'fulfilled' as const, answerCode: 'done' as const, answeredAt: '2026-11-05T09:00:00Z', version: 2 }
    renderDetail({
      [`GET /clinic/data-requests/${id}`]: () => Response.json(answered),
      [`DELETE /clinic/patients/${patient}/restriction`]: () => apiError(403, 'forbidden'),
    })
    fireEvent.click(await screen.findByRole('button', { name: 'Lift the restriction' }))
    expect(screen.queryByRole('button', { name: 'Send the answer' })).toBeNull()
    expect((await screen.findByRole('alert')).textContent).toBe('The erasure keeps this record restricted; the clinic cannot lift it.')
  })

  it('says when the request does not exist or a change fails', async () => {
    renderDetail({ [`GET /clinic/data-requests/${id}`]: () => apiError(404, 'not_found') })
    expect((await screen.findByRole('alert')).textContent).toBe('This request does not exist.')
    cleanup()

    renderDetail({ [`PUT /clinic/patients/${patient}/restriction`]: () => apiError(500, 'internal error') })
    fireEvent.click(await screen.findByRole('button', { name: 'Restrict the record' }))
    expect((await screen.findByRole('alert')).textContent).toBe('The change could not be saved. Try again.')
  })
})
