import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderInLanguage } from './i18n/renderInLanguage'
import type { Language } from './i18n/languages'
import { AppointmentQuestionnaires } from './AppointmentQuestionnaires'
import { newApiClient } from './api/client'
import { newFhirClient, type Questionnaire, type QuestionnaireResponse } from './api/fhir'
import { apiError, expectCsrfOnWrites, fakeFetch, fhirJSON, outcome, paths, type Route } from './test/fakeFetch'
import { appointment } from './test/fixtures'
import { questionnaireFHIR, translatedFHIR } from './test/questionnaires'
import { responseFHIR, responsesBundle } from './test/responses'

afterEach(cleanup)

const id = 'e1000000-0000-4000-8000-000000000001'
// Monday 9 November 2026, 09:00 to 09:30 in Europe/Sarajevo.
const booked = appointment(id, '2026-11-09T08:00:00Z', '2026-11-09T08:30:00Z')
const before = new Date('2026-11-09T07:00:00Z')
const after = new Date('2026-11-09T08:01:00Z')

const intake: Questionnaire = { ...questionnaireFHIR(), status: 'active' }
const followUp: Questionnaire = { ...intake, id: 'q2', url: 'https://salus.invalid/fhir/Questionnaire/s2', title: 'Placeholder follow-up', item: [] }
const third: Questionnaire = { ...intake, id: 'q3', url: 'https://salus.invalid/fhir/Questionnaire/s3', title: 'Placeholder third', item: [] }

// every is a version with one question of each type, a section, instructions and a hidden question.
const every: Questionnaire = {
  ...intake,
  id: 'q4',
  url: 'https://salus.invalid/fhir/Questionnaire/s4',
  title: 'Placeholder every type',
  item: [
    { linkId: 'intro', text: 'Placeholder instructions', type: 'display' },
    { linkId: 'short', text: 'Placeholder short', type: 'string' },
    { linkId: 'long', text: 'Placeholder long', type: 'text' },
    { linkId: 'yes', text: 'Placeholder yes or no', type: 'boolean' },
    { linkId: 'hidden', text: 'Placeholder hidden', type: 'string', enableWhen: [{ question: 'yes', operator: '=', answerBoolean: false }] },
    {
      linkId: 'sec',
      text: 'Placeholder section',
      type: 'group',
      item: [
        { linkId: 'one', text: 'Placeholder single', type: 'choice', answerOption: [{ valueString: 'Option A' }, { valueString: 'Option B' }] },
        { linkId: 'many', text: 'Placeholder multiple', type: 'choice', repeats: true, answerOption: [{ valueString: 'Option C' }, { valueString: 'Option D' }] },
        { linkId: 'whole', text: 'Placeholder whole', type: 'integer' },
        { linkId: 'dec', text: 'Placeholder decimal', type: 'decimal' },
        { linkId: 'when', text: 'Placeholder date', type: 'date' },
        { linkId: 'skipped', text: 'Placeholder optional', type: 'string' },
      ],
    },
  ],
}

const everyAnswered = responseFHIR('r4', 'completed', {
  questionnaire: `${every.url}|1`,
  authored: '2026-11-08T17:45:00Z',
  item: [
    { linkId: 'short', answer: [{ valueString: 'Placeholder short answer' }] },
    { linkId: 'long', answer: [{ valueString: 'Placeholder line one\nPlaceholder line two' }] },
    { linkId: 'yes', answer: [{ valueBoolean: true }] },
    {
      linkId: 'sec',
      item: [
        { linkId: 'one', answer: [{ valueString: 'Option B' }] },
        { linkId: 'many', answer: [{ valueString: 'Option C' }, { valueString: 'Option D' }] },
        { linkId: 'whole', answer: [{ valueInteger: 42 }] },
        { linkId: 'dec', answer: [{ valueDecimal: 36.6 }] },
        { linkId: 'when', answer: [{ valueDate: '1990-01-31' }] },
      ],
    },
  ],
})

