import createClient from 'openapi-fetch'
import type { components, paths } from './schema'

export type ApiClient = ReturnType<typeof createClient<paths>>
export type Account = components['schemas']['Account']
export type ErrorCode = components['schemas']['Error']['error']
export type Fetch = (input: Request) => Promise<Response>

// binaryContentURL is where a file's bytes are read (GET /binaries/{id}/content), on the same origin as every
// other call, so the session cookie goes with it. It is built from the binary id, never taken from a resource.
export function binaryContentURL(id: string): string {
  return `${globalThis.location.origin}/binaries/${encodeURIComponent(id)}/content`
}

// newApiClient calls the API on the page's own origin (the Vite dev server proxies it). The session travels
// in the salus_session cookie; every write carries X-Salus-CSRF: 1, which the API requires with the cookie
// and which also makes the login steps answer with the cookie. Tests pass a fetch.
export function newApiClient(fetchImpl?: Fetch): ApiClient {
  const client = createClient<paths>({ baseUrl: globalThis.location.origin, fetch: fetchImpl })
  client.use({
    onRequest({ request }) {
      if (request.method !== 'GET' && request.method !== 'HEAD') request.headers.set('X-Salus-CSRF', '1')
      return request
    },
  })
  return client
}
