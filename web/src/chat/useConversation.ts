import { useCallback, useEffect, useRef, useState } from 'react'
import type { ChatApi, ChatEnd, ChatMessage, Conversation } from './chatApi'

// useConversation keeps one conversation on the page: the conversation with its participants, its messages oldest
// first (the first page, the older ones read on demand, the live ones from the socket, each once by seq), and why
// the socket ended, if it did. The screen is keyed by the conversation's id, so another conversation starts afresh.

const pageSize = 100

// merge adds messages to a list sorted by seq; one with a seq already there replaces it (an edit, a scan's
// outcome).
function merge(list: ChatMessage[], incoming: ChatMessage[]): ChatMessage[] {
  if (incoming.length === 0) return list
  const sorted = [...incoming].sort((a, b) => a.seq - b.seq)
  if (list.length === 0) return sorted
  if (sorted[sorted.length - 1].seq < list[0].seq) return [...sorted, ...list]
  if (sorted.length === 1 && sorted[0].seq > list[list.length - 1].seq) return [...list, sorted[0]]
  const bySeq = new Map(list.map((m) => [m.seq, m]))
  for (const m of sorted) bySeq.set(m.seq, m)
  return [...bySeq.values()].sort((a, b) => a.seq - b.seq)
}

export type Older = { next?: string; loading: boolean; failed: boolean }

export function useConversation(chat: ChatApi, id: string) {
  const [conversation, setConversation] = useState<Conversation>()
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [older, setOlder] = useState<Older>({ loading: false, failed: false })
  const [failure, setFailure] = useState<{ error: unknown }>()
  const [ended, setEnded] = useState<ChatEnd>()
  const loading = useRef(false)

  const refresh = useCallback(() => {
    chat.get(id).then(setConversation, () => undefined) // the participants stay as they were; the next change asks again
  }, [chat, id])

  useEffect(() => {
    let current = true
    let stop: (() => void) | undefined
    Promise.all([chat.get(id), chat.messages(id, { limit: pageSize })]).then(
      ([c, page]) => {
        if (!current) return
        setConversation(c)
        setMessages([...page.messages].reverse())
        setOlder({ next: page.next, loading: false, failed: false })
        stop = chat.follow(
          id,
          page.messages[0]?.cursor ?? c.cursor,
          (frame) => {
            if (!current) return
            setMessages((list) => merge(list, [frame.payload]))
            if (frame.type !== 'message') return
            setConversation((x) => x && { ...x, last_seq: Math.max(x.last_seq, frame.payload.seq) })
            if (frame.payload.kind === 'system') refresh()
          },
          (end) => {
            if (!current) return
            setEnded(end)
            setMessages([]) // nothing of a conversation the doctor may no longer read stays in memory
            setConversation(undefined)
          },
        )
      },
      (error: unknown) => current && setFailure({ error }),
    )
    return () => {
      current = false
      stop?.()
    }
  }, [chat, id, refresh])

  const loadOlder = useCallback(async () => {
    if (loading.current || !older.next) return
    loading.current = true
    setOlder({ ...older, loading: true, failed: false })
    try {
      const page = await chat.messages(id, { before: older.next, limit: pageSize })
      setMessages((list) => merge(list, page.messages))
      setOlder({ next: page.next, loading: false, failed: false })
    } catch {
      setOlder({ ...older, loading: false, failed: true })
    } finally {
      loading.current = false
    }
  }, [chat, id, older])

  // add puts a message the page sent itself in the list at once; the socket's copy of it is the same seq.
  const add = useCallback((m: ChatMessage) => setMessages((list) => merge(list, [m])), [])

  return { conversation, messages, older, failure, ended, loadOlder, add, refresh }
}
