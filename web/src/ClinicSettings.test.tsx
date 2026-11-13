import { cleanup, fireEvent, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ClinicSettings } from './ClinicSettings'
import { newApiClient } from './api/client'
import { cityURL, contributesURL, defaultLanguageURL, listedURL, newFhirClient, timeZoneURL, transcriptCopyURL, type Organization } from './api/fhir'
import type { Language } from './i18n/languages'
import { renderInLanguage } from './i18n/renderInLanguage'
import { expectCsrfOnWrites, fakeFetch, fhirJSON, outcome, type Route } from './test/fakeFetch'

afterEach(cleanup)

const id = 'a0000000-0000-4000-8000-00000000000a'
const path = `/fhir/R4/Organization/${id}`
const org: Organization = {
  resourceType: 'Organization',
  id,
  name: 'Synthetic clinic',
  active: true,
  meta: { versionId: '3', lastUpdated: '2026-11-04T09:30:00Z' },
  extension: [
    { url: listedURL, valueBoolean: false },
    { url: timeZoneURL, valueCode: 'Europe/Sarajevo' },
  ],
}

// places are a few of GET /places.
const places = {
  places: [
    { id: 'BA-BIH', kind: 'entity', nameLatn: 'Federacija Bosne i Hercegovine', nameCyrl: 'Федерација Босне и Херцеговине' },
    { id: 'BA-09', kind: 'region', parent: 'BA-BIH', nameLatn: 'Kanton Sarajevo', nameCyrl: 'Кантон Сарајево' },
    { id: 'BA-SRP', kind: 'region', nameLatn: 'Republika Srpska', nameCyrl: 'Република Српска' },
    { id: 'Q1270190', kind: 'city', parent: 'BA-09', nameLatn: 'Centar', nameCyrl: 'Центар' },
    { id: 'Q12756621', kind: 'city', parent: 'BA-09', nameLatn: 'Ilidža', nameCyrl: 'Илиџа' },
    { id: 'Q1', kind: 'city', parent: 'BA-SRP', nameLatn: 'Banja Luka', nameCyrl: 'Бања Лука' },
  ],
}

const appointmentSettings = {
  reminder_offsets_minutes: [1440, 60],
  sms_reminders: false,
  reschedule_notice_hours: 24,
  reschedule_max: 2,
  waitlist_enabled: false,
  waitlist_hold_minutes: 15,
  noshow_flag_threshold: 0,
  noshow_flag_window_days: 180,
}

function renderSettings(routes: Record<string, Route>, language: Language = 'en', platform = true) {
  // The no-show flag's section (ClinicSettingsNoShows) reads the clinic's appointment settings beside the form.
  const fake = fakeFetch({ 'GET /places': () => Response.json(places), 'GET /clinic/appointment-settings': () => Response.json(appointmentSettings), ...routes })
  renderInLanguage(
    <ClinicSettings api={newApiClient(fake.fetch)} fhir={newFhirClient(fake.fetch)} organizationId={id} platform={platform} />,
    language,
  )
  return fake
}

async function rename(name: string) {
  fireEvent.change(await screen.findByLabelText('Clinic name'), { target: { value: name } })
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
}

