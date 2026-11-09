import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { newApiClient } from './api/client'
import { newFhirClient } from './api/fhir'
import { SignRefused, type ConsultMessage, type DraftEvent } from './consult'
import { consultSocket, type End } from './live'
import { apiError, fakeFetch, fhirJSON, type Route } from './test/fakeFetch'
import { draft, segment, suggestion } from './test/fakeConsult'

const appointment = 'e1000000-0000-4000-8000-000000000001'
const encounter = 'e2000000-0000-4000-8000-000000000001'
const search = 'GET /fhir/R4/Encounter'
const found = { resourceType: 'Bundle', type: 'searchset', entry: [{ fullUrl: `/fhir/R4/Encounter/${encounter}`, resource: { resourceType: 'Encounter', id: encounter } }] }
const none = { resourceType: 'Bundle', type: 'searchset' }

// FakeSocket stands in for the browser's WebSocket: the test opens it, sends it frames and closes it.
class FakeSocket {
  sent: string[] = []
  closed = false
  onopen: (() => void) | null = null
  onmessage: ((e: { data: unknown }) => void) | null = null
  onclose: ((e: { code: number; reason: string }) => void) | null = null
  readonly url: string
  constructor(url: string) {
    this.url = url
  }
  send(data: string) {
    this.sent.push(data)
  }
  close() {
    this.closed = true
  }
  open() {
    this.onopen?.()
  }
  frame(...frames: unknown[]) {
    for (const f of frames) this.onmessage?.({ data: typeof f === 'string' ? f : JSON.stringify(f) })
  }
  end(code: number, reason = '') {
    this.onclose?.({ code, reason })
  }
}

const cursor = (id: string) => ({ type: 'cursor', version: 1, payload: { id } })

const draftsGet = `GET /encounters/${encounter}/drafts`

function setup(searches: (() => Response)[] = [() => fhirJSON(found)], routes: Record<string, Route> = {}) {
  const sockets: FakeSocket[] = []
  let n = 0
  const fake = fakeFetch({ [search]: () => searches[Math.min(n++, searches.length - 1)](), [draftsGet]: () => Response.json({ drafts: [] }), ...routes })
  const { live, actions } = consultSocket({
    api: newApiClient(fake.fetch),
    fhir: newFhirClient(fake.fetch),
    open: (url) => {
      const s = new FakeSocket(url)
      sockets.push(s)
      return s as unknown as WebSocket
    },
  })
  const got: ConsultMessage[] = []
  const ended: End[] = []
  const subscribe = () => live.subscribe(appointment, (m) => got.push(m), (e) => ended.push(e))
  return { ...fake, sockets, actions, got, ended, subscribe, last: () => sockets[sockets.length - 1] }
}

// socketOpened waits until the subscription has found the encounter and opened its n-th socket.
const socketOpened = (sockets: FakeSocket[], n = 1) => vi.waitFor(() => expect(sockets).toHaveLength(n))

beforeEach(() => vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] }))
afterEach(() => vi.useRealTimers())

