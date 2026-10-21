import { useCallback, useEffect, useState } from 'react'
import { usePrefersReducedMotion } from './usePrefersReducedMotion'

type CycleOptions = {
  interval: number
  initialDelay?: number
}

type Cycle = {
  index: number
  // Counts every change, including a pick of the item already shown, so callers can restart an animation.
  step: number
  select: (index: number) => void
}

// Steps through count items on a timer. Pauses while the tab is hidden and stays put when motion is reduced;
// a manual pick restarts the timer.
export function useCycle(count: number, { interval, initialDelay = 0 }: CycleOptions): Cycle {
  const reduced = usePrefersReducedMotion()
  const [shown, setShown] = useState({ index: 0, step: 1 })
  const [run, setRun] = useState({ id: 0, delay: initialDelay })

  useEffect(() => {
    if (reduced) return
    let timer: ReturnType<typeof setInterval> | undefined
    const tick = () => {
      if (!document.hidden) setShown((s) => ({ index: (s.index + 1) % count, step: s.step + 1 }))
    }
    const start = setTimeout(() => {
      timer = setInterval(tick, interval)
    }, run.delay)
    return () => {
      clearTimeout(start)
      clearInterval(timer)
    }
  }, [reduced, count, interval, run])

  const select = useCallback(
    (index: number) => {
      setShown((s) => ({ index: ((index % count) + count) % count, step: s.step + 1 }))
      setRun((r) => ({ id: r.id + 1, delay: 0 }))
    },
    [count],
  )

  return { index: shown.index, step: shown.step, select }
}
