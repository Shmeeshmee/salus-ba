import type { AnswerValue } from './api/fhir'
import type { FileKind } from './builder'
import { files as t } from './messages'

// Attached is a file of a submitted answer as the appointment detail lists it. id is the binary's id,
// read from the attachment's url; a file whose url names no binary has none and cannot be opened.
export type Attached = { id?: string; kind: FileKind; contentType: string; title: string; size: number }

const kindOf = (contentType: string): FileKind => {
  if (contentType.startsWith('image/')) return 'photo'
  if (contentType.startsWith('video/')) return 'video'
  return 'document'
}

// The url ends in /binaries/<uuid>/content (attachments SPEC §2 "FHIR"); only the id is taken from it.
const binaryPath = /\/binaries\/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})\/content$/i

function idOf(url?: string): string | undefined {
  if (!url) return undefined
  try {
    return binaryPath.exec(new URL(url).pathname)?.[1]
  } catch {
    return undefined
  }
}

export function filesOf(values: AnswerValue[]): Attached[] {
  return values.flatMap(({ valueAttachment: a }) =>
    a ? [{ id: idOf(a.url), kind: kindOf(a.contentType), contentType: a.contentType, title: a.title ?? '', size: a.size }] : [],
  )
}

// sizeText is W8's size: whole kilobytes below a megabyte, else megabytes with one decimal (1 MB = 1,000,000
// bytes, as the limits of W3 count them).
export function sizeText(bytes: number): string {
  const kb = Math.max(1, Math.round(bytes / 1000))
  return kb < 1000 ? t.kb(kb) : t.mb((bytes / 1_000_000).toFixed(1))
}
