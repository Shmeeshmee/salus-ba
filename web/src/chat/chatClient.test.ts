import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { newApiClient } from '../api/client'
import { apiError, expectCsrfOnWrites, fakeFetch, type Route } from '../test/fakeFetch'
import { ChatError, type ChatEnd, type ChatMessageFrame } from './chatApi'
import { chatClient } from './chatClient'

const conv = 'c0000001-0000-4000-8000-000000000000'
const path = `/conversations/${conv}`
const clientID = 'cc000001-0000-4000-8000-000000000000'

// The JSON of the API (internal/communication handler tests), byte for byte.
const messageJSON = `{"body":"Synthetic placeholder","cursor":"AAAAAQ","sent_at":"2026-11-12T10:00:00Z","id":"f0000001-0000-4000-8000-000000000000","conversation_id":"${conv}","sender_participant_id":"d0000001-0000-4000-8000-000000000000","client_message_id":"${clientID}","sender_role":"patient","kind":"text","seq":1,"version_id":1}`
const message = JSON.parse(messageJSON)
const conversationJSON = `{"id":"${conv}","organization_id":"a0000001-0000-4000-8000-000000000000","patient_id":"b0000001-0000-4000-8000-000000000000","practitioner_role_id":"e0000001-0000-4000-8000-000000000000","created_at":"2026-11-12T10:00:00Z","last_seq":1,"cursor":"AAAAAQ","unread":0,"staff_only":false}`
const conversation = JSON.parse(conversationJSON)
const json = (text: string, status = 200) => new Response(text, { status, headers: { 'Content-Type': 'application/json' } })

function setup(routes: Record<string, Route>) {
  const fake = fakeFetch(routes)
  const sockets: FakeSocket[] = []
  const chat = chatClient({
    api: newApiClient(fake.fetch),
    open: (url) => {
      const s = new FakeSocket(url)
      sockets.push(s)
      return s as unknown as WebSocket
    },
  })
  return { ...fake, chat, sockets }
}

const bodyOf = async (r: Request) => r.text()