describe('consultSocket: the live source', () => {
  it("finds the appointment's encounter and follows its events on the page's own origin", async () => {
    const { requests, sockets, subscribe } = setup()
    subscribe()
    await socketOpened(sockets)

    expect(new URL(requests[0].url).searchParams.get('appointment')).toBe(appointment)
    expect(new URL(requests[0].url).pathname).toBe('/fhir/R4/Encounter')
    const url = new URL(sockets[0].url)
    expect(url.protocol).toBe('ws:')
    expect(url.host).toBe(location.host)
    expect(url.pathname).toBe(`/encounters/${encounter}/events`)
    expect(url.searchParams.has('after')).toBe(false)
  })

  it('delivers each message after its cursor and drops the cursor frames and what the page does not know', async () => {
    const { sockets, subscribe, got } = setup()
    subscribe()
    await socketOpened(sockets)
    const line = segment()
    const s = suggestion()

    sockets[0].open()
    sockets[0].frame(cursor('1-0'), line, cursor('2-0'), '{"type":"dubbing","version":1,"payload":{}}', cursor('3-0'), s)

    expect(got).toEqual([line, s])
  })

  it('reconnects after a dropped connection from the last cursor whose message arrived, so nothing is lost or repeated', async () => {
    const { sockets, subscribe } = setup()
    subscribe()
    await socketOpened(sockets)
    sockets[0].open()
    sockets[0].frame(cursor('1-0'), segment(), cursor('2-0'), segment(), cursor('3-0')) // the third message never came

    sockets[0].end(1006)
    expect(sockets).toHaveLength(1)
    await vi.advanceTimersByTimeAsync(1000)

    expect(sockets).toHaveLength(2)
    expect(new URL(sockets[1].url).searchParams.get('after')).toBe('2-0')
  })

  it('reconnects after a server error, waiting longer after each failed attempt and again from the start once a frame arrives', async () => {
    const { sockets, subscribe } = setup()
    subscribe()
    await socketOpened(sockets)

    sockets[0].end(1011)
    await vi.advanceTimersByTimeAsync(1000)
    expect(sockets).toHaveLength(2)
    sockets[1].end(1006) // refused before it opened
    await vi.advanceTimersByTimeAsync(1999)
    expect(sockets).toHaveLength(2)
    await vi.advanceTimersByTimeAsync(1)
    expect(sockets).toHaveLength(3)

    sockets[2].open()
    sockets[2].frame(cursor('1-0'), segment())
    sockets[2].end(1006)
    await vi.advanceTimersByTimeAsync(1000)
    expect(sockets).toHaveLength(4)
    expect(new URL(sockets[3].url).searchParams.get('after')).toBe('1-0')
  })

  it.each<[number, string, End]>([
    [4403, 'consent_required', 'consent_required'],
    [4403, 'forbidden', 'forbidden'],
    [4404, 'not_found', 'not_found'],
  ])('ends on %i %s and does not reconnect', async (code, reason, end) => {
    const { sockets, subscribe, ended } = setup()
    subscribe()
    await socketOpened(sockets)
    sockets[0].open()

    sockets[0].end(code, reason)
    await vi.advanceTimersByTimeAsync(60_000)

    expect(ended).toEqual([end])
    expect(sockets).toHaveLength(1)
  })

  it('waits for the encounter, which starts once the practitioner is in the room, and retries a failed search', async () => {
    const { sockets, subscribe, requests } = setup([() => fhirJSON(none), () => new Response('', { status: 503 }), () => fhirJSON(found)])
    subscribe()
    await vi.waitFor(() => expect(requests).toHaveLength(1))
    expect(sockets).toHaveLength(0)

    await vi.advanceTimersByTimeAsync(1000)
    expect(requests).toHaveLength(2)
    await vi.advanceTimersByTimeAsync(2000)
    await socketOpened(sockets)
    expect(requests.filter((r) => new URL(r.url).pathname === '/fhir/R4/Encounter')).toHaveLength(3)
  })

  it('closes the socket when unsubscribed, and neither delivers nor reconnects after', async () => {
    const { sockets, subscribe, got } = setup()
    const stop = subscribe()
    await socketOpened(sockets)
    sockets[0].open()

    stop()
    expect(sockets[0].closed).toBe(true)
    sockets[0].frame(cursor('1-0'), segment())
    sockets[0].end(1006)
    await vi.advanceTimersByTimeAsync(60_000)

    expect(got).toEqual([])
    expect(sockets).toHaveLength(1)
  })

  it('opens nothing when unsubscribed while it looks for the encounter', async () => {
    const { sockets, subscribe, requests } = setup([() => fhirJSON(none)])
    const stop = subscribe()
    await vi.waitFor(() => expect(requests).toHaveLength(1))

    stop()
    await vi.advanceTimersByTimeAsync(60_000)

    expect(requests).toHaveLength(1)
    expect(sockets).toHaveLength(0)
  })
})

