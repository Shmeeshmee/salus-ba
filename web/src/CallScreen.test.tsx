import { act, cleanup, fireEvent, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderInLanguage } from './i18n/renderInLanguage'
import type { Language } from './i18n/languages'
import { CallScreen } from './CallScreen'
import { newApiClient } from './api/client'
import type { Opened } from './call'
import { apiError, expectCsrfOnWrites, fakeFetch, paths, type Route } from './test/fakeFetch'
import { fakeMedia } from './test/fakeCall'
import { fakeActions, fakeLive, segment } from './test/fakeConsult'

afterEach(cleanup)

const id = 'e1000000-0000-4000-8000-000000000001'
const joinPost = `POST /appointments/${id}/consultation/join`
const endPost = `POST /appointments/${id}/consultation/end`
const statusGet = `GET /appointments/${id}/consultation`

// A synthetic join answer: an invented name and a token that is not one.
const joined = { url: 'ws://livekit.example.test', token: 'synthetic-token', expiresAt: '2026-11-09T08:06:00Z', peerName: 'Ada Example', role: 'practitioner' }
const open = { state: 'open', practitionerJoinedAt: '2026-11-09T08:01:00Z' }

type Setup = {
  routes?: Record<string, Route>
  camera?: Opened['problem']
  microphone?: Opened['problem']
  level?: number | null
  connectFails?: boolean
  language?: Language
  live?: ReturnType<typeof fakeLive>
}

function renderCall({ routes = {}, language = 'en', live, ...options }: Setup = {}) {
  const fake = fakeFetch({ [joinPost]: () => Response.json(joined), [statusGet]: () => Response.json(open), ...routes })
  const media = fakeMedia(options)
  const consult = live && { live: live.live, actions: fakeActions().actions }
  renderInLanguage(<CallScreen api={newApiClient(fake.fetch)} id={id} media={media.media} consult={consult} />, language)
  return { ...fake, ...media }
}

const button = (name: string) => screen.getByRole('button', { name })
const peer = (present: boolean, camera = true, microphone = true) => ({ type: 'peer', peer: { present, camera, microphone } }) as const

// inCall joins from the device check and waits for the room.
async function inCall(setup: Setup = {}) {
  const call = renderCall(setup)
  fireEvent.click(await screen.findByRole('button', { name: 'Join' }))
  await screen.findByRole('button', { name: 'Leave' })
  return call
}

// levelPolling waits until the microphone level's interval is set. The meter can render before React runs the
// effect that sets it, so advancing the fake clock at once could pass the 5 seconds with nothing polling.
const levelPolling = () => vi.waitFor(() => expect(vi.getTimerCount()).toBeGreaterThan(0))

describe('CallScreen: in the UI language', () => {
  it('checks the devices, waits for the patient and controls the call in the language', async () => {
    const { last } = renderCall({ language: 'bs' })

    expect(await screen.findByRole('heading', { name: 'Provjerite kameru i mikrofon' })).toBeTruthy()
    fireEvent.click(await screen.findByRole('button', { name: 'Pridruži se' }))
    expect(await screen.findByRole('heading', { name: 'Čeka se pacijent' })).toBeTruthy()
    expect(screen.getByText('Ada Example još nije u konsultaciji.')).toBeTruthy()
    act(() => last().emit(peer(true, false, false)))
    expect(screen.getByText('Ada Example ima isključenu kameru')).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Završi konsultaciju' })).toBeTruthy()
  })
})

