import { cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { DeskPatient } from './DeskPatient'
import { newApiClient, type Account } from './api/client'
import { renderInLanguage } from './i18n/renderInLanguage'
import { ana, bo, ida } from './test/desk'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from './test/fakeFetch'
import { admin } from './test/fixtures'

afterEach(cleanup)

const reception: Account = { ...admin, id: 'a1000000-0000-4000-8000-000000000009', email: 'desk@example.test', role: 'reception' }

function renderCard(routes: Record<string, Route>, account: Account = admin, patient = bo) {
  const fake = fakeFetch({ 'GET /auth/me': () => Response.json(account), ...routes })
  renderInLanguage(<DeskPatient api={newApiClient(fake.fetch)} patient={patient} />, 'en')
  return fake
}

async function openCard() {
  fireEvent.click(screen.getByRole('button', { name: 'Bo Sample' }))
  return within(await screen.findByRole('dialog', { name: 'Patient card' }))
}

describe('DeskPatient', () => {
  it("shows an app patient's name as text, without a card", () => {
    renderCard({}, admin, ana)
    expect(screen.getByText('Ana Example')).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Ana Example' })).toBeNull()
  })

  it('corrects the profile at the version read and shows the new name', async () => {
    const fake = renderCard({
      [`PUT /clinic/patients/${bo.id}`]: () => Response.json({ ...bo, givenName: 'Boris', phone: '+38761000009', versionId: 2 }),
    })
    const card = await openCard()
    fireEvent.change(card.getByLabelText('Given name'), { target: { value: ' Boris ' } })
    fireEvent.change(card.getByLabelText('Phone (optional)'), { target: { value: '+387 61 000 009' } })
    fireEvent.click(card.getByRole('button', { name: 'Save' }))
    expect(await card.findByText('The profile is saved.')).toBeTruthy()
    const put = fake.requests.find((r) => r.method === 'PUT')!
    expect(put.headers.get('If-Match')).toBe('W/"1"')
    expect(await put.json()).toEqual({ givenName: 'Boris', familyName: 'Sample', birthDate: '1992-07-14', phone: '+38761000009' })
    expectCsrfOnWrites(fake.requests)
    fireEvent.click(card.getByRole('button', { name: 'Close' }))
    expect(await screen.findByRole('button', { name: 'Boris Sample' })).toBeTruthy()
  })

  it('says when someone else changed the profile meanwhile', async () => {
    renderCard({ [`PUT /clinic/patients/${bo.id}`]: () => apiError(409, 'version_conflict') })
    const card = await openCard()
    fireEvent.click(card.getByRole('button', { name: 'Save' }))
    expect(await card.findByText('Someone changed this profile meanwhile. Close the card and open it again.')).toBeTruthy()
  })

  it('records the withdrawal of the consent only after it is confirmed', async () => {
    const fake = renderCard({ [`POST /clinic/patients/${bo.id}/consent-withdrawal`]: () => new Response(null, { status: 204 }) }, reception)
    const card = await openCard()
    fireEvent.click(card.getByRole('button', { name: 'Record withdrawal of consent' }))
    expect(fake.requests.some((r) => r.url.includes('consent-withdrawal'))).toBe(false)
    expect(card.getByText(/Their visits ahead are cancelled/)).toBeTruthy()
    fireEvent.click(card.getByRole('button', { name: 'Confirm withdrawal' }))
    expect(await card.findByText('The withdrawal is recorded.')).toBeTruthy()
    expect(fake.requests.filter((r) => r.url.includes('consent-withdrawal'))).toHaveLength(1)
  })

  it('says when there is no consent given in person to withdraw', async () => {
    renderCard({ [`POST /clinic/patients/${bo.id}/consent-withdrawal`]: () => apiError(409, 'consent_inactive') })
    const card = await openCard()
    fireEvent.click(card.getByRole('button', { name: 'Record withdrawal of consent' }))
    fireEvent.click(card.getByRole('button', { name: 'Confirm withdrawal' }))
    expect(await card.findByText('There is no active consent given in person to withdraw.')).toBeTruthy()
  })

  it("offers the record's download to the admin only", async () => {
    renderCard({})
    let card = await openCard()
    const link = await card.findByRole('link', { name: 'Download the record' })
    expect(link.getAttribute('href')).toContain(`/clinic/patients/${bo.id}/record`)
    expect(link.hasAttribute('download')).toBe(true)
    cleanup()
    renderCard({}, reception)
    card = await openCard()
    await waitFor(() => expect(card.getByRole('button', { name: 'Record withdrawal of consent' })).toBeTruthy())
    expect(card.queryByRole('link', { name: 'Download the record' })).toBeNull()
  })

  it('records the consent an imported patient gives in person, from their card (wave 4)', async () => {
    const fake = renderCard({ [`POST /clinic/patients/${ida.id}/consent`]: () => new Response(null, { status: 204 }) }, reception, ida)
    fireEvent.click(screen.getByRole('button', { name: 'Ida Import' }))
    const card = within(await screen.findByRole('dialog', { name: 'Patient card' }))
    expect(card.queryByRole('button', { name: 'Record withdrawal of consent' })).toBeNull()
    fireEvent.click(card.getByLabelText('The patient gave the clinic their consent to care, in person.'))
    fireEvent.click(card.getByRole('button', { name: 'Record consent' }))
    expect(await card.findByText('The consent is recorded.')).toBeTruthy()
    expect(card.getByRole('button', { name: 'Record withdrawal of consent' })).toBeTruthy()
    expect(fake.requests.filter((r) => r.method === 'POST').map((r) => new URL(r.url).pathname)).toEqual([`/clinic/patients/${ida.id}/consent`])
    expectCsrfOnWrites(fake.requests)
  })

  it('offers to record a new consent once the patient withdrew theirs', async () => {
    renderCard({ [`POST /clinic/patients/${bo.id}/consent-withdrawal`]: () => new Response(null, { status: 204 }) }, reception)
    const card = await openCard()
    fireEvent.click(card.getByRole('button', { name: 'Record withdrawal of consent' }))
    fireEvent.click(card.getByRole('button', { name: /Confirm/ }))
    expect(await card.findByRole('button', { name: 'Record consent' })).toBeTruthy()
  })

  it('tells the desk to read the day again when the card closes after a change, and not otherwise', async () => {
    const fake = fakeFetch({
      'GET /auth/me': () => Response.json(reception),
      [`PUT /clinic/patients/${bo.id}`]: () => Response.json({ ...bo, phone: '+38761000009', versionId: 2 }),
    })
    let changed = 0
    renderInLanguage(<DeskPatient api={newApiClient(fake.fetch)} patient={bo} onChanged={() => changed++} />, 'en')
    let card = await openCard()
    fireEvent.click(card.getByRole('button', { name: 'Close' }))
    expect(changed).toBe(0)
    card = await openCard()
    fireEvent.change(card.getByLabelText('Phone (optional)'), { target: { value: '061 000 009' } })
    fireEvent.click(card.getByRole('button', { name: 'Save' }))
    await card.findByText('The profile is saved.')
    fireEvent.click(card.getByRole('button', { name: 'Close' }))
    expect(changed).toBe(1)
  })
})
