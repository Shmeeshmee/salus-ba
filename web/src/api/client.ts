import createClient from 'openapi-fetch'
import type { paths } from './schema'

export type ApiClient = ReturnType<typeof createClient<paths>>

// newApiClient calls the API on the page's own origin (the Vite dev server proxies it). Tests pass a fetch.
export function newApiClient(fetchImpl?: (input: Request) => Promise<Response>): ApiClient {
  return createClient<paths>({ baseUrl: globalThis.location.origin, fetch: fetchImpl })
}
