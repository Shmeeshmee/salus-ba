import { useSyncExternalStore } from 'react'

// The app has a handful of screens and no router library: the screen follows location.pathname.

// A screen with unsaved changes guards leaving it: ask answers whether to leave here for to now. navigate asks
// it first, and usePath asks it before following the browser's back or forward, which it undoes on a no. The
// check sits in usePath's own listener: the browser renders between popstate listeners, so a separate one
// could run after the screen is already gone.
let guard: { here: string; ask: (to: string) => boolean } | undefined

export function guardLeaving(ask: (to: string) => boolean): () => void {
  const mine = { here: location.pathname, ask }
  guard = mine
  return () => {
    if (guard === mine) guard = undefined
  }
}

function subscribe(onChange: () => void) {
  const listener = () => {
    if (guard && location.pathname !== guard.here && !guard.ask(location.pathname)) {
      history.pushState(null, '', guard.here)
      return
    }
    onChange()
  }
  globalThis.addEventListener('popstate', listener)
  return () => globalThis.removeEventListener('popstate', listener)
}

export function usePath(): string {
  return useSyncExternalStore(subscribe, () => location.pathname)
}

export function navigate(path: string, replace = false) {
  if (guard && path !== location.pathname && !guard.ask(path)) return
  if (replace) history.replaceState(null, '', path)
  else history.pushState(null, '', path)
  globalThis.dispatchEvent(new PopStateEvent('popstate'))
}
