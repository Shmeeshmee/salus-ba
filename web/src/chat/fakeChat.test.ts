import { describe, expect, it } from 'vitest'
import { ChatError, type ChatMessageFrame } from './chatApi'
import { fakeChat } from './fakeChat'

const clientId = (n: number) => `c${String(n).padStart(7, '0')}-0000-4000-8000-000000000000`

describe('fakeChat', () => {
  it('pages a long history newest first and walks back by the cursor', async () => {
    const chat = fakeChat({ history: 2500 })
    const [c] = (await chat.list()).conversations
    const first = await chat.messages(c.id)
    expect(first.messages).toHaveLength(50)
    expect(first.messages[0].seq).toBe(2500)
    let next = first.next
    let seen = first.messages.length
    while (next) {
      const page = await chat.messages(c.id, { before: next, limit: 100 })
      seen += page.messages.length
      next = page.next
    }
    expect(seen).toBe(2500)
  })

  it('sends once per client id and counts what the caller has not read', async () => {
    const chat = fakeChat()
    const id = chat.conversations[0].id
    const a = await chat.send(id, clientId(1), '  Synthetic placeholder  ')
    const again = await chat.send(id, clientId(1), 'typed again')
    expect(again).toEqual(a)
    expect(a.body).toBe('Synthetic placeholder')
    chat.deliver(id, 'Synthetic placeholder answer')
    expect((await chat.get(id)).unread).toBe(1)
    await chat.markRead(id, 2)
    await chat.markRead(id, 1)
    expect((await chat.get(id)).unread).toBe(0)
    await expect(chat.send(id, clientId(2), ' ')).rejects.toEqual(new ChatError('invalid_request'))
  })

  it('lets only the sender edit the current version and keeps the earlier body', async () => {
    const chat = fakeChat()
    const id = chat.conversations[0].id
    const m = await chat.send(id, clientId(1), 'Synthetic placeholder')
    const theirs = chat.deliver(id, 'Synthetic placeholder answer')
    const edited = await chat.edit(id, m.id, 'Synthetic placeholder, edited', 1)
    expect(edited.version_id).toBe(2)
    await expect(chat.edit(id, m.id, 'stale', 1)).rejects.toMatchObject({ code: 'version_conflict' })
    await expect(chat.edit(id, theirs.id, 'not mine', 1)).rejects.toMatchObject({ code: 'forbidden' })
    expect(await chat.revisions(id, m.id)).toEqual([expect.objectContaining({ body: 'Synthetic placeholder', version_id: 1 })])
  })

  it('replays after the cursor, forwards live frames once and ends with the reason', async () => {
    const chat = fakeChat({ history: 3 })
    const id = chat.conversations[0].id
    const page = await chat.messages(id)
    const frames: ChatMessageFrame[] = []
    const ends: string[] = []
    const stop = chat.follow(id, page.messages[1].cursor, (f) => frames.push(f), (e) => ends.push(e))
    chat.deliver(id, 'Synthetic placeholder live')
    expect(frames.map((f) => f.payload.seq)).toEqual([3, 4])
    chat.end(id, 'consent_required')
    expect(ends).toEqual(['consent_required'])
    stop()
  })

  it("gives the clinic's chat hours, and not_found for another clinic", async () => {
    const closed = { time_zone: 'Europe/Sarajevo', hours: [{ day: 'mon' as const, start: '08:00', end: '16:00' }], response_time_hours: 8, open_now: false }
    const chat = fakeChat({ hours: closed })
    expect(await chat.chatHours(chat.conversations[0].organization_id)).toEqual(closed)
    expect((await fakeChat().chatHours(chat.conversations[0].organization_id)).open_now).toBe(true)
    await expect(chat.chatHours('00000000-0000-4000-8000-000000000000')).rejects.toEqual(new ChatError('not_found'))
  })
})
