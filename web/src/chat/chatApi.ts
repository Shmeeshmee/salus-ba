import type { components } from '../api/schema'

// chatApi.ts is the chat's contract for the screens (docs/plans/chat.md, SAL-27): the calls of the JSON API's
// chat tag and the conversation's socket, as one interface. fakeChat.ts implements it in memory, so SAL-31's
// screens are built and tested before the backend of SAL-28 to SAL-30 is merged; chatClient.ts calls the API
// behind the same interface. Nothing here logs, and no message text leaves the page but to the API.

type Schemas = components['schemas']

export type ChatAttachment = Schemas['ChatAttachment']
export type ConversationParticipant = Schemas['ConversationParticipant']
export type Conversation = Schemas['Conversation']
export type ChatMessage = Schemas['ChatMessage']
export type MessagePage = Schemas['MessagePage']
export type ChatMessageFrame = Schemas['ChatMessageFrame']
export type MessageRevision = Schemas['MessageRevision']
export type OpenConversationRequest = Schemas['OpenConversationRequest']
export type ChatHours = Schemas['ChatHours']
export type ErrorCode = Schemas['Error']['error']
export type Summary = Schemas['SummaryResponse']
export type SummaryLanguage = NonNullable<Schemas['SummaryRequest']['lang']>
// AddParticipantRequest adds a practitioner role (SAL-28): with history `from`, from_seq is the last message the
// added doctor does not read; they read the messages after it.
export type AddParticipantRequest = Schemas['AddParticipantRequest']
export type Reason = AddParticipantRequest['reason']

export const reasons = ['second_opinion', 'specialist_review', 'handover'] as const satisfies readonly Reason[]

// fileTypes are the types an upload may have (SAL-30: detected from the bytes on the server).
export const fileTypes = ['image/jpeg', 'image/png', 'application/pdf'] as const satisfies readonly ChatAttachment['content_type'][]

// ChatEnd is why a conversation's socket ended for good: access ended (close codes 4403 and 4404), or the
// session did (4401).
export type ChatEnd = 'consent_required' | 'forbidden' | 'not_found' | 'unauthenticated'

// ChatError is a refused call, with the API's code.
export class ChatError extends Error {
  readonly code: ErrorCode
  constructor(code: ErrorCode) {
    super(code)
    this.code = code
  }
}

// attachmentURL is where an attachment's bytes (or its thumbnail) are read once clean, on the page's own origin so
// the session cookie goes with it; built from the ids, never taken from a message.
export function attachmentURL(conversationId: string, attachmentId: string, part: 'content' | 'thumbnail'): string {
  return `${globalThis.location.origin}/conversations/${encodeURIComponent(conversationId)}/attachments/${encodeURIComponent(attachmentId)}/${part}`
}

export interface ChatApi {
  // list is a page of the caller's conversations, newest activity first; next is the following page's cursor.
  list(before?: string): Promise<{ conversations: Conversation[]; next?: string }>
  open(req: OpenConversationRequest): Promise<Conversation>
  get(id: string): Promise<Conversation>
  // messages is a page newest first, before the cursor; next is the older page's cursor, absent at the start.
  messages(id: string, opts?: { before?: string; limit?: number }): Promise<MessagePage>
  // send is idempotent by clientMessageId: a retry with the same id gives the first message. With attachments the
  // body may be empty.
  send(id: string, clientMessageId: string, body: string, attachmentIds?: string[]): Promise<ChatMessage>
  edit(id: string, messageId: string, body: string, version: number): Promise<ChatMessage>
  revisions(id: string, messageId: string): Promise<MessageRevision[]>
  markRead(id: string, seq: number): Promise<void>
  // follow delivers every message after the cursor (or from now) and then the live ones, each once, until the
  // returned stop is called or access ends.
  follow(id: string, after: string | undefined, onFrame: (frame: ChatMessageFrame) => void, onEnd: (end: ChatEnd) => void): () => void
  // summarize is the AI summary of a long conversation (SAL-31), for its practitioner; not stored.
  summarize(id: string, lang: SummaryLanguage): Promise<Summary>
  addParticipant(id: string, req: AddParticipantRequest): Promise<ConversationParticipant>
  removeParticipant(id: string, participantId: string): Promise<void>
  // threads and openThread are the doctor-only side threads of a conversation (SAL-28), never the patient's.
  threads(id: string): Promise<Conversation[]>
  openThread(id: string): Promise<Conversation>
  // upload sends a file through the scan pipeline (SAL-30); it comes back scanning.
  upload(id: string, file: File): Promise<ChatAttachment>
  // chatHours is the clinic's chat hours, its stated response time and whether it is within its hours now (SAL-29):
  // the screens show the out-of-hours notice from it.
  chatHours(organizationId: string): Promise<ChatHours>
}