describe('CallScreen: the device check', () => {
  it('previews the camera, measures the microphone and offers both pickers', async () => {
    const { previews } = renderCall()

    expect(await screen.findByRole('heading', { name: 'Check your camera and microphone' })).toBeTruthy()
    expect(screen.getByText('The patient will see and hear you once you join.')).toBeTruthy()
    const camera = (await screen.findByRole('combobox', { name: 'Camera' })) as HTMLSelectElement
    expect([...camera.options].map((o) => o.textContent)).toEqual(['Synthetic camera', 'Second synthetic camera'])
    expect(screen.getByRole('combobox', { name: 'Microphone' })).toBeTruthy()
    await vi.waitFor(() => expect(previews.find((p) => p.kind === 'camera')?.attached).toBeGreaterThan(0))
    expect(screen.getByRole('meter', { name: 'Microphone' })).toBeTruthy()
  })

  it('offers Join only once the camera and the microphone have opened or failed', async () => {
    renderCall({ camera: 'denied' })

    expect((button('Join') as HTMLButtonElement).disabled).toBe(true)
    await screen.findByRole('meter', { name: 'Microphone' })
    expect((button('Join') as HTMLButtonElement).disabled).toBe(false)
  })

  it('opens the camera picked, and stops the one before', async () => {
    const { opened, previews } = renderCall()

    fireEvent.change(await screen.findByRole('combobox', { name: 'Camera' }), { target: { value: 'cam-2' } })

    await vi.waitFor(() => expect(opened).toContainEqual({ kind: 'camera', deviceId: 'cam-2' }))
    expect(previews.filter((p) => p.kind === 'camera')[0].stopped).toBe(true)
  })

  it('says when no sound reaches the microphone for 5 seconds', async () => {
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] })
    try {
      renderCall({ level: 0 })
      await screen.findByRole('meter', { name: 'Microphone' })
      await levelPolling()
      expect(screen.queryByText('No sound detected. Check that your microphone is on and not muted.')).toBeNull()

      act(() => vi.advanceTimersByTime(5_000))

      expect(screen.getByText('No sound detected. Check that your microphone is on and not muted.')).toBeTruthy()
    } finally {
      vi.useRealTimers()
    }
  })

  it('does not take a level it cannot measure for silence', async () => {
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] })
    try {
      renderCall({ level: null })
      await screen.findByRole('meter', { name: 'Microphone' })
      await levelPolling()

      act(() => vi.advanceTimersByTime(10_000))

      expect(screen.queryByText('No sound detected. Check that your microphone is on and not muted.')).toBeNull()
    } finally {
      vi.useRealTimers()
    }
  })

  it('still joins with audio when the browser denies the camera', async () => {
    const { last, requests } = renderCall({ camera: 'denied' })

    expect(
      await screen.findByText('Your browser is not allowing the camera. Allow it in the site settings, then reload the page. You can still join without video.'),
    ).toBeTruthy()
    const off = screen.getByLabelText('Join with camera off') as HTMLInputElement
    expect(off.checked && off.disabled).toBe(true)
    fireEvent.click(button('Join'))

    await screen.findByRole('button', { name: 'Leave' })
    expect(last().settings).toEqual({ camera: false, microphone: true, microphoneId: 'mic-1' })
    expect((button('Turn camera on') as HTMLButtonElement).disabled).toBe(true)
    expectCsrfOnWrites(requests)
  })

  it('still joins, hearing and seeing the other side, when the browser denies the microphone', async () => {
    const { last } = renderCall({ microphone: 'denied' })

    expect(
      await screen.findByText(
        'Your browser is not allowing the microphone. Allow it in the site settings, then reload the page. Without it, the patient cannot hear you.',
      ),
    ).toBeTruthy()
    fireEvent.click(button('Join'))

    await screen.findByRole('button', { name: 'Leave' })
    expect(last().settings).toEqual({ camera: true, microphone: false, cameraId: 'cam-1' })
  })

  it('says when there is no camera or no microphone', async () => {
    renderCall({ camera: 'missing', microphone: 'missing' })

    expect(await screen.findByText('No camera found.')).toBeTruthy()
    expect(screen.getByText('No microphone found.')).toBeTruthy()
  })

  it('joins with the camera off when asked', async () => {
    const { last } = renderCall()

    fireEvent.click(await screen.findByLabelText('Join with camera off'))
    fireEvent.click(button('Join'))

    await screen.findByRole('button', { name: 'Leave' })
    expect(last().settings).toEqual({ camera: false, microphone: true, cameraId: 'cam-1', microphoneId: 'mic-1' })
    expect(button('Turn camera on')).toBeTruthy()
  })
})

