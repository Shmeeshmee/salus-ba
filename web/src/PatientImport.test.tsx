import { cleanup, fireEvent, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { PatientImport } from './PatientImport'
import { newApiClient } from './api/client'
import { renderInLanguage } from './i18n/renderInLanguage'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from './test/fakeFetch'

afterEach(cleanup)

const path = '/clinic/patients/import'
// Synthetic patients, not real people.
const ada = { givenName: 'Ada', familyName: 'Sample', birthDate: '1990-01-31', phone: '+38761000111' }
const bo = { givenName: 'Bo', familyName: 'Sample', birthDate: '1991-02-03' }

function renderImport(route: Route) {
  const fake = fakeFetch({ [`POST ${path}`]: route })
  const onImported = vi.fn()
  const onCancel = vi.fn()
  renderInLanguage(<PatientImport api={newApiClient(fake.fetch)} onImported={onImported} onCancel={onCancel} />, 'en')
  return { ...fake, onImported, onCancel }
}

const choose = () =>
  fireEvent.change(screen.getByLabelText('File (CSV or XLSX)'), {
    target: { files: [new File(['ime;prezime;datum rođenja\n'], 'patients.csv', { type: 'text/csv' })] },
  })

describe('PatientImport', () => {
  it('previews the file with its likely duplicates, then imports every row', async () => {
    const { requests, onImported } = renderImport(async (r) => {
      const save = new URL(r.url).searchParams.get('save') === 'true'
      return Response.json({
        saved: save,
        rows: [
          { line: 2, errors: [], duplicate: true, patient: ada },
          { line: 3, errors: [], duplicate: false, patient: bo },
        ],
      })
    })
    choose()
    expect(await screen.findByText('2 patients read. Nothing is saved until you import.')).toBeTruthy()
    expect(screen.getByText(/^1 row has the name and birth date of a patient already at the clinic/)).toBeTruthy()
    expect(screen.getAllByRole('listitem')).toHaveLength(1)
    expect(screen.getByRole('listitem').textContent).toMatch(/^Line 2: Ada Sample, born .*1990.*, may already be at the clinic$/)
    fireEvent.click(screen.getByRole('button', { name: 'Import 2 patients' }))
    await vi.waitFor(() => expect(onImported).toHaveBeenCalledWith(2))
    expect(requests.map((r) => new URL(r.url).search)).toEqual(['', '?save=true'])
    expect(await requests[0].text()).toMatch(/name="file"; filename="[^"]+"\r\nContent-Type: text\/csv/)
    expectCsrfOnWrites(requests)
  })

  it('lists every row with an error and imports nothing', async () => {
    const { requests } = renderImport(() =>
      Response.json({
        saved: false,
        rows: [
          { line: 2, errors: [], duplicate: false, patient: ada },
          { line: 3, errors: ['given_name_invalid', 'birth_date_invalid'], duplicate: false },
          { line: 4, errors: ['phone_invalid'], duplicate: true },
        ],
      }),
    )
    choose()
    expect(await screen.findByText('2 of 3 rows have errors. Correct them in the file and choose it again.')).toBeTruthy()
    expect(screen.getByText('Line 3: the given name is missing or longer than 100 characters; the birth date is not a valid date in the past')).toBeTruthy()
    expect(screen.getByText('Line 4: the phone is not a valid number')).toBeTruthy()
    expect(screen.queryByText(/Ada/)).toBeNull()
    expect(screen.queryByRole('button', { name: /^Import/ })).toBeNull()
    expect(requests).toHaveLength(1)
  })

  it('says why a file cannot be read', async () => {
    for (const [response, message] of [
      [apiError(400, 'invalid_import'), 'The file could not be read. Use a CSV or XLSX file with a header row naming at least the given name, family name and birth date columns, and at most 5000 rows.'],
      [apiError(413, 'too_large'), 'The file is too large.'],
      [apiError(403, 'forbidden'), "Only the clinic's admin can import patients."],
      [apiError(500, 'internal'), 'The file could not be imported. Try again.'],
    ] as const) {
      renderImport(() => response.clone())
      choose()
      expect(await screen.findByText(message)).toBeTruthy()
      cleanup()
    }
  })

  it('goes back without importing', () => {
    const { onCancel, requests } = renderImport(() => apiError(500, 'internal'))
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onCancel).toHaveBeenCalled()
    expect(requests).toHaveLength(0)
  })
})
