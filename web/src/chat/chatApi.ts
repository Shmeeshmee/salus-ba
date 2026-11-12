import type { components } from '../api/schema'

// chatApi.ts is the chat's contract for the screens (docs/plans/chat.md, SAL-27): the calls of the JSON API's
// chat tag and the conversation's socket, as one interface. fakeChat.ts implements it in memory, so SAL-31's
// screens are built and tested before the backend of SAL-28 to SAL-30 is merged; the real client (SAL-31) calls
// the API behind the same interface. Nothing here logs, and no message text leaves the page but to the API.

export type Conversation = components['schemas']['Conversation']
export type ConversationParticipant = components['schemas']['ConversationParticipant']
export type ChatMessage = components['schemas']['ChatMessage']
export type MessagePage = components['schemas']['MessagePage']
export type MessageRevision = components['schemas']['MessageRevision']
export type OpenConversationRequest = components['schemas']['OpenConversationRequest']
export type ChatMessageFrame = components['schemas']['ChatMessageFrame']
export type ChatHours = components['schemas']['ChatHours']
export type ErrorCode = components['schemas']['Error']['error']

// ChatEnd is why a conversation's socket ended for good: access ended (close codes 4403 and 4404).
export type ChatEnd = 'consent_required' | 'forbidden' | 'not_found'

// ChatError is a refused call, with the API's code.
export class ChatError extends Error {
  readonly code: ErrorCode
  constructor(code: ErrorCode) {
    super(code)
    this.code = code
  }
}

export interface ChatApi {
  // list is a page of the caller's conversations, newest activity first; next is the following page's cursor.
  list(before?: string): Promise<{ conversations: Conversation[]; next?: string }>
  open(req: OpenConversationRequest): Promise<Conversation>
  get(id: string): Promise<Conversation>
  // messages is a page newest first, before the cursor; next is the older page's cursor, absent at the start.
  messages(id: string, opts?: { before?: string; limit?: number }): Promise<MessagePage>
  // send is idempotent by clientMessageId: a retry with the same id gives the first message.
  send(id: string, clientMessageId: string, body: string): Promise<ChatMessage>
  edit(id: string, messageId: string, body: string, version: number): Promise<ChatMessage>
  revisions(id: string, messageId: string): Promise<MessageRevision[]>
  markRead(id: string, seq: number): Promise<void>
  // follow delivers every message after the cursor (or from now) and then the live ones, each once, until the
  // returned stop is called or access ends.
  follow(id: string, after: string | undefined, onFrame: (frame: ChatMessageFrame) => void, onEnd: (end: ChatEnd) => void): () => void
  // chatHours is the clinic's chat hours, its stated response time and whether it is within its hours now (SAL-29):
  // the screens show the out-of-hours notice from it.
  chatHours(organizationId: string): Promise<ChatHours>
}
