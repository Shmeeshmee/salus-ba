import { describe, expect, it } from 'vitest'
import type { components } from '../api/schema'
import { catalogs } from './catalog'
import { staffLanguages } from './languages'

type Status = components['schemas']['Appointment']['status']

// Every appointment status the API can return, checked against the contract's enum at type level: a status added to
// api/openapi.yaml fails the typecheck here until it is listed (docs/clinic-cms/walk/wave3, finding 3).
const statuses = ['booked', 'arrived', 'cancelled', 'fulfilled', 'noshow'] as const satisfies readonly Status[]
type Unlisted = Exclude<Status, (typeof statuses)[number]>
const everyStatus: [Unlisted] extends [never] ? true : never = true

describe('the calendar catalogs', () => {
  it('label every appointment status the API can return, in every staff language', () => {
    expect(everyStatus).toBe(true)
    for (const language of staffLanguages) {
      for (const status of statuses) {
        expect(catalogs.calendar?.[language]?.[`status.${status}`], `calendar/${language}.json status.${status}`).toBeTruthy()
      }
    }
  })
})