describe('consultSocket: a suggestion status', () => {
  const id = 'b1000000-0000-4000-8000-000000000001'

  async function connected() {
    const s = setup()
    s.subscribe()
    await socketOpened(s.sockets)
    s.sockets[0].open()
    return s
  }

  it('sends the command on the socket and is done once the suggestion comes back with that status', async () => {
    const { sockets, actions } = await connected()
    let done = false
    const pinning = actions.setSuggestionStatus(id, 'pinned').then(() => (done = true))

    expect(sockets[0].sent.map((f) => JSON.parse(f))).toEqual([{ type: 'suggestion_status', version: 1, payload: { id, status: 'pinned' } }])
    sockets[0].frame(cursor('1-0'), suggestion({ id, status: 'shown' }))
    await Promise.resolve()
    expect(done).toBe(false)
    sockets[0].frame(cursor('2-0'), suggestion({ id, status: 'pinned' }))
    await pinning
    expect(done).toBe(true)
  })

  it('fails on the command_error for that suggestion', async () => {
    const { sockets, actions } = await connected()
    const dismissing = actions.setSuggestionStatus(id, 'dismissed')

    sockets[0].frame({ type: 'command_error', version: 1, payload: { command: 'suggestion_status', id: 'b1000000-0000-4000-8000-000000000002', error: 'not_found' } })
    sockets[0].frame({ type: 'command_error', version: 1, payload: { command: 'suggestion_status', id, error: 'consent_required' } })

    await expect(dismissing).rejects.toThrow()
  })

  it('fails every command waiting on a command_error that names no suggestion, as the server does not take them', async () => {
    const { sockets, actions } = await connected()
    const pinning = actions.setSuggestionStatus(id, 'pinned')
    const dismissing = actions.setSuggestionStatus('b1000000-0000-4000-8000-000000000002', 'dismissed')

    sockets[0].frame({ type: 'command_error', version: 1, payload: { command: 'suggestion_status', error: 'unknown_command' } })

    await expect(pinning).rejects.toThrow()
    await expect(dismissing).rejects.toThrow()
  })

  it('does not take a command_error frame for a message after a cursor', async () => {
    const { sockets, actions } = await connected()
    void actions.setSuggestionStatus(id, 'dismissed').catch(() => undefined)
    sockets[0].frame(cursor('1-0'), segment(), cursor('2-0'))
    sockets[0].frame({ type: 'command_error', version: 1, payload: { command: 'suggestion_status', id, error: 'internal' } })

    sockets[0].end(1006)
    await vi.advanceTimersByTimeAsync(1000)
    expect(new URL(sockets[1].url).searchParams.get('after')).toBe('1-0')
  })

  it('fails at once while no socket is open, and when the socket closes before the answer', async () => {
    const s = setup()
    await expect(s.actions.setSuggestionStatus(id, 'pinned')).rejects.toThrow()
    s.subscribe()
    await socketOpened(s.sockets)
    await expect(s.actions.setSuggestionStatus(id, 'pinned')).rejects.toThrow() // not open yet
    expect(s.sockets[0].sent).toEqual([])

    s.sockets[0].open()
    const pinning = s.actions.setSuggestionStatus(id, 'pinned')
    s.sockets[0].end(1006)
    await expect(pinning).rejects.toThrow()
  })
})

describe('consultSocket: the stored drafts', () => {
  it('delivers the drafts the encounter already has, once its encounter is found', async () => {
    const stored = draft({ version_id: 4 }).payload as DraftEvent
    const { subscribe, got, sockets } = setup(undefined, { [draftsGet]: () => Response.json({ drafts: [stored] }) })
    subscribe()
    await socketOpened(sockets)

    await vi.waitFor(() => expect(got).toEqual([{ type: 'draft', version: 1, payload: stored }]))
  })

  it('goes on following the stream when they cannot be read', async () => {
    const { subscribe, got, sockets } = setup(undefined, { [draftsGet]: () => apiError(500, 'internal') })
    subscribe()
    await socketOpened(sockets)
    sockets[0].open()
    sockets[0].frame(cursor('1-0'), segment())
    await vi.waitFor(() => expect(got).toHaveLength(1))
  })
})

