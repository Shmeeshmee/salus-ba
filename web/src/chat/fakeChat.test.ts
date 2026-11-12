import { describe, expect, it } from 'vitest'
import { ChatError, type ChatMessageFrame } from './chatApi'
import { fakeChat } from './fakeChat'

const role2 = 'e0000000-0000-4000-8000-000000000002'
const role3 = 'e0000000-0000-4000-8000-000000000003'
const otherClinic = 'a0000000-0000-4000-8000-000000000009'
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

  it('summarizes for the practitioner only, once they read 50 messages, in the asked language', async () => {
    const short = fakeChat({ history: 49 })
    await expect(short.summarize(short.conversations[0].id, 'bs')).rejects.toEqual(new ChatError('invalid_request'))
    const chat = fakeChat({ history: 60 })
    const id = chat.conversations[0].id
    expect(await chat.summarize(id, 'hr')).toEqual({ summary: '[hr] Placeholder summary of 60 messages', generated_at: '2026-11-12T08:01:00.000Z', up_to_seq: 60, ai_generated: true })
    const patient = fakeChat({ history: 60, me: { role: 'patient', participantId: 'd0000000-0000-4000-8000-000000000001' } })
    await expect(patient.summarize(patient.conversations[0].id, 'bs')).rejects.toEqual(new ChatError('forbidden'))
  })

  it('adds a doctor of the clinic at once and one of another clinic pending consent, each with a system message', async () => {
    const chat = fakeChat({ history: 4 })
    const c = chat.conversations[0]
    const frames: ChatMessageFrame[] = []
    chat.follow(c.id, c.cursor, (f) => frames.push(f), () => {})
    const added = await chat.addParticipant(c.id, { practitioner_role_id: role2, organization_id: c.organization_id, reason: 'second_opinion', history: 'from', from_seq: 3 })
    expect(added).toMatchObject({ role: 'practitioner', practitioner_role_id: role2, organization_id: c.organization_id, status: 'active', history_from_seq: 3 })
    const invited = await chat.addParticipant(c.id, { practitioner_role_id: role3, organization_id: otherClinic, reason: 'specialist_review', history: 'full' })
    expect(invited).toMatchObject({ status: 'pending_consent', history_from_seq: 0 })
    expect(frames.map((f) => f.payload.system)).toEqual([
      { code: 'participant_added', args: { practitioner_role_id: role2, organization_id: c.organization_id, reason: 'second_opinion' } },
      { code: 'participant_invited', args: { practitioner_role_id: role3, organization_id: otherClinic, reason: 'specialist_review' } },
    ])
    expect(frames[0].payload).toMatchObject({ kind: 'system', sender_role: 'system', seq: 5 })
    expect((await chat.get(c.id)).participants?.map((p) => p.practitioner_role_id)).toContain(role2)
    await expect(chat.addParticipant(c.id, { practitioner_role_id: role2, organization_id: c.organization_id, reason: 'handover', history: 'from' })).rejects.toEqual(new ChatError('invalid_request'))
    // from_seq is the last message the added doctor does not read (SAL-28): at most the conversation's last.
    const role4 = 'e0000000-0000-4000-8000-000000000004'
    await expect(chat.addParticipant(c.id, { practitioner_role_id: role4, organization_id: c.organization_id, reason: 'handover', history: 'from', from_seq: 7 })).rejects.toEqual(new ChatError('invalid_request'))
    expect(await chat.addParticipant(c.id, { practitioner_role_id: role4, organization_id: c.organization_id, reason: 'handover', history: 'from', from_seq: 6 })).toMatchObject({ history_from_seq: 6 })
  })

  it('removes an added doctor, never the patient or the caller', async () => {
    const chat = fakeChat()
    const c = chat.conversations[0]
    const added = await chat.addParticipant(c.id, { practitioner_role_id: role2, organization_id: c.organization_id, reason: 'handover', history: 'full' })
    await chat.removeParticipant(c.id, added.id)
    const after = await chat.get(c.id)
    expect(after.participants?.map((p) => p.id)).not.toContain(added.id)
    expect((await chat.messages(c.id)).messages[0].system).toEqual({ code: 'participant_removed', args: { practitioner_role_id: role2 } })
    const [patient, me] = after.participants ?? []
    await expect(chat.removeParticipant(c.id, patient.id)).rejects.toEqual(new ChatError('forbidden'))
    await expect(chat.removeParticipant(c.id, me.id)).rejects.toEqual(new ChatError('forbidden'))
  })

  it("opens doctor-only threads of a conversation and lists them, never for the patient", async () => {
    const chat = fakeChat()
    const c = chat.conversations[0]
    const thread = await chat.openThread(c.id)
    expect(thread).toMatchObject({ staff_only: true, parent_id: c.id, patient_id: c.patient_id })
    expect(thread.participants?.every((p) => p.role === 'practitioner')).toBe(true)
    expect((await chat.threads(c.id)).map((t) => t.id)).toEqual([thread.id])
    await chat.send(thread.id, clientId(1), 'Synthetic placeholder between doctors')
    expect((await chat.messages(c.id)).messages).toEqual([])
    const patient = fakeChat({ me: { role: 'patient', participantId: 'd0000000-0000-4000-8000-000000000001' } })
    await expect(patient.threads(patient.conversations[0].id)).rejects.toEqual(new ChatError('forbidden'))
    await expect(patient.openThread(patient.conversations[0].id)).rejects.toEqual(new ChatError('forbidden'))
  })

  it('uploads attachments while scanning, sends them without a body and moves their status on the scan', async () => {
    const chat = fakeChat()
    const id = chat.conversations[0].id
    const file = new File(['synthetic bytes'], 'placeholder.png', { type: 'image/png' })
    const a = await chat.upload(id, file)
    expect(a).toMatchObject({ filename: 'placeholder.png', content_type: 'image/png', size: 15, status: 'scanning', thumbnail: false })
    const frames: ChatMessageFrame[] = []
    chat.follow(id, undefined, (f) => frames.push(f), () => {})
    const m = await chat.send(id, clientId(1), '', [a.id])
    expect(m.attachments).toEqual([a])
    chat.scan(a.id, 'clean')
    expect(frames.at(-1)).toMatchObject({ type: 'message_edited', payload: { seq: m.seq, attachments: [{ id: a.id, status: 'clean', thumbnail: true }] } })
    expect((await chat.messages(id)).messages[0].attachments?.[0].status).toBe('clean')
    await expect(chat.send(id, clientId(2), '', [])).rejects.toEqual(new ChatError('invalid_request'))
    await expect(chat.send(id, clientId(3), '', ['f0000000-0000-4000-8000-00000000dead'])).rejects.toEqual(new ChatError('invalid_request'))
  })

  it('refuses a file of another type, and an eleventh unsent upload of the caller (SAL-30)', async () => {
    const chat = fakeChat()
    const id = chat.conversations[0].id
    const png = () => new File(['synthetic'], 'placeholder.png', { type: 'image/png' })
    await expect(chat.upload(id, new File(['x'], 'placeholder.mp4', { type: 'video/mp4' }))).rejects.toEqual(new ChatError('unsupported_type'))
    const ten = await Promise.all(Array.from({ length: 10 }, png))
    const uploaded = []
    for (const f of ten) uploaded.push(await chat.upload(id, f))
    await expect(chat.upload(id, png())).rejects.toEqual(new ChatError('too_many_files'))
    await chat.send(id, clientId(9), '', [uploaded[0].id])
    expect(await chat.upload(id, png())).toMatchObject({ status: 'scanning' })
  })
})