describe('CallScreen: joining', () => {
  it('asks the API for a token with the CSRF header and connects with it', async () => {
    let answer: (r: Response) => void = () => {}
    const { requests, last } = renderCall({ routes: { [joinPost]: () => new Promise<Response>((r) => (answer = r)) } })

    fireEvent.click(await screen.findByRole('button', { name: 'Join' }))

    expect(((await screen.findByRole('button', { name: 'Joining…' })) as HTMLButtonElement).disabled).toBe(true)
    answer(Response.json(joined))
    await screen.findByRole('button', { name: 'Leave' })
    expect(paths(requests)).toContain(joinPost)
    expectCsrfOnWrites(requests)
    expect(last().url).toBe(joined.url)
    expect(last().token).toBe(joined.token)
    // The room is shown from an effect of the call's stage, which React can run after Leave renders.
    await vi.waitFor(() => expect(last().shown).toBe(true))
  })

  it.each([
    [409, 'join_window_not_open', 'This consultation opens 10 minutes before the start.'],
    [409, 'consultation_ended', 'This consultation has ended.'],
    [409, 'practitioner_did_not_join', 'The time to join this consultation has passed.'],
    [409, 'consent_required', "The patient's consent for this clinic is not active, so the consultation cannot take place."],
    [404, 'not_found', 'This consultation is not available.'],
    [500, 'internal error', 'Something went wrong. Try again.'],
  ])('shows the message for a %i %s and connects nothing', async (status, code, message) => {
    const { rooms } = renderCall({ routes: { [joinPost]: () => apiError(status, code) } })

    fireEvent.click(await screen.findByRole('button', { name: 'Join' }))

    await vi.waitFor(() => expect(screen.getByRole('alert').textContent).toBe(message))
    expect(rooms).toHaveLength(0)
    expect(button('Join')).toBeTruthy()
  })

  it('says when the video connection cannot be started', async () => {
    renderCall({ connectFails: true })

    fireEvent.click(await screen.findByRole('button', { name: 'Join' }))

    await vi.waitFor(() =>
      expect(screen.getByRole('alert').textContent).toBe('The video connection could not be started. Check your connection and try again.'),
    )
  })
})