const search = 'GET /fhir/R4/QuestionnaireResponse'
const actives = 'GET /fhir/R4/Questionnaire'
const requestPost = `POST /appointments/${id}/questionnaires`
const withdrawPost = (r: string) => `POST /questionnaire-responses/${r}/withdraw`

function json(r: string, status: QuestionnaireResponse['status'], questionnaireId = 'q2') {
  return Response.json(
    { id: r, appointmentId: id, questionnaireId, status, source: 'added', answers: [], dueAt: booked.start, lastUpdated: '2026-11-06T09:30:00Z', versionId: 2 },
    { status: status === 'in-progress' ? 201 : 200 },
  )
}

type Who = 'own' | 'admin' | 'other'

function renderSection(
  who: Who,
  responses: QuestionnaireResponse[],
  versions: Questionnaire[],
  routes: Record<string, Route> = {},
  opts: { now?: Date; appt?: typeof booked; language?: Language } = {},
) {
  const fake = fakeFetch({ [search]: () => fhirJSON(responsesBundle(responses, versions)), ...routes })
  renderInLanguage(
    <AppointmentQuestionnaires
      api={newApiClient(fake.fetch)}
      fhir={newFhirClient(fake.fetch)}
      appt={opts.appt ?? booked}
      zone="Europe/Sarajevo"
      own={who === 'own'}
      admin={who === 'admin'}
      now={opts.now ?? before}
    />,
    opts.language ?? 'en',
  )
  return fake
}

const section = async () => within(await screen.findByRole('region', { name: 'Questionnaires' }))
const rowOf = async (title: string) => within((await (await section()).findByText(title)).closest('li')!)