describe('chatClient: the JSON API', () => {
  it('lists conversations by page and reads one with its messages', async () => {
    const { chat, requests } = setup({
      'GET /conversations': () => json(`{"conversations":[${conversationJSON}],"next":"n1"}`),
      [`GET ${path}`]: () => json(conversationJSON),
      [`GET ${path}/messages`]: () => json(`{"messages":[${messageJSON}],"next":"AAAAAA"}`),
    })

    expect(await chat.list('p1')).toEqual({ conversations: [conversation], next: 'n1' })
    expect(await chat.get(conv)).toEqual(conversation)
    expect(await chat.messages(conv, { before: 'AAAAAQ', limit: 100 })).toEqual({ messages: [message], next: 'AAAAAA' })
    expect(new URL(requests[0].url).search).toBe('?before=p1')
    expect(new URL(requests[2].url).search).toBe('?before=AAAAAQ&limit=100')
  })

  it('sends with the client id and the attachments, edits with If-Match and moves the read position', async () => {
    const { chat, requests } = setup({
      [`POST ${path}/messages`]: () => json(messageJSON, 201),
      [`PATCH ${path}/messages/${message.id}`]: () => json(messageJSON),
      [`PUT ${path}/read`]: () => new Response(null, { status: 204 }),
    })

    expect(await chat.send(conv, clientID, 'Synthetic placeholder')).toEqual(message)
    await chat.send(conv, clientID, '', ['ab000001-0000-4000-8000-000000000000'])
    await chat.edit(conv, message.id, 'Edited placeholder', 1)
    await chat.markRead(conv, 3)

    expect(await bodyOf(requests[0])).toBe(`{"client_message_id":"${clientID}","body":"Synthetic placeholder"}`)
    expect(await bodyOf(requests[1])).toBe(`{"client_message_id":"${clientID}","body":"","attachment_ids":["ab000001-0000-4000-8000-000000000000"]}`)
    expect(requests[2].headers.get('If-Match')).toBe('W/"1"')
    expect(await bodyOf(requests[2])).toBe('{"body":"Edited placeholder"}')
    expect(await bodyOf(requests[3])).toBe('{"seq":3}')
    expectCsrfOnWrites(requests)
  })

  it("throws the API's code for a refused call, and a plain error when the API cannot be reached", async () => {
    const { chat } = setup({
      [`GET ${path}`]: () => apiError(404, 'not_found'),
      [`POST ${path}/summary`]: () => {
        throw new TypeError('network')
      },
    })

    await expect(chat.get(conv)).rejects.toEqual(new ChatError('not_found'))
    const failed = await chat.summarize(conv, 'bs').catch((e: unknown) => e)
    expect(failed).toBeInstanceOf(Error)
    expect(failed).not.toBeInstanceOf(ChatError)
  })

  it('asks for the summary in the language and reads it as the API writes it', async () => {
    const { chat, requests } = setup({
      [`POST ${path}/summary`]: () => json('{"generated_at":"2026-11-12T10:00:00Z","summary":"Placeholder summary","up_to_seq":60,"ai_generated":true}'),
    })

    expect(await chat.summarize(conv, 'hr')).toEqual({ generated_at: '2026-11-12T10:00:00Z', summary: 'Placeholder summary', up_to_seq: 60, ai_generated: true })
    expect(await bodyOf(requests[0])).toBe('{"lang":"hr"}')
    expectCsrfOnWrites(requests)
  })

  it('adds and removes a doctor and opens and lists the doctor-only threads (the contract of SAL-28)', async () => {
    const participant = { id: 'd0000003-0000-4000-8000-000000000000', role: 'practitioner', practitioner_role_id: 'e0000002-0000-4000-8000-000000000000', organization_id: conversation.organization_id, status: 'active', added_at: '2026-11-12T10:00:00Z', history_from_seq: 2 }
    const thread = { ...conversation, id: 'c0000002-0000-4000-8000-000000000000', parent_id: conv, staff_only: true }
    const { chat, requests } = setup({
      [`POST ${path}/participants`]: () => Response.json(participant, { status: 201 }),
      [`DELETE ${path}/participants/${participant.id}`]: () => new Response(null, { status: 204 }),
      [`POST ${path}/threads`]: () => Response.json(thread, { status: 201 }),
      [`GET ${path}/threads`]: () => Response.json({ conversations: [thread] }),
    })

    const req = { practitioner_role_id: participant.practitioner_role_id, organization_id: participant.organization_id, reason: 'second_opinion' as const, history: 'from' as const, from_seq: 3 }
    expect(await chat.addParticipant(conv, req)).toEqual(participant)
    await chat.removeParticipant(conv, participant.id)
    expect(await chat.openThread(conv)).toEqual(thread)
    expect(await chat.threads(conv)).toEqual([thread])

    expect(JSON.parse(await bodyOf(requests[0]))).toEqual(req)
    expect(await bodyOf(requests[2])).toBe('{}')
    expectCsrfOnWrites(requests)
  })

  it("streams a file's bytes to the attachments with its name in Content-Disposition (the contract of SAL-30)", async () => {
    const attachment = { id: 'ab000001-0000-4000-8000-000000000000', content_type: 'image/png', size: 15, filename: 'placeholder scan.png', status: 'scanning' }
    const { chat, requests } = setup({ [`POST ${path}/attachments`]: () => Response.json(attachment, { status: 201 }) })

    expect(await chat.upload(conv, new File(['synthetic bytes'], 'placeholder scan.png', { type: 'image/png' }))).toEqual(attachment)
    const r = requests[0]
    expect(r.headers.get('Content-Type')).toBe('application/octet-stream')
    expect(r.headers.get('Content-Disposition')).toBe("attachment; filename*=UTF-8''placeholder%20scan.png")
    expect(await r.text()).toBe('synthetic bytes')
    expect(new URL(r.url).search).toBe('')
    expectCsrfOnWrites(requests)
  })
})

