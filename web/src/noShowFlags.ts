import { useEffect, useState } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'

// Flag is a repeat no-show flag: how many no-shows the appointment's patient had at the clinic in its last window_days.
export type Flag = { noshows: number; window_days: number }

// useNoShowFlags is the repeat no-show flags of the practitioner's appointments starting in [from, to), by appointment
// id, read once for a calendar's range (GET /clinic/noshow-flags, the rules of NoShowFlag, narrowed to the
// practitioner shown so nothing of the others' patients is read): none while they load, while the rule is off or when
// they cannot be read, so they never hold the calendar up and never stop a booking.
export function useNoShowFlags(api: ApiClient, practitionerId: string, from: Date, to: Date): Map<string, Flag> {
  const [read, setRead] = useState<{ range: string; flags: Map<string, Flag> }>()
  const start = from.getTime()
  const end = to.getTime()
  const range = `${practitionerId}/${start}/${end}`
  useEffect(() => {
    let current = true
    const query = { from: new Date(start).toISOString(), to: new Date(end).toISOString(), practitioner: practitionerId }
    attempt(api.GET('/clinic/noshow-flags', { params: { query } })).then((res) => {
      if (current && res.ok) setRead({ range: `${practitionerId}/${start}/${end}`, flags: new Map(res.data.flags.map((f) => [f.appointment_id, f])) })
    })
    return () => {
      current = false
    }
  }, [api, practitionerId, start, end])
  // The flags of another week are not this one's: none until this week's are read.
  return read?.range === range ? read.flags : noFlags
}

const noFlags = new Map<string, Flag>()
