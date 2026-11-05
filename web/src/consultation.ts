import { useEffect, useState } from 'react'
import type { ErrorCode } from './api/client'
import type { Appointment } from './api/fhir'
import { consultation as text, somethingWrong } from './messages'

const minute = 60_000

export type Window = 'before' | 'open' | 'closed'

// joinWindow is where now stands against a booked appointment's join window: from 10 minutes before the start
// to 30 minutes after the end (SPEC §2 "Join button"). It only decides whether the button shows; the server
// decides whether a token is issued.
export function joinWindow(appt: Appointment, now: Date): Window {
  if (appt.status !== 'booked') return 'closed'
  const t = now.getTime()
  if (t < new Date(appt.start).getTime() - 10 * minute) return 'before'
  if (t >= new Date(appt.end).getTime() + 30 * minute) return 'closed'
  return 'open'
}

// joinMessage is the approved sentence for a refused join or end.
export function joinMessage(code?: ErrorCode): string {
  switch (code) {
    case 'join_window_not_open':
      return text.windowNotOpen
    case 'consultation_ended':
      return text.consultationEnded
    case 'practitioner_did_not_join':
      return text.timePassed
    case 'consent_required':
      return text.consentRequired
    case 'consultation_not_started':
      return text.notStarted
    case 'not_found':
      return text.notAvailable
    default:
      return somethingWrong
  }
}

// useNow is now(), read again every 15 seconds, so a join button appears and goes while its screen is open.
export function useNow(now: () => Date): Date {
  const [, setTick] = useState(0)
  useEffect(() => {
    const timer = setInterval(() => setTick((n) => n + 1), 15_000)
    return () => clearInterval(timer)
  }, [])
  return now()
}
