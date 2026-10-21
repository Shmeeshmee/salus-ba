// jsdom has no matchMedia. Installs one whose prefers-reduced-motion answer a test controls; returns the
// function that removes it again.
export function stubReducedMotion(initial: boolean) {
  let matches = initial
  const listeners = new Set<() => void>()
  window.matchMedia = (query: string) =>
    ({
      get matches() {
        return query === '(prefers-reduced-motion: reduce)' && matches
      },
      media: query,
      addEventListener: (_: string, listener: () => void) => listeners.add(listener),
      removeEventListener: (_: string, listener: () => void) => listeners.delete(listener),
    }) as unknown as MediaQueryList

  return {
    set(next: boolean) {
      matches = next
      listeners.forEach((listener) => listener())
    },
    restore() {
      Reflect.deleteProperty(window, 'matchMedia')
    },
  }
}