describe('CallScreen: in the call', () => {
  it('waits for the patient, then shows them', async () => {
    const { last } = await inCall()

    expect(screen.getByText('Waiting for the patient')).toBeTruthy()
    expect(screen.getByText('Ada Example has not joined yet.')).toBeTruthy()

    act(() => last().emit(peer(true)))

    expect(screen.queryByText('Waiting for the patient')).toBeNull()
    expect(screen.queryByText(/camera is off|is muted/)).toBeNull()
  })

  it("says when the patient's camera is off and when they are muted", async () => {
    const { last } = await inCall()

    act(() => last().emit(peer(true, false, false)))

    expect(screen.getByText("Ada Example's camera is off")).toBeTruthy()
    expect(screen.getByText('Ada Example is muted')).toBeTruthy()
  })

  it('says transcription is on while the agent is in the room, and not before or after', async () => {
    const { last } = await inCall()
    act(() => last().emit(peer(true)))
    expect(screen.queryByText('Transcription is on')).toBeNull()

    act(() => last().emit({ type: 'agent', present: true }))
    expect(screen.getByRole('status', { name: 'Transcription is on' })).toBeTruthy()

    act(() => last().emit({ type: 'agent', present: false }))
    expect(screen.queryByText('Transcription is on')).toBeNull()
  })

  it('stops saying transcription is on once the connection is lost', async () => {
    const { last } = await inCall()
    act(() => last().emit({ type: 'agent', present: true }))

    act(() => last().emit({ type: 'disconnected', ending: 'lost' }))

    expect(screen.queryByText('Transcription is on')).toBeNull()
  })

  it('turns the microphone and the camera off and on', async () => {
    const { last } = await inCall()

    fireEvent.click(button('Mute microphone'))
    await screen.findByRole('button', { name: 'Unmute microphone' })
    fireEvent.click(button('Turn camera off'))
    await screen.findByRole('button', { name: 'Turn camera on' })
    fireEvent.click(button('Unmute microphone'))
    await screen.findByRole('button', { name: 'Mute microphone' })

    expect(last().microphone).toEqual([false, true])
    expect(last().camera).toEqual([false])
  })

  // The status says why the patient is gone; the join endpoint is not asked (SPEC §2 "Consultation status").
  it.each([
    ['left on their own', 'left', 'Ada Example has left the consultation. They can join again until the consultation ends.'],
    ['was removed after withdrawing consent', 'consent_withdrawn', 'The patient withdrew consent for this clinic, so the consultation has ended.'],
    ['was removed after a cancellation', 'cancelled', 'This appointment was cancelled, so the consultation has ended.'],
  ])('says why when the patient %s, from the status alone', async (_, reason, line) => {
    let state: object = open
    const { last, requests } = await inCall({ routes: { [statusGet]: () => Response.json(state) } })
    act(() => last().emit(peer(true)))
    state = { ...open, patientJoinedAt: '2026-11-09T08:02:00Z', reason }
    const before = requests.length

    act(() => last().emit(peer(false)))

    expect(await screen.findByText(line)).toBeTruthy()
    expect(paths(requests.slice(before))).toEqual([statusGet])
    expect(screen.getByRole('button', { name: 'End consultation' })).toBeTruthy()
  })

  it('reads the status once when the patient goes, however many events LiveKit sends while they are gone', async () => {
    const { last, requests } = await inCall()
    act(() => last().emit(peer(true)))
    const before = requests.length

    act(() => last().emit(peer(false)))
    act(() => last().emit(peer(false, false, false)))
    act(() => last().emit(peer(false)))

    await screen.findByText('Ada Example has left the consultation. They can join again until the consultation ends.')
    expect(paths(requests.slice(before))).toEqual([statusGet])
  })

  it('says the patient left when the status cannot be read', async () => {
    let readable = true
    const { last } = await inCall({ routes: { [statusGet]: () => (readable ? Response.json(open) : apiError(500, 'internal')) } })
    act(() => last().emit(peer(true)))
    readable = false

    act(() => last().emit(peer(false)))

    expect(await screen.findByText('Ada Example has left the consultation. They can join again until the consultation ends.')).toBeTruthy()
  })

  it('shows reconnecting, then reconnected', async () => {
    const { last } = await inCall()

    act(() => last().emit({ type: 'reconnecting' }))
    expect(screen.getByText('Connection lost. Reconnecting…')).toBeTruthy()

    act(() => last().emit({ type: 'reconnected' }))
    expect(screen.queryByText('Connection lost. Reconnecting…')).toBeNull()
    expect(screen.getByText('Reconnected')).toBeTruthy()
  })

  it('asks for a new token and joins again when the connection is lost', async () => {
    const { last, rooms, requests } = await inCall()

    act(() => last().emit({ type: 'disconnected', ending: 'lost' }))

    await vi.waitFor(() => expect(rooms).toHaveLength(2))
    expect(paths(requests).filter((p) => p === joinPost)).toHaveLength(2)
    expect(rooms[1].settings).toEqual(rooms[0].settings)
    expect(await screen.findByText('Reconnected')).toBeTruthy()
  })

  it('does not say it waits for the patient while it reconnects', async () => {
    let answer: (r: Response) => void = () => {}
    let held = false
    const { last } = await inCall({ routes: { [joinPost]: () => (held ? new Promise<Response>((r) => (answer = r)) : Response.json(joined)) } })
    act(() => last().emit(peer(true)))
    held = true

    act(() => last().emit({ type: 'disconnected', ending: 'lost' }))

    expect(await screen.findByText('Connection lost. Reconnecting…')).toBeTruthy()
    expect(screen.queryByText('Waiting for the patient')).toBeNull()
    answer(Response.json(joined))
    await screen.findByText('Reconnected')
  })

  it('offers to try again when the connection cannot be restored', async () => {
    let reachable = true
    const { last, rooms } = await inCall({ routes: { [joinPost]: () => (reachable ? Response.json(joined) : Promise.reject(new TypeError('offline'))) } })
    reachable = false

    act(() => last().emit({ type: 'disconnected', ending: 'lost' }))

    expect(await screen.findByText('The connection could not be restored.')).toBeTruthy()
    reachable = true
    fireEvent.click(button('Try again'))
    await vi.waitFor(() => expect(rooms).toHaveLength(2))
    expect(await screen.findByRole('button', { name: 'Leave' })).toBeTruthy()
  })

  it('ignores the events of a room it has left', async () => {
    const { last, rooms } = await inCall()
    const first = last()
    act(() => first.emit({ type: 'disconnected', ending: 'lost' }))
    await vi.waitFor(() => expect(rooms).toHaveLength(2))

    act(() => first.emit({ type: 'reconnecting' }))

    expect(screen.queryByText('Connection lost. Reconnecting…')).toBeNull()
  })

  it('lets the practitioner leave and join again', async () => {
    const { last, rooms } = await inCall()

    fireEvent.click(button('Leave'))

    expect(await screen.findByText('You left the consultation. You can join again until it ends.')).toBeTruthy()
    expect(last().left).toBe(true)
    fireEvent.click(button('Join again'))
    await vi.waitFor(() => expect(rooms).toHaveLength(2))
    expect(await screen.findByRole('button', { name: 'Leave' })).toBeTruthy()
  })

  it('says the consultation is open in another tab when LiveKit replaced this connection, and rejoins only when asked', async () => {
    const { last, rooms, requests } = await inCall()
    const before = requests.length

    act(() => last().emit({ type: 'disconnected', ending: 'replaced' }))

    expect(await screen.findByText('The consultation is open in another tab or window.')).toBeTruthy()
    expect(screen.queryByText('You left the consultation. You can join again until it ends.')).toBeNull()
    expect(requests.length).toBe(before)
    fireEvent.click(button('Join again'))
    await vi.waitFor(() => expect(rooms).toHaveLength(2))
  })

  it('keeps saying the consultation is open elsewhere while it joins again from this tab', async () => {
    let answer: ((r: Response) => void) | undefined
    let held = false
    const { last } = await inCall({ routes: { [joinPost]: () => (held ? new Promise<Response>((r) => (answer = r)) : Response.json(joined)) } })
    act(() => last().emit({ type: 'disconnected', ending: 'replaced' }))
    held = true

    fireEvent.click(await screen.findByRole('button', { name: 'Join again' }))

    await vi.waitFor(() => expect(answer).toBeDefined())
    expect(screen.getByText('The consultation is open in another tab or window.')).toBeTruthy()
    expect((button('Join again') as HTMLButtonElement).disabled).toBe(true)
    answer?.(Response.json(joined))
    expect(await screen.findByRole('button', { name: 'Leave' })).toBeTruthy()
  })
})

