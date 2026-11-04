import { useEffect, useState, type DependencyList } from 'react'

export type Load<T> = { name: 'loading' } | { name: 'failed'; error: unknown } | { name: 'loaded'; value: T }

type Result<T> = { deps: DependencyList; reads: number; state: Load<T> }

// useLoad runs load when deps change, and again on reload. Until the answer for the current deps arrives the
// state is loading; an answer for earlier deps is dropped. set replaces the loaded value (after a save).
export function useLoad<T>(load: () => Promise<T>, deps: DependencyList) {
  const [result, setResult] = useState<Result<T>>()
  const [reads, setReads] = useState(0)

  useEffect(() => {
    let current = true
    load().then(
      (value) => current && setResult({ deps, reads, state: { name: 'loaded', value } }),
      (error: unknown) => current && setResult({ deps, reads, state: { name: 'failed', error } }),
    )
    return () => {
      current = false
    }
    // load is a new closure every render; the caller's deps say when it means something new, which the
    // rule cannot check from here.
    // oxlint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, reads])

  const fresh = result !== undefined && result.reads === reads && sameDeps(result.deps, deps)
  return {
    state: fresh ? result.state : ({ name: 'loading' } as Load<T>),
    reload: () => setReads((n) => n + 1),
    set: (value: T) => setResult({ deps, reads, state: { name: 'loaded', value } }),
  }
}

function sameDeps(a: DependencyList, b: DependencyList): boolean {
  return a.length === b.length && a.every((x, i) => Object.is(x, b[i]))
}
