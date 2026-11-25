import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { DeskPatients } from './DeskPatients'
import { newApiClient } from './api/client'
import { renderInLanguage } from './i18n/renderInLanguage'
import { ana, ida } from './test/desk'
import { apiError, fakeFetch, type Route } from './test/fakeFetch'
import { admin } from './test/fixtures'

afterEach(cleanup)

function renderPatients(routes: Record<string, Route>) {
  const fake = fakeFetch({ 'GET /auth/me': () => Response.json(admin), ...routes })
  const onChanged = vi.fn()
  const onClose = vi.fn()
  renderInLanguage(<DeskPatients api={newApiClient(fake.fetch)} onChanged={onChanged} onClose={onClose} />, 'en')
  return { fake, onChanged, onClose, dialog: within(screen.getByRole('dialog', { name: 'Patients' })) }
}

describe('DeskPatients', () => {
  it("finds the clinic's patients by name and birth date and opens the card of one without a consent (wave 4 walk, finding 5)", async () => {
    const { fake, dialog } = renderPatients({ 'GET /clinic/patients': () => Response.json([ana, ida]) })
    fireEvent.change(dialog.getByLabelText('Name'), { target: { value: ' Im ' } })
    fireEvent.change(dialog.getByLabelText('Birth date (optional)'), { target: { value: '1975-05-05' } })
    fireEvent.click(dialog.getByRole('button', { name: 'Search' }))

    const found = within(await dialog.findByRole('list', { name: 'Patients found' }))
    const rows = found.getAllByRole('listitem').map((r) => r.textContent)
    expect(rows[0]).toContain('Ana Example')
    expect(rows[1]).toContain('Ida Import')
    expect(rows[1]).toContain('5 May 1975')
    expect(rows[1]).toContain('+387 61 000 004')
    const asked = new URL(fake.requests.find((r) => r.url.includes('/clinic/patients'))!.url).searchParams
    expect(asked.get('name')).toBe('Im')
    expect(asked.get('birthDate')).toBe('1975-05-05')
    // An app patient manages their profile in the app: their card is read-only (wave 5 walk, finding 6).
    expect(found.getByRole('button', { name: 'Ana Example' })).toBeTruthy()
    fireEvent.click(found.getByRole('button', { name: 'Ida Import' }))
    expect(await screen.findByRole('dialog', { name: 'Patient card' })).toBeTruthy()
  })

  it('says when nobody is found and when the search fails', async () => {
    let fail = false
    const { dialog } = renderPatients({ 'GET /clinic/patients': () => (fail ? apiError(500, 'internal') : Response.json([])) })
    fireEvent.change(dialog.getByLabelText('Name'), { target: { value: 'Nobody' } })
    fireEvent.click(dialog.getByRole('button', { name: 'Search' }))
    expect(await dialog.findByText('No patient found.')).toBeTruthy()
    fail = true
    fireEvent.click(dialog.getByRole('button', { name: 'Search' }))
    expect(await dialog.findByText('The patients could not be searched. Try again.')).toBeTruthy()
  })

  it('closes, and tells the desk when a card changed something', async () => {
    const { onClose, onChanged, dialog } = renderPatients({})
    fireEvent.click(dialog.getByRole('button', { name: 'Close' }))
    expect(onClose).toHaveBeenCalled()
    expect(onChanged).not.toHaveBeenCalled()
  })
})