describe('consultSocket: documents', () => {
  const template = (p: Record<string, unknown>) => ({
    id: 'd1000000-0000-4000-8000-000000000001',
    title: 'Placeholder izvještaj',
    language: 'bs',
    status: 'draft',
    typeCode: '11488-4',
    fields: [
      { key: 'reason', type: 'text', label: 'Placeholder razlog', required: true },
      { key: 'plan', type: 'list', label: 'Placeholder plan', required: false },
    ],
    translations: [{ language: 'en', title: 'Placeholder report', fields: { reason: { label: 'Placeholder reason' }, plan: { label: 'Placeholder plan EN' } } }],
    lastUpdated: '2026-11-09T09:00:00Z',
    versionId: 1,
    ...p,
  })

  it("lists the clinic's templates in the language asked for where they have it, and in their own otherwise", async () => {
    const { actions } = setup(undefined, {
      'GET /document-templates': () => Response.json({ templates: [template({}), template({ id: 'd1000000-0000-4000-8000-000000000002', language: 'hr', title: 'Placeholder nalaz', translations: [] })] }),
    })

    expect(await actions.templates('en')).toEqual([
      { id: 'd1000000-0000-4000-8000-000000000001', title: 'Placeholder report', language: 'en', fields: [{ key: 'reason', label: 'Placeholder reason', required: true }, { key: 'plan', label: 'Placeholder plan EN', required: false }] },
      { id: 'd1000000-0000-4000-8000-000000000002', title: 'Placeholder nalaz', language: 'hr', fields: [{ key: 'reason', label: 'Placeholder razlog', required: true }, { key: 'plan', label: 'Placeholder plan', required: false }] },
    ])
    expect((await actions.templates('bs'))[0]).toMatchObject({ title: 'Placeholder izvještaj', language: 'bs' })
  })

  it('fails when the templates cannot be listed', async () => {
    const { actions } = setup(undefined, { 'GET /document-templates': () => apiError(500, 'internal') })
    await expect(actions.templates('en')).rejects.toThrow()
  })

  it("drafts on the appointment's encounter in the language given and answers with the stored draft", async () => {
    const stored = draft({ version_id: 2 }).payload as DraftEvent
    const { actions, requests } = setup(undefined, { [`POST /encounters/${encounter}/drafts`]: () => Response.json(stored) })

    expect(await actions.generate(appointment, 'd1000000-0000-4000-8000-000000000001', 'en')).toEqual(stored)
    const post = requests.find((r) => r.method === 'POST')!
    expect(post.headers.get('X-Salus-CSRF')).toBe('1')
    expect(await post.json()).toEqual({ templateId: 'd1000000-0000-4000-8000-000000000001', language: 'en' })
  })

  it.each([
    ['the drafting model is unavailable', () => fhirJSON(found), () => apiError(503, 'drafting_unavailable')],
    ['the appointment has no encounter', () => fhirJSON(none), () => Response.json({})],
  ])('fails to draft when %s', async (_, searched, posted) => {
    const { actions } = setup([searched], { [`POST /encounters/${encounter}/drafts`]: posted })
    await expect(actions.generate(appointment, 'd1000000-0000-4000-8000-000000000001', 'en')).rejects.toThrow()
  })

  const signPost = 'POST /drafts/c1000000-0000-4000-8000-000000000001/sign'
  const fields = [
    { key: 'reason', value: 'Placeholder typed reason' },
    { key: 'plan', value: '' },
  ]

  it('signs the draft as it was read, with If-Match its version, and answers with the signed draft', async () => {
    const read = draft({ version_id: 3 }).payload as DraftEvent
    const signed = { ...read, status: 'final', version_id: 4 }
    const { actions, requests } = setup(undefined, { [signPost]: () => Response.json(signed) })

    expect(await actions.sign(read, fields)).toEqual(signed)
    const post = requests[0]
    expect(post.headers.get('If-Match')).toBe('W/"3"')
    expect(post.headers.get('X-Salus-CSRF')).toBe('1')
    expect(await post.json()).toEqual({ fields })
  })

  it('signs nothing for a draft read without its version', async () => {
    const { actions, requests } = setup(undefined, { [signPost]: () => Response.json({}) })
    const read = { ...(draft().payload as DraftEvent), version_id: undefined }

    await expect(actions.sign(read, fields)).rejects.toThrow()
    expect(requests).toEqual([])
  })

  it('fails with the required fields it names when a required field is empty', async () => {
    const { actions } = setup(undefined, { [signPost]: () => Response.json({ error: 'required_missing', keys: ['reason'] }, { status: 422 }) })

    const refused = await actions.sign(draft().payload as DraftEvent, fields).catch((e: unknown) => e)
    expect(refused).toBeInstanceOf(SignRefused)
    expect((refused as SignRefused).keys).toEqual(['reason'])
  })

  it.each([
    ['a redraft since it was read', 412, 'version_mismatch'],
    ['a value of the wrong type', 422, 'invalid_values'],
    ['it is signed already', 409, 'not_preliminary'],
  ])('fails, not as a missing field, when %s', async (_, status, code) => {
    const { actions } = setup(undefined, { [signPost]: () => Response.json({ error: code, keys: [] }, { status }) })
    const refused = await actions.sign(draft().payload as DraftEvent, fields).catch((e: unknown) => e)
    expect(refused).toBeInstanceOf(Error)
    expect(refused).not.toBeInstanceOf(SignRefused)
  })
})
