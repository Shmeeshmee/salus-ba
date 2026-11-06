import { createConnection, createServer, type Socket } from 'node:net'

// The join answer's LiveKit URL (E2E_LIVEKIT_PUBLIC_URL, ws://localhost:<port>) is the one for a browser on a
// developer's host, where Docker forwards the published port to LiveKit. This does the same inside the playwright
// container: a TCP forward from that localhost port to LiveKit's signalling port on the compose network
// (TEST_LIVEKIT_URL). Media does not pass through it: the browser reaches LiveKit's ICE candidates (its container
// address, UDP 7882 and TCP 7881) directly, as any client on the compose network. Only the consultation journey
// joins a call and sets E2E_LIVEKIT_PUBLIC_URL (its spec requires it); without it nothing is forwarded.
export default async function forwardLiveKit(): Promise<() => Promise<void>> {
  if (!process.env.E2E_LIVEKIT_PUBLIC_URL) return async () => {}
  const from = new URL(required('E2E_LIVEKIT_PUBLIC_URL'))
  const to = new URL(required('TEST_LIVEKIT_URL'))
  const open = new Set<Socket>()
  const server = createServer((client) => {
    const upstream = createConnection(Number(to.port), to.hostname)
    for (const s of [client, upstream]) {
      open.add(s)
      s.on('close', () => open.delete(s))
    }
    client.on('error', () => upstream.destroy())
    upstream.on('error', () => client.destroy())
    client.pipe(upstream).pipe(client)
  })
  await new Promise<void>((resolve, reject) => {
    server.once('error', reject)
    server.listen(Number(from.port), '127.0.0.1', resolve)
  })
  return () =>
    new Promise<void>((resolve) => {
      for (const s of open) s.destroy()
      server.close(() => resolve())
    })
}

function required(name: string): string {
  const value = process.env[name]
  if (!value) throw new Error(`${name} is not set; make e2e runs the specs from the Go tests in test/e2e`)
  return value
}