describe('CallScreen: the live panels', () => {
  it('shows none while no live source is wired', async () => {
    await inCall()
    expect(screen.queryByRole('complementary')).toBeNull()
    expect(screen.queryByRole('tab')).toBeNull()
  })

  it('follows the live source only in the call, beside the video, with the captions on it', async () => {
    const live = fakeLive()
    const media = renderCall({ live })
    await screen.findByRole('button', { name: 'Join' })
    expect(live.opened).toEqual([])

    fireEvent.click(screen.getByRole('button', { name: 'Join' }))
    const side = await screen.findByRole('complementary', { name: 'Transcript, suggestions and documents' })
    expect(live.opened).toEqual([id])
    act(() => live.emit(segment({ translation: 'Placeholder caption' })))
    expect(screen.queryByRole('group', { name: 'Captions' })).toBeNull() // the patient is not in the room
    act(() => media.last().emit(peer(true)))

    const stage = document.querySelector('.stage')!
    const captions = screen.getByRole('group', { name: 'Captions' })
    expect(stage.contains(captions)).toBe(true)
    expect(stage.contains(side)).toBe(false)
    expect(side.contains(stage)).toBe(false)
  })

  it('keeps the transcript while it reconnects, and stops following once the practitioner leaves', async () => {
    const live = fakeLive()
    const { last } = await inCall({ live })
    act(() => live.emit(segment({ original: 'Placeholder kept line' })))

    act(() => last().emit({ type: 'reconnecting' }))
    expect(within(screen.getByRole('list', { name: 'Transcript' })).getByText('Placeholder kept line')).toBeTruthy()

    fireEvent.click(button('Leave'))
    expect(live.closed()).toBe(1)
    expect(screen.queryByRole('complementary')).toBeNull()
  })
})

