// The placeholder for the web app's address, like [PRICE]: filled in at build time with VITE_APP_URL.
const placeholder = '[APP URL]'

// appHref is path on the web app at appUrl (VITE_APP_URL, read at build time). Without one it is on the [APP URL]
// placeholder, so an unset address shows as a placeholder rather than a relative link that breaks on another host.
export function appHref(path: string, appUrl: string | undefined): string {
  const base = appUrl?.trim().replace(/\/+$/, '')
  return `${base || placeholder}${path}`
}