// FakeSocket stands in for the browser's WebSocket (as in live.test.ts).
class FakeSocket {
  closed = false
  onopen: (() => void) | null = null
  onmessage: ((e: { data: unknown }) => void) | null = null
  onclose: ((e: { code: number; reason: string }) => void) | null = null
  readonly url: string
  constructor(url: string) {
    this.url = url
  }
  send() {}
  close() {
    this.closed = true
  }
  frame(...frames: unknown[]) {
    for (const f of frames) this.onmessage?.({ data: typeof f === 'string' ? f : JSON.stringify(f) })
  }
  end(code: number, reason = '') {
    this.onclose?.({ code, reason })
  }
}

const frame = (seq: number, type = 'message') => ({ type, version: 1, payload: { ...message, seq, cursor: `cur-${seq}` } })

describe('chatClient: follow', () => {
  beforeEach(() => vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] }))
  afterEach(() => vi.useRealTimers())

  function follow(after?: string) {
    const t = setup({})
    const frames: ChatMessageFrame[] = []
    const ends: ChatEnd[] = []
    const stop = t.chat.follow(conv, after, (f) => frames.push(f), (e) => ends.push(e))
    return { ...t, frames, ends, stop }
  }

  it("follows the conversation's events on the page's own origin after the cursor", () => {
    const { sockets } = follow('AAAAAQ')
    const url = new URL(sockets[0].url)
    expect(url.protocol).toBe('ws:')
    expect(url.host).toBe(location.host)
    expect(url.pathname).toBe(`${path}/events`)
    expect(url.searchParams.get('after')).toBe('AAAAAQ')
  })

  it('delivers messages and edits once each, and drops what it does not know', () => {
    const { sockets, frames } = follow()
    sockets[0].frame(frame(2), frame(3), frame(2), frame(2, 'message_edited'), { type: 'send_ack', version: 1, payload: {} }, { type: 'typing', version: 1, payload: { participant_id: 'x' } }, 'not json', { type: 'message', version: 2, payload: {} })
    expect(frames.map((f) => `${f.type} ${f.payload.seq}`)).toEqual(['message 2', 'message 3', 'message_edited 2'])
  })

  it('reconnects after a dropped connection from the last message it delivered, waiting longer each time', async () => {
    const { sockets, frames } = follow('AAAAAQ')
    sockets[0].frame(frame(2))
    sockets[0].end(1006)
    await vi.advanceTimersByTimeAsync(999)
    expect(sockets).toHaveLength(1)
    await vi.advanceTimersByTimeAsync(1)
    expect(new URL(sockets[1].url).searchParams.get('after')).toBe('cur-2')
    sockets[1].end(1011)
    await vi.advanceTimersByTimeAsync(1999)
    expect(sockets).toHaveLength(2)
    await vi.advanceTimersByTimeAsync(1)
    expect(sockets).toHaveLength(3)
    sockets[2].frame(frame(2), frame(3))
    expect(frames.map((f) => f.payload.seq)).toEqual([2, 3])
  })

  it.each([
    [4403, 'consent_required', 'consent_required'],
    [4403, 'forbidden', 'forbidden'],
    [4404, 'not_found', 'not_found'],
    [4401, 'unauthenticated', 'unauthenticated'],
  ])('stops for good on close %i %s', async (code, reason, end) => {
    const { sockets, ends } = follow()
    sockets[0].end(code, reason)
    await vi.advanceTimersByTimeAsync(60_000)
    expect(sockets).toHaveLength(1)
    expect(ends).toEqual([end])
  })

  it('closes the socket and does not come back once stopped', async () => {
    const { sockets, stop } = follow()
    stop()
    expect(sockets[0].closed).toBe(true)
    await vi.advanceTimersByTimeAsync(60_000)
    expect(sockets).toHaveLength(1)
  })
})