describe('CallScreen: the end', () => {
  const ended = (extra: object) => ({ state: 'ended', practitionerJoinedAt: '2026-11-09T08:01:00Z', endedAt: '2026-11-09T08:25:00Z', ...extra })

  it.each([
    ['ended by the practitioner with the patient', { endReason: 'practitioner', patientJoinedAt: '2026-11-09T08:02:00Z' }, 'The appointment is marked as fulfilled.'],
    ['ended without the patient', { endReason: 'practitioner' }, 'The appointment is marked as a no-show.'],
    ['at the time limit', { endReason: 'time_limit', patientJoinedAt: '2026-11-09T08:02:00Z' }, 'The consultation reached its time limit and has ended.'],
    ['by a cancellation', { endReason: 'cancelled', patientJoinedAt: '2026-11-09T08:02:00Z', reason: 'cancelled' }, 'This appointment was cancelled, so the consultation has ended.'],
    [
      'by the practitioner after removing the patient for a withdrawn consent',
      { endReason: 'practitioner', patientJoinedAt: '2026-11-09T08:02:00Z', reason: 'consent_withdrawn' },
      'The appointment is marked as fulfilled.',
    ],
    [
      'by a revocation before the start',
      { endReason: 'cancelled', patientJoinedAt: '2026-11-09T08:02:00Z', reason: 'consent_withdrawn' },
      'The patient withdrew consent for this clinic, so the consultation has ended.',
    ],
  ])('says the consultation has ended when the room is deleted, %s', async (_, status, line) => {
    let state: object = open
    const { last } = await inCall({ routes: { [statusGet]: () => Response.json(state) } })
    state = ended(status)

    act(() => last().emit({ type: 'disconnected', ending: 'ended' }))

    expect(await screen.findByRole('heading', { name: 'The consultation has ended' })).toBeTruthy()
    expect(screen.getByText(line)).toBeTruthy()
  })

  it('asks to confirm, and keeps the call on cancel', async () => {
    const { requests, last } = await inCall()

    fireEvent.click(button('End consultation'))

    expect(screen.getByText('End the consultation?')).toBeTruthy()
    expect(screen.getByText('The patient has not joined. The appointment will be marked as a no-show.')).toBeTruthy()
    fireEvent.click(button('Cancel'))
    expect(screen.queryByText('End the consultation?')).toBeNull()
    expect(paths(requests)).not.toContain(endPost)
    expect(last().left).toBe(false)
  })

  it('ends after the confirmation with the CSRF header, and says the appointment is fulfilled', async () => {
    const { requests, last } = await inCall({ routes: { [endPost]: () => Response.json(ended({ endReason: 'practitioner', patientJoinedAt: '2026-11-09T08:02:00Z' })) } })
    act(() => last().emit(peer(true)))

    fireEvent.click(button('End consultation'))
    expect(screen.getByText('The patient will be disconnected and cannot join again. The appointment will be marked as fulfilled.')).toBeTruthy()
    fireEvent.click(screen.getAllByRole('button', { name: 'End consultation' }).at(-1)!)

    expect(await screen.findByRole('heading', { name: 'The consultation has ended' })).toBeTruthy()
    expect(screen.getByText('The appointment is marked as fulfilled.')).toBeTruthy()
    expect(paths(requests)).toContain(endPost)
    expectCsrfOnWrites(requests)
    // The room is left in the cleanup of the effect on `room` (CallScreen), which React runs after the commit that
    // shows the heading: on a slow runner the heading is found first (CI, PR #33). Wait for the effect, not the tick.
    await vi.waitFor(() => expect(last().left).toBe(true))
  })

  it('counts a patient who joined before this connection when asking to confirm', async () => {
    await inCall({ routes: { [statusGet]: () => Response.json({ ...open, patientJoinedAt: '2026-11-09T08:00:30Z' }) } })

    fireEvent.click(button('End consultation'))

    expect(await screen.findByText('The patient will be disconnected and cannot join again. The appointment will be marked as fulfilled.')).toBeTruthy()
  })

  it('shows the message when the end is refused', async () => {
    await inCall({ routes: { [endPost]: () => apiError(409, 'consultation_not_started') } })

    fireEvent.click(button('End consultation'))
    fireEvent.click(screen.getAllByRole('button', { name: 'End consultation' }).at(-1)!)

    await vi.waitFor(() => expect(screen.getByRole('alert').textContent).toBe('You have not joined this consultation yet, so there is nothing to end.'))
  })
})

