import { cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { newApiClient } from './api/client'
import { renderInLanguage } from './i18n/renderInLanguage'
import { PatientPicker } from './PatientPicker'
import { ana, bo } from './test/desk'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from './test/fakeFetch'

afterEach(cleanup)

function renderPicker(routes: Record<string, Route>) {
  const fake = fakeFetch(routes)
  const onPick = vi.fn()
  renderInLanguage(<PatientPicker api={newApiClient(fake.fetch)} onPick={onPick} />, 'en')
  return { fake, onPick }
}

function search(name: string, birthDate = '') {
  fireEvent.change(screen.getByLabelText('Name'), { target: { value: name } })
  fireEvent.change(screen.getByLabelText('Birth date (optional)'), { target: { value: birthDate } })
  fireEvent.click(screen.getByRole('button', { name: 'Search' }))
}

describe('PatientPicker', () => {
  it('searches the clinic’s patients by name and birth date and picks one', async () => {
    const { fake, onPick } = renderPicker({ 'GET /clinic/patients': () => Response.json([ana, bo]) })
    search('Exa', '1980-03-02')

    const found = within(await screen.findByRole('list', { name: 'Patients found' }))
    const options = found.getAllByRole('button')
    expect(options.map((b) => b.textContent)).toEqual(['Ana Example2 March 1980', 'Bo Sample14 July 1992'])
    fireEvent.click(options[0])

    expect(onPick).toHaveBeenCalledWith(ana)
    const asked = new URL(fake.requests[0].url).searchParams
    expect([asked.get('name'), asked.get('birthDate')]).toEqual(['Exa', '1980-03-02'])
  })

  it('searches by name alone, and only from two letters', async () => {
    const { fake } = renderPicker({ 'GET /clinic/patients': () => Response.json([]) })
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'A' } })
    expect((screen.getByRole('button', { name: 'Search' }) as HTMLButtonElement).disabled).toBe(true)
    search('An')

    expect(await screen.findByText('No patient found.')).toBeTruthy()
    expect(new URL(fake.requests[0].url).searchParams.has('birthDate')).toBe(false)
  })

  it('says when the search fails', async () => {
    renderPicker({ 'GET /clinic/patients': () => apiError(500, 'internal error') })
    search('Ana')
    expect(await screen.findByText('The patients could not be searched. Try again.')).toBeTruthy()
  })

  it('registers a patient who has no account, with the consent given in person, and picks them', async () => {
    let body: unknown
    const { fake, onPick } = renderPicker({
      'GET /clinic/patients': () => Response.json([]),
      'POST /clinic/patients': async (r) => {
        body = await r.json()
        return Response.json(bo, { status: 201 })
      },
    })
    search('Bo')
    fireEvent.click(await screen.findByRole('button', { name: 'Register a new patient' }))
    const form = within(screen.getByRole('group', { name: 'New patient' }))
    fireEvent.change(form.getByLabelText('Given name'), { target: { value: ' Bo ' } })
    fireEvent.change(form.getByLabelText('Family name'), { target: { value: 'Sample' } })
    fireEvent.change(form.getByLabelText('Birth date'), { target: { value: '1992-07-14' } })
    fireEvent.change(form.getByLabelText('Phone (optional)'), { target: { value: '' } })
    const register = form.getByRole('button', { name: 'Register' }) as HTMLButtonElement
    expect(register.disabled).toBe(true)
    fireEvent.click(form.getByLabelText('The patient gave the clinic their consent to care, in person.'))
    expect(register.disabled).toBe(false)
    fireEvent.click(register)

    await waitFor(() => expect(onPick).toHaveBeenCalledWith(bo))
    expect(body).toEqual({ givenName: 'Bo', familyName: 'Sample', birthDate: '1992-07-14', consentGivenInPerson: true })
    expectCsrfOnWrites(fake.requests)
  })

  it.each([
    ['invalid_phone', 'Enter the phone number with its country code, such as +387 61 123 456.'],
    ['invalid_birth_date', 'Enter a birth date in the past.'],
    ['invalid_name', 'Enter the given and family name.'],
    ['forbidden', 'The patient could not be registered. Try again.'],
  ])('says why a registration was refused (%s)', async (code, text) => {
    renderPicker({ 'GET /clinic/patients': () => Response.json([]), 'POST /clinic/patients': () => apiError(400, code) })
    search('Bo')
    fireEvent.click(await screen.findByRole('button', { name: 'Register a new patient' }))
    const form = within(screen.getByRole('group', { name: 'New patient' }))
    fireEvent.change(form.getByLabelText('Given name'), { target: { value: 'Bo' } })
    fireEvent.change(form.getByLabelText('Family name'), { target: { value: 'Sample' } })
    fireEvent.change(form.getByLabelText('Birth date'), { target: { value: '1992-07-14' } })
    fireEvent.change(form.getByLabelText('Phone (optional)'), { target: { value: '+38761000002' } })
    fireEvent.click(form.getByLabelText('The patient gave the clinic their consent to care, in person.'))
    fireEvent.click(form.getByRole('button', { name: 'Register' }))

    expect(await screen.findByText(text)).toBeTruthy()
  })
})
