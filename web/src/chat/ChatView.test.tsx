import { act, cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderInLanguage } from '../i18n/renderInLanguage'
import type { Member } from '../staff'
import { ChatError, type Conversation } from './chatApi'
import { ChatView } from './ChatView'
import { fakeChat, type FakeChat } from './fakeChat'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

const role = (n: number) => `e0000000-0000-4000-8000-00000000000${n}`
const org = 'a0000000-0000-4000-8000-000000000001'
const member = (n: number): Member => ({ roleId: role(n), practitionerId: `p${n}`, name: `Synthetic Doctor ${n}`, active: true, version: '1' })
const staff = [member(1), member(2), member(3)]
const patientName = async () => 'Synthetic Patient'

function show(chat: FakeChat, id = chat.conversations[0].id) {
  return renderInLanguage(<ChatView chat={chat} id={id} staff={staff} practitionerId="p1" organizationId={org} zone="Europe/Sarajevo" patientName={patientName} />, 'en')
}

const log = () => screen.getByRole('log')

describe('ChatView', () => {
  it('shows the newest messages at the bottom with the patient named, and marks them read', async () => {
    const chat = fakeChat({ history: 5 })
    const read = vi.spyOn(chat, 'markRead')
    show(chat)

    expect(await screen.findByText('Synthetic placeholder message 5')).toBeTruthy()
    expect(await screen.findByRole('heading', { name: 'Synthetic Patient' })).toBeTruthy()
    const rows = within(log()).getAllByRole('listitem')
    expect(rows.at(-1)?.textContent).toContain('Synthetic placeholder message 5')
    await waitFor(() => expect(read).toHaveBeenLastCalledWith(chat.conversations[0].id, 5))
  })

  it('adds live messages once and marks them read', async () => {
    const chat = fakeChat({ history: 2 })
    const read = vi.spyOn(chat, 'markRead')
    show(chat)
    await screen.findByText('Synthetic placeholder message 2')

    act(() => void chat.deliver(chat.conversations[0].id, 'Synthetic placeholder live'))

    expect(await screen.findByText('Synthetic placeholder live')).toBeTruthy()
    await waitFor(() => expect(read).toHaveBeenLastCalledWith(chat.conversations[0].id, 3))
  })

  it('sends with a fresh client id and retries a failed send with the same one', async () => {
    const chat = fakeChat()
    const send = vi.spyOn(chat, 'send').mockRejectedValueOnce(new Error('offline'))
    show(chat)
    const box = await screen.findByRole('textbox', { name: 'Message' })

    fireEvent.change(box, { target: { value: 'Synthetic placeholder question' } })
    fireEvent.click(screen.getByRole('button', { name: 'Send' }))
    expect(await screen.findByText('The message was not sent.')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }))

    expect(await within(log()).findByText('Synthetic placeholder question')).toBeTruthy()
    expect(within(log()).getAllByText('Synthetic placeholder question')).toHaveLength(1)
    expect(send).toHaveBeenCalledTimes(2)
    expect(send.mock.calls[0][1]).toMatch(/^[0-9a-f-]{36}$/)
    expect(send.mock.calls[1][1]).toBe(send.mock.calls[0][1])
    expect((box as HTMLTextAreaElement).value).toBe('')

    fireEvent.change(box, { target: { value: 'Synthetic placeholder second' } })
    fireEvent.keyDown(box, { key: 'Enter' })
    await within(log()).findByText('Synthetic placeholder second')
    expect(send.mock.calls[2][1]).not.toBe(send.mock.calls[0][1])
  })

  it('renders system messages from the catalog with the doctor and the reason, and a doctor it does not know as a doctor', async () => {
    const chat = fakeChat({ history: 1 })
    const id = chat.conversations[0].id
    await chat.addParticipant(id, { practitioner_role_id: role(2), organization_id: org, reason: 'second_opinion', history: 'full' })
    await chat.addParticipant(id, { practitioner_role_id: role(9), organization_id: 'a0000000-0000-4000-8000-000000000009', reason: 'handover', history: 'full' })
    show(chat)

    expect(await screen.findByText('Synthetic Doctor 2 joined the conversation.')).toBeTruthy()
    expect(screen.getByText('Reason: Second opinion')).toBeTruthy()
    expect(screen.getByText('A doctor was invited and joins once the patient consents.')).toBeTruthy()
    expect(screen.getByText('Reason: Handover')).toBeTruthy()
  })

  it('adds a doctor of the clinic from a chosen message, and lists them with the doctors', async () => {
    const chat = fakeChat({ history: 6 })
    const add = vi.spyOn(chat, 'addParticipant')
    show(chat)
    fireEvent.click(await screen.findByRole('button', { name: 'Add a doctor' }))

    const dialog = screen.getByRole('dialog', { name: 'Add a doctor' })
    const doctor = within(dialog).getByLabelText('Doctor')
    expect(within(doctor).getAllByRole('option').map((o) => o.textContent)).toEqual(['Synthetic Doctor 2', 'Synthetic Doctor 3'])
    fireEvent.change(doctor, { target: { value: role(3) } })
    fireEvent.change(within(dialog).getByLabelText('Reason'), { target: { value: 'specialist_review' } })
    fireEvent.click(within(dialog).getByLabelText('From a message on'))
    const from = within(dialog).getByLabelText('First message they read')
    expect(within(from).getAllByRole('option')[0].textContent).toContain('Synthetic placeholder message 6')
    fireEvent.change(from, { target: { value: '4' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'Add' }))

    expect(await screen.findByText('Synthetic Doctor 3 joined the conversation.')).toBeTruthy()
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(add).toHaveBeenCalledWith(chat.conversations[0].id, { practitioner_role_id: role(3), organization_id: org, reason: 'specialist_review', history: 'from', from_seq: 4 })
    fireEvent.click(await screen.findByText('2 doctors'))
    expect(screen.getByRole('button', { name: 'Remove Synthetic Doctor 3' })).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Remove Synthetic Doctor 1' })).toBeNull()
  })

  it('removes an added doctor, and says so through the catalog when it is refused', async () => {
    const chat = fakeChat()
    const id = chat.conversations[0].id
    await chat.addParticipant(id, { practitioner_role_id: role(2), organization_id: org, reason: 'handover', history: 'full' })
    const remove = vi.spyOn(chat, 'removeParticipant').mockRejectedValueOnce(new ChatError('forbidden'))
    show(chat)
    fireEvent.click(await screen.findByText('2 doctors'))

    fireEvent.click(screen.getByRole('button', { name: 'Remove Synthetic Doctor 2' }))
    expect(await screen.findByText('You may not do this in this conversation.')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Remove Synthetic Doctor 2' }))

    expect(await screen.findByText('Synthetic Doctor 2 was removed from the conversation.')).toBeTruthy()
    expect(remove).toHaveBeenCalledTimes(2)
  })

  it('lets only the treating doctor add or remove doctors', async () => {
    const chat = fakeChat({ history: 1 })
    const id = chat.conversations[0].id
    await chat.addParticipant(id, { practitioner_role_id: role(2), organization_id: org, reason: 'second_opinion', history: 'full' })
    renderInLanguage(<ChatView chat={chat} id={id} staff={staff} practitionerId="p2" organizationId={org} zone="Europe/Sarajevo" patientName={patientName} />, 'en')

    await screen.findByText('Synthetic placeholder message 1')
    expect(screen.queryByRole('button', { name: 'Add a doctor' })).toBeNull()
    expect(screen.queryByRole('button', { name: /^Remove/ })).toBeNull()
    expect(screen.getByRole('button', { name: 'Doctor-only threads' })).toBeTruthy()
  })

  it('lists and starts doctor-only threads, marked as not visible to the patient', async () => {
    history.replaceState(null, '', '/chats/x')
    const chat = fakeChat()
    const id = chat.conversations[0].id
    const old = await chat.openThread(id)
    show(chat)

    fireEvent.click(await screen.findByRole('button', { name: 'Doctor-only threads' }))
    const panel = screen.getByRole('complementary', { name: 'Doctor-only threads' })
    expect(within(panel).getByText('Not visible to the patient.')).toBeTruthy()
    expect(await within(panel).findAllByRole('link')).toHaveLength(1)
    fireEvent.click(within(panel).getByRole('button', { name: 'Start a thread' }))

    await waitFor(() => expect(location.pathname).not.toBe('/chats/x'))
    const started = (await chat.threads(id)).find((t) => t.id !== old.id) as Conversation
    expect(location.pathname).toBe(`/chats/${started.id}`)
  })

  it('shows a thread as doctors only, with the way back to its conversation', async () => {
    const chat = fakeChat()
    const thread = await chat.openThread(chat.conversations[0].id)
    show(chat, thread.id)

    expect(await screen.findByText('Only doctors see this thread. It is not visible to the patient.')).toBeTruthy()
    expect(screen.getByText('Doctors only')).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Back to the conversation' }).getAttribute('href')).toBe(`/chats/${chat.conversations[0].id}`)
    expect(screen.queryByRole('button', { name: 'Add a doctor' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Doctor-only threads' })).toBeNull()
  })

  it('uploads from the composer, shows the file scanning, then its thumbnail, opened in a preview', async () => {
    const chat = fakeChat()
    const id = chat.conversations[0].id
    show(chat)
    const input = await screen.findByLabelText('Attach a file')

    fireEvent.change(input, { target: { files: [new File(['synthetic'], 'placeholder.png', { type: 'image/png' })] } })
    expect(await screen.findByText('placeholder.png: checking the file')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Send' }))
    expect(await within(log()).findByText('placeholder.png: checking the file')).toBeTruthy()

    const sent = (await chat.messages(id)).messages[0]
    act(() => chat.scan(sent.attachments![0].id, 'clean'))
    const open = await within(log()).findByRole('button', { name: 'Open placeholder.png' })
    const aid = sent.attachments![0].id
    expect(within(open).getByRole('img').getAttribute('src')).toBe(`${location.origin}/conversations/${id}/attachments/${aid}/thumbnail`)
    fireEvent.click(open)
    const preview = screen.getByRole('dialog', { name: 'placeholder.png' })
    expect(within(preview).getByRole('img').getAttribute('src')).toBe(`${location.origin}/conversations/${id}/attachments/${aid}/content`)
    fireEvent.keyDown(preview, { key: 'Escape' })
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('offers any other clean file as a download, and a rejected one as blocked', async () => {
    const chat = fakeChat()
    const id = chat.conversations[0].id
    const pdf = await chat.upload(id, new File(['%PDF'], 'placeholder.pdf', { type: 'application/pdf' }))
    const bad = await chat.upload(id, new File(['x'], 'placeholder.jpg', { type: 'image/jpeg' }))
    await chat.send(id, crypto.randomUUID(), '', [pdf.id, bad.id])
    chat.scan(pdf.id, 'clean')
    chat.scan(bad.id, 'rejected')
    show(chat)

    const link = await within(await screen.findByRole('log')).findByRole('link', { name: /placeholder\.pdf/ })
    expect(link.getAttribute('href')).toBe(`${location.origin}/conversations/${id}/attachments/${pdf.id}/content`)
    expect(link.hasAttribute('download')).toBe(true)
    expect(screen.getByText('placeholder.jpg: blocked by the malware check')).toBeTruthy()
  })

  it('says why an upload was refused, through the catalog', async () => {
    const chat = fakeChat()
    vi.spyOn(chat, 'upload').mockRejectedValueOnce(new ChatError('too_large'))
    show(chat)
    fireEvent.change(await screen.findByLabelText('Attach a file'), { target: { files: [new File(['x'], 'placeholder.mp4', { type: 'video/mp4' })] } })

    expect(await screen.findByText('The file is too large.')).toBeTruthy()
  })

  it('offers no summary for a short conversation', async () => {
    show(fakeChat({ history: 49 }))
    await screen.findByText('Synthetic placeholder message 49')

    expect(screen.queryByRole('button', { name: 'Summarize' })).toBeNull()
  })

  it('shows the summary of a long conversation above the history, labelled AI generated, the messages unchanged, until dismissed', async () => {
    const chat = fakeChat({ history: 60 })
    const summarize = vi.spyOn(chat, 'summarize')
    show(chat)
    fireEvent.click(await screen.findByRole('button', { name: 'Summarize' }))

    const panel = await screen.findByRole('region', { name: 'Summary of the conversation' })
    expect(within(panel).getByText('AI generated')).toBeTruthy()
    expect(within(panel).getByText('[en] Placeholder summary of 60 messages')).toBeTruthy()
    expect(within(panel).getByText('Generated 09:01, up to message 60. The messages below are unchanged.')).toBeTruthy()
    expect(summarize).toHaveBeenCalledWith(chat.conversations[0].id, 'en')
    expect(panel.compareDocumentPosition(log()) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(within(log()).getByText('Synthetic placeholder message 60')).toBeTruthy()

    fireEvent.click(within(panel).getByRole('button', { name: 'Dismiss the summary' }))
    expect(screen.queryByRole('region', { name: 'Summary of the conversation' })).toBeNull()
    expect(within(log()).getByText('Synthetic placeholder message 60')).toBeTruthy()
  })

  it('says through the catalog when the summary is refused', async () => {
    const chat = fakeChat({ history: 60 })
    vi.spyOn(chat, 'summarize').mockRejectedValueOnce(new Error('offline'))
    show(chat)
    fireEvent.click(await screen.findByRole('button', { name: 'Summarize' }))

    expect(await screen.findByText('The summary could not be generated.')).toBeTruthy()
  })

  it('closes the conversation when access ends, and says why', async () => {
    const chat = fakeChat({ history: 1 })
    show(chat)
    await screen.findByText('Synthetic placeholder message 1')

    act(() => chat.end(chat.conversations[0].id, 'consent_required'))

    expect(await screen.findByText("The patient's consent for care has ended, so this conversation is closed.")).toBeTruthy()
    expect((screen.getByRole('textbox', { name: 'Message' }) as HTMLTextAreaElement).disabled).toBe(true)
  })

  it('says when the conversation cannot be loaded', async () => {
    const chat = fakeChat()
    show(chat, 'c0000000-0000-4000-8000-0000000000ff')

    expect(await screen.findByText('This conversation is no longer available.')).toBeTruthy()
  })
})