describe('AppointmentQuestionnaires', () => {
  it('is in the UI language, the date of a submission included', async () => {
    renderSection('own', [responseFHIR('r1', 'completed', { authored: '2026-11-08T17:45:00Z' })], [intake], {}, { language: 'sr-Latn' })

    const region = within(await screen.findByRole('region', { name: 'Upitnici' }))
    const row = within((await region.findByText('Placeholder intake')).closest('li')!)
    expect(row.getByText('Predat: nedjelja, 8. novembar 2026. u 18:45')).toBeTruthy()
    expect(row.getByRole('button', { name: 'Prikaži odgovore' })).toBeTruthy()
  })

  it('names a request whose version the search did not include by W35\'s word', async () => {
    renderSection('own', [responseFHIR('r1', 'in-progress')], [])

    expect((await rowOf('Questionnaire')).getByText('Not submitted yet')).toBeTruthy()
  })

  it("searches the appointment's responses with their versions included", async () => {
    const { requests } = renderSection('own', [], [])

    await (await section()).findByText('No questionnaires requested.')
    const url = new URL(requests[0].url)
    expect(url.searchParams.get('appointment')).toBe(`Appointment/${id}`)
    expect(url.searchParams.get('_include')).toBe('QuestionnaireResponse:questionnaire')
  })

  it.each([
    ['in-progress', before, 'Not submitted yet'],
    ['in-progress', after, 'Not submitted before the appointment'],
    ['stopped', before, 'Withdrawn'],
  ] as const)('shows a %s request as its status', async (status, now, text) => {
    renderSection('own', [responseFHIR('r1', status)], [intake], {}, { now })

    expect((await rowOf('Placeholder intake')).getByText(text)).toBeTruthy()
  })

  it('shows a submitted one with its time in the clinic zone', async () => {
    renderSection('own', [responseFHIR('r1', 'completed', { authored: '2026-11-08T17:45:00Z' })], [intake])

    expect((await rowOf('Placeholder intake')).getByText('Submitted Sunday, 8 November 2026, 18:45')).toBeTruthy()
  })

  it('says when the requests cannot be loaded', async () => {
    renderSection('own', [], [], { [search]: () => outcome(500, 'exception') })

    expect((await (await section()).findByRole('alert')).textContent).toBe('Something went wrong. Try again.')
  })

  describe('the answers', () => {
    it('are shown by section on request, each type written out, hidden items left out, unanswered as Not answered', async () => {
      renderSection('own', [everyAnswered], [every])

      const row = await rowOf('Placeholder every type')
      expect(row.queryByText('Placeholder short answer')).toBeNull()
      fireEvent.click(row.getByRole('button', { name: 'View answers' }))

      const answers = (q: string) => row.getByText(q).closest('div')!.querySelector('dd')!
      expect(answers('Placeholder short').textContent).toBe('Placeholder short answer')
      expect(answers('Placeholder long').textContent).toBe('Placeholder line one\nPlaceholder line two')
      expect(answers('Placeholder yes or no').textContent).toBe('Yes')
      expect(answers('Placeholder single').textContent).toBe('Option B')
      expect(within(answers('Placeholder multiple')).getAllByRole('listitem').map((li) => li.textContent)).toEqual(['Option C', 'Option D'])
      expect(answers('Placeholder whole').textContent).toBe('42')
      expect(answers('Placeholder decimal').textContent).toBe('36.6')
      expect(answers('Placeholder date').textContent).toBe('31 January 1990')
      expect(answers('Placeholder optional').textContent).toBe('Not answered')
      expect(row.getByRole('heading', { name: 'Placeholder section' })).toBeTruthy()
      expect(row.queryByText('Placeholder hidden')).toBeNull()
      expect(row.queryByText('Placeholder instructions')).toBeNull()

      fireEvent.click(row.getByRole('button', { name: 'Hide answers' }))
      expect(row.queryByText('Placeholder short answer')).toBeNull()
    })

    describe('answered in a translation (B9)', () => {
      const translated: Questionnaire = { ...translatedFHIR(), status: 'active' }
      const inLanguage = (language?: string) =>
        ({
          ...responseFHIR('r1', 'completed', {
            authored: '2026-11-08T17:45:00Z',
            item: [{ linkId: 'smoker', answer: [{ valueBoolean: true }] }, { linkId: 'sec', item: [{ linkId: 'one', answer: [{ valueString: 'B' }] }] }],
          }),
          language,
        }) as QuestionnaireResponse

      it('say the language, show the questions as the patient read them, and the original on the switch', async () => {
        renderSection('own', [inLanguage('en')], [translated])
        const row = await rowOf('Placeholder intake')
        fireEvent.click(row.getByRole('button', { name: 'View answers' }))

        expect(row.getByText('Filled in in English')).toBeTruthy()
        const answers = (q: string) => row.getByText(q).closest('div')!.querySelector('dd')!
        expect(answers('Placeholder single in English').textContent).toBe('B in English')
        expect(row.getByRole('heading', { name: 'Placeholder section in English' })).toBeTruthy()
        const original = row.getByRole('switch', { name: 'Show the original questions' }) as HTMLInputElement
        expect(original.checked).toBe(false)

        fireEvent.click(original)
        expect(answers('Placeholder single').textContent).toBe('B')
        expect(row.getByRole('heading', { name: 'Placeholder section' })).toBeTruthy()
        expect(row.queryByText('Placeholder single in English')).toBeNull()
      })

      it('say the language without a switch when the patient answered in the original', async () => {
        renderSection('own', [inLanguage('bs')], [translated])
        const row = await rowOf('Placeholder intake')
        fireEvent.click(row.getByRole('button', { name: 'View answers' }))

        expect(row.getByText('Filled in in Bosanski')).toBeTruthy()
        expect(row.queryByRole('switch')).toBeNull()
        expect(row.getByText('Placeholder single')).toBeTruthy()
      })

      it('are in the UI language', async () => {
        renderSection('own', [inLanguage('en')], [translated], {}, { language: 'hr' })
        const row = within((await screen.findByText('Placeholder intake')).closest('li')!)
        fireEvent.click(row.getByRole('button', { name: 'Prikaži odgovore' }))

        expect(row.getByText('Ispunjeno na jeziku: English')).toBeTruthy()
        expect(row.getByRole('switch', { name: 'Prikaži izvorna pitanja' })).toBeTruthy()
      })
    })

    it('list the files of a file upload item, each to open, and say Not answered without one (W7, W8)', async () => {
      const uploads: Questionnaire = {
        ...intake,
        id: 'q5',
        url: 'https://salus.invalid/fhir/Questionnaire/s5',
        title: 'Placeholder uploads',
        item: [
          { linkId: 'files', text: 'Placeholder files', type: 'attachment', repeats: true },
          { linkId: 'one', text: 'Placeholder one file', type: 'attachment' },
          { linkId: 'none', text: 'Placeholder no file', type: 'attachment' },
        ],
      }
      const file = (n: number, contentType: string, title: string, size: number) => ({
        valueAttachment: { contentType, size, title, hash: 'AAAA', url: `https://api.example.test/binaries/b0000000-0000-4000-8000-00000000000${n}/content` },
      })
      renderSection('own', [
        responseFHIR('r5', 'completed', {
          questionnaire: `${uploads.url}|1`,
          authored: '2026-11-08T17:45:00Z',
          item: [
            { linkId: 'files', answer: [file(1, 'image/png', 'placeholder.png', 3000), file(2, 'video/mp4', 'placeholder.mp4', 12_300_000)] },
            { linkId: 'one', answer: [file(3, 'application/pdf', 'placeholder.pdf', 640_000)] },
          ],
        }),
      ], [uploads])

      const row = await rowOf('Placeholder uploads')
      fireEvent.click(row.getByRole('button', { name: 'View answers' }))
      const answers = (q: string) => within(row.getByText(q).closest('div')!.querySelector('dd')!)
      expect(answers('Placeholder files').getByText('2 files')).toBeTruthy()
      expect(answers('Placeholder files').getAllByRole('listitem').map((li) => li.querySelector('.file-details')?.textContent)).toEqual(['Photo, 3 KB', 'Video, 12.3 MB'])
      expect(answers('Placeholder one file').getByText('1 file')).toBeTruthy()
      expect(answers('Placeholder one file').getByText('placeholder.pdf')).toBeTruthy()
      expect(answers('Placeholder no file').getByText('Not answered')).toBeTruthy()

      fireEvent.click(answers('Placeholder files').getAllByRole('button', { name: 'Open' })[1])
      const v = screen.getByRole('dialog').querySelector('video') as HTMLVideoElement
      expect(v.getAttribute('src')).toBe(`${location.origin}/binaries/b0000000-0000-4000-8000-000000000002/content`)
    })

    it('are not offered before the patient submits, nor once withdrawn', async () => {
      renderSection('own', [responseFHIR('r1', 'in-progress'), responseFHIR('r2', 'stopped', { questionnaire: `${followUp.url}|1` })], [intake, followUp])

      await rowOf('Placeholder intake')
      expect((await section()).queryByRole('button', { name: 'View answers' })).toBeNull()
    })
  })

  describe('for an admin', () => {
    it('shows the requests and their status only, with W39 on the status', async () => {
      renderSection('admin', [responseFHIR('r1', 'completed', { authored: '2026-11-08T17:45:00Z' }), responseFHIR('r2', 'in-progress', { questionnaire: `${followUp.url}|1` })], [intake, followUp])

      const row = await rowOf('Placeholder intake')
      const status = row.getByText('Submitted Sunday, 8 November 2026, 18:45')
      const tip = row.getByRole('tooltip')
      expect(tip.textContent).toBe("Only the appointment's practitioner can see the answers.")
      expect(status.getAttribute('aria-describedby')).toBe(tip.id)
      expect(status.tabIndex).toBe(0)
      expect((await section()).queryByRole('button')).toBeNull()
    })

    it('shows the status of a Name view of a submitted response, never answers', async () => {
      renderSection('admin', [responseFHIR('r1', 'completed', { questionnaire: `${every.url}|1`, authored: '2026-11-08T17:45:00Z' })], [every])

      const row = await rowOf('Placeholder every type')
      expect(row.getByText('Submitted Sunday, 8 November 2026, 18:45')).toBeTruthy()
      expect(row.queryByText('Not answered')).toBeNull()
      expect(row.queryByRole('button')).toBeNull()
    })
  })

  describe('adding', () => {
    const activeBundle = () => fhirJSON(responsesBundle([], [intake, followUp, third]))

    it('offers the active questionnaires not already requested, and requests the one picked with the CSRF header', async () => {
      const { requests } = renderSection('own', [responseFHIR('r1', 'in-progress'), responseFHIR('r2', 'stopped', { questionnaire: `${third.url}|1` })], [intake, third], {
        [actives]: activeBundle,
        [requestPost]: () => json('r9', 'in-progress', 'q2'),
      })

      fireEvent.click(await (await section()).findByRole('button', { name: 'Add questionnaire' }))
      const picker = (await screen.findByLabelText('Questionnaire')) as HTMLSelectElement
      expect(new URL(requests.find((r) => new URL(r.url).pathname === '/fhir/R4/Questionnaire')!.url).searchParams.get('status')).toBe('active')
      expect(Array.from(picker.options).map((o) => o.textContent)).toEqual(['Placeholder follow-up', 'Placeholder third'])
      expect(screen.getByText('The patient sees it in the Salus app and is asked to submit it before the appointment starts.')).toBeTruthy()
      fireEvent.change(picker, { target: { value: 'q2' } })
      fireEvent.click(screen.getByRole('button', { name: 'Add' }))

      expect((await rowOf('Placeholder follow-up')).getByText('Not submitted yet')).toBeTruthy()
      expect(await requests.find((r) => r.method === 'POST')!.json()).toEqual({ questionnaireId: 'q2' })
      expectCsrfOnWrites(requests)
      expect(screen.queryByLabelText('Questionnaire')).toBeNull()
    })

    it('is cancelled without a request', async () => {
      const { requests } = renderSection('own', [], [], { [actives]: activeBundle })

      fireEvent.click(await (await section()).findByRole('button', { name: 'Add questionnaire' }))
      await screen.findByLabelText('Questionnaire')
      fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))

      expect(screen.queryByLabelText('Questionnaire')).toBeNull()
      expect(requests.filter((r) => r.method === 'POST')).toHaveLength(0)
    })

    it.each([
      ['already_requested', 'This questionnaire has already been requested for this appointment.'],
      ['not_active', 'This questionnaire is no longer active.'],
      ['appointment_started', 'The appointment has started, so questionnaires can no longer be added.'],
      ['consent_required', "The patient's consent for this clinic is not active, so a questionnaire cannot be requested."],
      ['appointment_closed', 'Something went wrong. Try again.'],
    ])('maps 409 %s to a message', async (code, message) => {
      renderSection('own', [], [], { [actives]: activeBundle, [requestPost]: () => apiError(409, code) })

      fireEvent.click(await (await section()).findByRole('button', { name: 'Add questionnaire' }))
      await screen.findByLabelText('Questionnaire')
      fireEvent.click(screen.getByRole('button', { name: 'Add' }))

      await vi.waitFor(() => expect(screen.getByRole('alert').textContent).toBe(message))
      expect(screen.getByText('No questionnaires requested.')).toBeTruthy()
    })

    it.each([
      ['an admin', 'admin', before, booked],
      ['another practitioner', 'other', before, booked],
      ['the practitioner once it has started', 'own', after, booked],
      ['the practitioner once it is cancelled', 'own', before, { ...booked, status: 'cancelled' as const }],
    ] as const)('is not offered to %s', async (_, who, now, appt) => {
      renderSection(who, [], [], {}, { now, appt })

      await (await section()).findByText('No questionnaires requested.')
      expect(screen.queryByRole('button', { name: 'Add questionnaire' })).toBeNull()
    })
  })

  describe('withdrawing', () => {
    it('asks first, then withdraws with the CSRF header', async () => {
      const { requests } = renderSection('own', [responseFHIR('r1', 'in-progress')], [intake], { [withdrawPost('r1')]: () => json('r1', 'stopped') })

      fireEvent.click((await rowOf('Placeholder intake')).getByRole('button', { name: 'Withdraw' }))
      expect(screen.getByText('Withdraw this questionnaire?')).toBeTruthy()
      expect(screen.getByText('The patient no longer needs to fill it in, and any answers they have saved are deleted.')).toBeTruthy()
      expect(requests.filter((r) => r.method === 'POST')).toHaveLength(0)
      fireEvent.click(within(screen.getByText('Withdraw this questionnaire?').closest('.confirm') as HTMLElement).getByRole('button', { name: 'Withdraw' }))

      expect(await (await rowOf('Placeholder intake')).findByText('Withdrawn')).toBeTruthy()
      expect(paths(requests)).toContain(withdrawPost('r1'))
      expectCsrfOnWrites(requests)
      expect(screen.queryByRole('button', { name: 'Withdraw' })).toBeNull()
    })

    it('asks in an alert dialog that takes the focus, and gives it back to Withdraw on Cancel', async () => {
      renderSection('own', [responseFHIR('r1', 'in-progress')], [intake])

      fireEvent.click((await rowOf('Placeholder intake')).getByRole('button', { name: 'Withdraw' }))
      const dialog = screen.getByRole('alertdialog', { name: 'Withdraw this questionnaire?' })
      const described = document.getElementById(dialog.getAttribute('aria-describedby') ?? '')
      expect(described?.textContent).toBe('The patient no longer needs to fill it in, and any answers they have saved are deleted.')
      expect(dialog.contains(document.activeElement)).toBe(true)

      fireEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }))
      expect(document.activeElement).toBe((await rowOf('Placeholder intake')).getByRole('button', { name: 'Withdraw' }))
    })

    it('keeps the request when not confirmed', async () => {
      const { requests } = renderSection('own', [responseFHIR('r1', 'in-progress')], [intake])

      fireEvent.click((await rowOf('Placeholder intake')).getByRole('button', { name: 'Withdraw' }))
      fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))

      expect(screen.queryByText('Withdraw this questionnaire?')).toBeNull()
      expect(requests.filter((r) => r.method === 'POST')).toHaveLength(0)
    })

    it('says when the patient has already submitted it', async () => {
      renderSection('own', [responseFHIR('r1', 'in-progress')], [intake], { [withdrawPost('r1')]: () => apiError(409, 'submitted') })

      fireEvent.click((await rowOf('Placeholder intake')).getByRole('button', { name: 'Withdraw' }))
      fireEvent.click(within(screen.getByText('Withdraw this questionnaire?').closest('.confirm') as HTMLElement).getByRole('button', { name: 'Withdraw' }))

      await vi.waitFor(() => expect(screen.getByRole('alert').textContent).toBe('The patient has already submitted this questionnaire.'))
      expect(document.activeElement).toBe((await rowOf('Placeholder intake')).getByRole('button', { name: 'Withdraw' }))
    })

    it.each([
      ['submitted', 'completed', 'own'],
      ['withdrawn', 'stopped', 'own'],
      ['in progress, to an admin', 'in-progress', 'admin'],
    ] as const)('is not offered on one %s', async (_, status, who) => {
      renderSection(who, [responseFHIR('r1', status, { authored: '2026-11-08T17:45:00Z' })], [intake])

      await rowOf('Placeholder intake')
      expect(screen.queryByRole('button', { name: 'Withdraw' })).toBeNull()
    })
  })
})
