import { useSyncExternalStore } from 'react'

// The app has a handful of screens and no router library: the screen follows location.pathname.

function subscribe(onChange: () => void) {
  globalThis.addEventListener('popstate', onChange)
  return () => globalThis.removeEventListener('popstate', onChange)
}

export function usePath(): string {
  return useSyncExternalStore(subscribe, () => location.pathname)
}

export function navigate(path: string, replace = false) {
  if (replace) history.replaceState(null, '', path)
  else history.pushState(null, '', path)
  globalThis.dispatchEvent(new PopStateEvent('popstate'))
}
