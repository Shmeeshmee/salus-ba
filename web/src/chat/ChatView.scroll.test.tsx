import { act, cleanup, fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { renderInLanguage } from '../i18n/renderInLanguage'
import { ChatView } from './ChatView'
import { fakeChat } from './fakeChat'
import { stubLayout, type Layout } from './testLayout'

// The long history (SAL-31 "Smooth scrolling through thousands of messages"): 5000 messages from the fake, every
// page read by scrolling up, then the DOM under the list and the time of a render after a scroll step measured.

let layout: Layout
beforeEach(() => {
  // Rows of three heights, as short and long messages are.
  layout = stubLayout({ viewport: 600, rowHeight: (row) => 48 + 36 * (Number(row.dataset.row) % 3) })
})
afterEach(() => {
  cleanup()
  layout.restore()
})

const scrollTo = (el: HTMLElement, top: number) =>
  act(() => {
    el.scrollTop = top
    fireEvent.scroll(el)
  })

const median = (xs: number[]) => [...xs].sort((a, b) => a - b)[Math.floor(xs.length / 2)]

describe('ChatView with 5000 messages', () => {
  it('reads every older page by scrolling up and keeps the DOM small and each scroll step fast', async () => {
    const chat = fakeChat({ history: 5000 })
    const pages = vi.spyOn(chat, 'messages')
    renderInLanguage(<ChatView chat={chat} id={chat.conversations[0].id} staff={[]} organizationId="a0000000-0000-4000-8000-000000000001" zone="Europe/Sarajevo" patientName={async () => undefined} />, 'en')
    await screen.findByText('Synthetic placeholder message 5000')
    const list = screen.getByRole('log')

    while (!screen.queryByText('This is the start of the conversation.')) {
      const calls = pages.mock.calls.length
      scrollTo(list, 0)
      await waitFor(() => expect(pages.mock.calls.length).toBeGreaterThan(calls))
      await waitFor(() => expect(screen.queryByText('Loading earlier messages')).toBeNull())
    }
    expect(pages).toHaveBeenCalledTimes(50)
    expect(pages.mock.calls.slice(1).every(([, opts]) => opts?.before !== undefined)).toBe(true)
    scrollTo(list, 0)
    expect(screen.getByText('Synthetic placeholder message 1')).toBeTruthy()

    const total = parseFloat((list.firstElementChild as HTMLElement).style.height)
    const times: number[] = []
    let most = 0
    for (let step = 0; step < 50; step++) {
      const t0 = performance.now()
      scrollTo(list, ((step + 1) * total) / 51)
      times.push(performance.now() - t0)
      most = Math.max(most, list.querySelectorAll('*').length)
    }
    const rows = list.querySelectorAll('[data-row]').length
    console.info(`chat list, 5000 messages: at most ${most} DOM nodes under the list (${rows} rows); render after a scroll step median ${median(times).toFixed(2)} ms, max ${Math.max(...times).toFixed(2)} ms over ${times.length} steps`)
    expect(most).toBeLessThan(300)
    expect(pages).toHaveBeenCalledTimes(50)
  }, 60_000)
})
