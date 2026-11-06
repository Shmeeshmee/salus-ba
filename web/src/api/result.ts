import type { ErrorCode } from './client'

export type Outcome<T> = { ok: true; data: T; response: Response } | { ok: false; code?: ErrorCode; linkIds?: string[]; response?: Response }

// attempt turns an openapi-fetch call into one result: the data, or the API's error code (none when the API
// could not be reached or answered something else), with the offending items' linkIds on a 422 that names them.
export async function attempt<T>(request: Promise<{ data?: T; error?: unknown; response: Response }>): Promise<Outcome<T>> {
  try {
    const { data, error, response } = await request
    if (response.ok) return { ok: true, data: data as T, response }
    return { ok: false, code: errorCode(error), linkIds: linkIdsOf(error), response }
  } catch {
    return { ok: false }
  }
}

function errorCode(error: unknown): ErrorCode | undefined {
  if (typeof error !== 'object' || error === null || !('error' in error)) return undefined
  return (error as { error: ErrorCode }).error
}

function linkIdsOf(error: unknown): string[] | undefined {
  if (typeof error !== 'object' || error === null || !('linkIds' in error)) return undefined
  const ids = (error as { linkIds: unknown }).linkIds
  return Array.isArray(ids) ? ids.filter((x): x is string => typeof x === 'string') : undefined
}