describe('CallScreen: a rejoin after the window closed', () => {
  // The window has closed while the room is still occupied: the status stays open (the window-close job
  // snoozes) and the API refuses a new token. A refused rejoin ends the screen; it does not ask again.
  it.each([
    ['the connection is lost', 'lost'],
    ['the room is closed while the status is still open', 'ended'],
  ] as const)('shows the end, without asking again, when %s and the rejoin is refused', async (_, ending) => {
    let refuse = false
    const { last, requests } = await inCall({ routes: { [joinPost]: () => (refuse ? apiError(409, 'consultation_ended') : Response.json(joined)) } })
    refuse = true

    act(() => last().emit({ type: 'disconnected', ending }))

    expect(await screen.findByRole('heading', { name: 'The consultation has ended' })).toBeTruthy()
    expect(paths(requests).filter((p) => p === joinPost)).toHaveLength(2)
  })
})

describe('CallScreen: unmounting', () => {
  it('leaves a room whose connection finishes after the screen was closed while joining', async () => {
    const fake = fakeFetch({ [joinPost]: () => Response.json(joined), [statusGet]: () => Response.json(open) })
    const { media, rooms } = fakeMedia()
    let release: (() => void) | undefined
    const held = { ...media, connect: (...args: Parameters<typeof media.connect>) => new Promise<Awaited<ReturnType<typeof media.connect>>>((r) => (release = () => r(media.connect(...args)))) }
    const { unmount } = renderInLanguage(<CallScreen api={newApiClient(fake.fetch)} id={id} media={held} />, 'en')
    fireEvent.click(await screen.findByRole('button', { name: 'Join' }))
    await vi.waitFor(() => expect(release).toBeDefined())

    unmount()
    release?.()

    await vi.waitFor(() => expect(rooms[0]?.left).toBe(true))
  })
})
