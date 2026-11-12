import { cleanup, fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderInLanguage } from '../i18n/renderInLanguage'
import { Conversations } from './Conversations'
import { fakeChat, type FakeChat } from './fakeChat'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

function show(chat: FakeChat, patientName = async (): Promise<string | undefined> => 'Synthetic Patient') {
  return renderInLanguage(<Conversations chat={chat} zone="Europe/Sarajevo" patientName={patientName} />, 'en')
}

describe('Conversations', () => {
  it('lists the conversations with the patient, the last activity and the unread count, each opening its chat', async () => {
    const chat = fakeChat({ history: 1 })
    const id = chat.conversations[0].id
    chat.deliver(id, 'Synthetic placeholder unread')
    chat.deliver(id, 'Synthetic placeholder unread again')
    show(chat)

    expect(await screen.findByRole('heading', { name: 'Messages' })).toBeTruthy()
    const link = await screen.findByRole('link', { name: /Synthetic Patient/ })
    expect(link.getAttribute('href')).toBe(`/chats/${id}`)
    expect(within(link).getByText('2 unread messages')).toBeTruthy()
    expect(within(link).getByText(/09:00/)).toBeTruthy()
    fireEvent.click(link)
    expect(location.pathname).toBe(`/chats/${id}`)
  })

  it('marks a doctor-only thread, shows no count when all is read, and falls back to "Patient" without a name', async () => {
    const chat = fakeChat({ history: 1 })
    await chat.markRead(chat.conversations[0].id, 1)
    await chat.openThread(chat.conversations[0].id)
    show(chat, async () => undefined)

    const links = await screen.findAllByRole('link')
    expect(links).toHaveLength(2)
    expect(screen.getByText('Doctors only')).toBeTruthy()
    expect(screen.getAllByText('Patient')).toHaveLength(2)
    expect(screen.queryByText(/unread/)).toBeNull()
  })

  it('pages with Show more', async () => {
    const chat = fakeChat()
    const list = vi.spyOn(chat, 'list')
    list.mockResolvedValueOnce({ conversations: [{ ...chat.conversations[0], participants: undefined }], next: 'n1' })
    list.mockResolvedValueOnce({ conversations: [{ ...chat.conversations[0], id: 'c0000000-0000-4000-8000-0000000000aa', participants: undefined }] })
    show(chat)

    fireEvent.click(await screen.findByRole('button', { name: 'Show more' }))

    await waitFor(() => expect(screen.getAllByRole('link')).toHaveLength(2))
    expect(list).toHaveBeenLastCalledWith('n1')
    expect(screen.queryByRole('button', { name: 'Show more' })).toBeNull()
  })

  it('says when there are none, and when they cannot be loaded', async () => {
    const chat = fakeChat()
    vi.spyOn(chat, 'list').mockResolvedValueOnce({ conversations: [] })
    show(chat)
    expect(await screen.findByText('No conversations yet.')).toBeTruthy()
    cleanup()

    vi.spyOn(chat, 'list').mockRejectedValueOnce(new Error('offline'))
    show(chat)
    expect(await screen.findByText('The conversations could not be loaded.')).toBeTruthy()
  })
})