describe('ClinicSettings', () => {
  it('shows that the clinic is loading', () => {
    renderSettings({ [`GET ${path}`]: () => new Promise<Response>(() => {}) })

    expect(screen.getByText('Loading')).toBeTruthy()
  })

  it('shows the clinic name from the Organization', async () => {
    renderSettings({ [`GET ${path}`]: () => fhirJSON(org) })

    expect(((await screen.findByLabelText('Clinic name')) as HTMLInputElement).value).toBe('Synthetic clinic')
  })

  it('saves with If-Match of the version read, then of the version saved', async () => {
    let version = 3
    const { requests } = renderSettings({
      [`GET ${path}`]: () => fhirJSON(org),
      [`PUT ${path}`]: async (r) => {
        version++
        const body = (await r.json()) as Organization
        return fhirJSON({ ...org, name: body.name, meta: { ...org.meta, versionId: String(version) } })
      },
    })

    const puts = () => requests.filter((r) => r.method === 'PUT')

    await rename('Renamed clinic')
    expect((await screen.findByRole('status')).textContent).toBe('Saved.')
    await rename('Renamed again')

    await vi.waitFor(() => expect(puts()).toHaveLength(2))
    expect(puts()[0].headers.get('If-Match')).toBe('W/"3"')
    expect(((await puts()[0].json()) as Organization).name).toBe('Renamed clinic')
    expect(puts()[1].headers.get('If-Match')).toBe('W/"4"')
    expectCsrfOnWrites(requests)
  })

  it.each([
    [412, 'conflict', 'Someone else changed the clinic settings. Reload to see their changes.'],
    [422, 'business-rule', 'Enter a name of 1 to 200 characters.'],
    [403, 'forbidden', 'Only a clinic admin can change these settings.'],
    [500, 'exception', 'Something went wrong. Try again.'],
  ])('maps %i %s to a message', async (status, code, message) => {
    renderSettings({ [`GET ${path}`]: () => fhirJSON(org), [`PUT ${path}`]: () => outcome(status, code) })

    await rename('Renamed clinic')

    expect((await screen.findByRole('alert')).textContent).toBe(message)
  })

  it('reloads the clinic after a conflict', async () => {
    let reads = 0
    renderSettings({
      [`GET ${path}`]: () => {
        reads++
        return fhirJSON(reads === 1 ? org : { ...org, name: 'Changed elsewhere', meta: { ...org.meta, versionId: '5' } })
      },
      [`PUT ${path}`]: () => outcome(412, 'conflict'),
    })

    await rename('Renamed clinic')
    fireEvent.click(await screen.findByRole('button', { name: 'Reload' }))

    await vi.waitFor(async () =>
      expect(((await screen.findByLabelText('Clinic name')) as HTMLInputElement).value).toBe('Changed elsewhere'),
    )
  })

  it('says when the clinic cannot be loaded', async () => {
    renderSettings({ [`GET ${path}`]: () => outcome(404, 'not-found') })

    expect((await screen.findByRole('alert')).textContent).toBe('The clinic could not be loaded. Reload the page to try again.')
  })

  it('shows whether the clinic is listed and its time zone', async () => {
    renderSettings({ [`GET ${path}`]: () => fhirJSON(org) })

    expect(((await screen.findByLabelText('List the clinic in the patient app')) as HTMLInputElement).checked).toBe(false)
    expect((screen.getByLabelText('Time zone') as HTMLSelectElement).value).toBe('Europe/Sarajevo')
  })

  it('saves listed, the time zone and the default language as the Organization extensions with If-Match', async () => {
    const { requests } = renderSettings({
      [`GET ${path}`]: () => fhirJSON(org),
      [`PUT ${path}`]: async (r) => fhirJSON({ ...((await r.json()) as Organization), meta: { ...org.meta, versionId: '4' } }),
    })

    fireEvent.click(await screen.findByLabelText('List the clinic in the patient app'))
    fireEvent.change(screen.getByLabelText('Time zone'), { target: { value: 'Europe/Berlin' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect((await screen.findByRole('status')).textContent).toBe('Saved.')
    const put = requests.find((r) => r.method === 'PUT')!
    expect(put.headers.get('If-Match')).toBe('W/"3"')
    expect(((await put.json()) as Organization).extension).toEqual([
      { url: listedURL, valueBoolean: true },
      { url: timeZoneURL, valueCode: 'Europe/Berlin' },
      { url: defaultLanguageURL, valueCode: 'bs' },
      { url: transcriptCopyURL, valueBoolean: false },
      { url: contributesURL, valueBoolean: true },
    ])
    expectCsrfOnWrites(requests)
    expect((screen.getByLabelText('List the clinic in the patient app') as HTMLInputElement).checked).toBe(true)
  })

  it('maps an unknown time zone to a message', async () => {
    renderSettings({ [`GET ${path}`]: () => fhirJSON(org), [`PUT ${path}`]: () => outcome(400, 'value') })

    await rename('Renamed clinic')

    expect((await screen.findByRole('alert')).textContent).toBe('Choose a time zone from the list.')
  })
})

describe("the clinic's default language", () => {
  it('shows it among the staff languages, each by its own name', async () => {
    renderSettings({ [`GET ${path}`]: () => fhirJSON({ ...org, extension: [...org.extension!, { url: defaultLanguageURL, valueCode: 'sr-Cyrl' }] }) })

    const select = (await screen.findByLabelText('Default language')) as HTMLSelectElement
    expect(select.value).toBe('sr-Cyrl')
    expect([...select.options].map((o) => o.textContent)).toEqual(['Bosanski', 'Hrvatski', 'Srpski (latinica)', 'Српски (ћирилица)', 'English'])
    expect(
      screen.getByText('Used for e-mails to staff who have not chosen a language, and as the original language of new questionnaires.'),
    ).toBeTruthy()
  })

  it('saves it with the other settings', async () => {
    const { requests } = renderSettings({
      [`GET ${path}`]: () => fhirJSON(org),
      [`PUT ${path}`]: async (r) => fhirJSON({ ...((await r.json()) as Organization), meta: { ...org.meta, versionId: '4' } }),
    })

    expect(((await screen.findByLabelText('Default language')) as HTMLSelectElement).value).toBe('bs')
    fireEvent.change(screen.getByLabelText('Default language'), { target: { value: 'en' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect((await screen.findByRole('status')).textContent).toBe('Saved.')
    const put = (await requests.find((r) => r.method === 'PUT')!.json()) as Organization
    expect(put.extension).toContainEqual({ url: defaultLanguageURL, valueCode: 'en' })
  })

  it('is in the language of the admin', async () => {
    renderSettings({ [`GET ${path}`]: () => fhirJSON(org) }, 'hr')

    expect(await screen.findByRole('heading', { name: 'Postavke klinike' })).toBeTruthy()
    expect(screen.getByLabelText('Zadani jezik')).toBeTruthy()
    expect(screen.getByLabelText('Naziv klinike')).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Spremi' })).toBeTruthy()
  })
})

describe("the patients' transcript copy", () => {
  const label = 'Give patients a copy of their consultation transcripts'

  it('is off unless the Organization says it is on', async () => {
    renderSettings({ [`GET ${path}`]: () => fhirJSON(org) })

    expect(((await screen.findByLabelText(label)) as HTMLInputElement).checked).toBe(false)
    expect(screen.getByText('Patients can then read the transcripts of their own consultations in the patient app.')).toBeTruthy()
  })

  it('saves it with the other settings', async () => {
    const { requests } = renderSettings({
      [`GET ${path}`]: () => fhirJSON(org),
      [`PUT ${path}`]: async (r) => fhirJSON({ ...((await r.json()) as Organization), meta: { ...org.meta, versionId: '4' } }),
    })

    fireEvent.click(await screen.findByLabelText(label))
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect((await screen.findByRole('status')).textContent).toBe('Saved.')
    const put = requests.find((r) => r.method === 'PUT')!
    expect(((await put.json()) as Organization).extension).toContainEqual({ url: transcriptCopyURL, valueBoolean: true })
    expect((screen.getByLabelText(label) as HTMLInputElement).checked).toBe(true)
  })

  it('shows it in Bosnian', async () => {
    renderSettings({ [`GET ${path}`]: () => fhirJSON({ ...org, extension: [...org.extension!, { url: transcriptCopyURL, valueBoolean: true }] }) }, 'bs')

    expect(((await screen.findByLabelText('Omogući pacijentima kopiju transkripta njihovih konsultacija')) as HTMLInputElement).checked).toBe(true)
  })
})

describe("the clinic's location and its contribution to the platform figures", () => {
  const located: Organization = {
    ...org,
    extension: [...org.extension!, { url: contributesURL, valueBoolean: true }],
    address: [{ state: 'BA-09', city: 'Centar', extension: [{ url: cityURL, valueCode: 'Q1270190' }] }],
  }

  it('shows the region and the city from the address, the cities of the region only', async () => {
    renderSettings({ [`GET ${path}`]: () => fhirJSON(located) })

    await vi.waitFor(() => expect((screen.getByLabelText('Region') as HTMLSelectElement).value).toBe('BA-09'))
    const city = screen.getByLabelText('City') as HTMLSelectElement
    expect(city.value).toBe('Q1270190')
    expect([...city.options].map((o) => o.textContent)).toEqual(['Choose a city', 'Centar', 'Ilidža'])
  })

  it('saves a new location as the address with the city id, and the opt-out', async () => {
    const { requests } = renderSettings({
      [`GET ${path}`]: () => fhirJSON(org),
      [`PUT ${path}`]: async (r) => fhirJSON({ ...((await r.json()) as Organization), meta: { ...org.meta, versionId: '4' } }),
    })

    await vi.waitFor(() => expect((screen.getByLabelText('Region') as HTMLSelectElement).options.length).toBeGreaterThan(1))
    fireEvent.change(screen.getByLabelText('Region'), { target: { value: 'BA-SRP' } })
    fireEvent.change(screen.getByLabelText('City'), { target: { value: 'Q1' } })
    const contributes = screen.getByLabelText('Contribute to the All of Salus figures') as HTMLInputElement
    expect(contributes.checked).toBe(true)
    fireEvent.click(contributes)
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect((await screen.findByRole('status')).textContent).toBe('Saved.')
    const body = (await requests.find((r) => r.method === 'PUT')!.json()) as Organization
    expect(body.address).toEqual([{ state: 'BA-SRP', city: 'Banja Luka', extension: [{ url: cityURL, valueCode: 'Q1' }] }])
    expect(body.extension).toContainEqual({ url: contributesURL, valueBoolean: false })
  })

  it('explains what opting out means', async () => {
    renderSettings({ [`GET ${path}`]: () => fhirJSON(located) })

    expect(
      await screen.findByText(
        "The clinic's diagnoses and appointments count, anonymized, in the figures across all clinics. If you opt out, the clinic is left out from the next build on, and its staff no longer see these figures.",
      ),
    ).toBeTruthy()
  })

  it('maps a city outside its region to a message', async () => {
    renderSettings({ [`GET ${path}`]: () => fhirJSON(located), [`PUT ${path}`]: () => outcome(400, 'code-invalid') })

    await rename('Renamed clinic')

    expect((await screen.findByRole('alert')).textContent).toBe('Choose a city of the region.')
  })

  it('hides the opt-out and the location hint while the All of Salus view is off', async () => {
    renderSettings({ [`GET ${path}`]: () => fhirJSON(located) }, 'en', false)

    expect(await screen.findByLabelText('Region')).toBeTruthy()
    expect(screen.queryByLabelText('Contribute to the All of Salus figures')).toBeNull()
    expect(screen.queryByText(/The All of Salus figures group clinics/)).toBeNull()
  })

  it('refuses to save a region without a city', async () => {
    const { requests } = renderSettings({ [`GET ${path}`]: () => fhirJSON(located) })

    await vi.waitFor(() => expect((screen.getByLabelText('Region') as HTMLSelectElement).value).toBe('BA-09'))
    fireEvent.change(screen.getByLabelText('Region'), { target: { value: 'BA-SRP' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect((await screen.findByRole('alert')).textContent).toBe('Choose a city of the region.')
    expect(requests.filter((r) => r.method === 'PUT')).toHaveLength(0)
  })

  it('does not offer to clear a location once set', async () => {
    renderSettings({ [`GET ${path}`]: () => fhirJSON(located) })

    await vi.waitFor(() => expect((screen.getByLabelText('Region') as HTMLSelectElement).value).toBe('BA-09'))
    const none = [...(screen.getByLabelText('Region') as HTMLSelectElement).options].find((o) => o.value === '')!
    expect(none.disabled).toBe(true)
  })

  it('says when the regions and cities cannot be loaded', async () => {
    renderSettings({ 'GET /places': () => new Response(null, { status: 500 }), [`GET ${path}`]: () => fhirJSON(located) })

    expect((await screen.findByRole('alert')).textContent).toBe('The regions and cities could not be loaded. Reload the page to try again.')
  })

  it('is in Serbian Cyrillic', async () => {
    renderSettings({ [`GET ${path}`]: () => fhirJSON(located) }, 'sr-Cyrl')

    expect(await screen.findByLabelText('Регија')).toBeTruthy()
    expect(screen.getByLabelText('Град')).toBeTruthy()
  })
})
