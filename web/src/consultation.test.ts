import { describe, expect, it } from 'vitest'
import { joinMessage, joinWindow } from './consultation'
import { appointment } from './test/fixtures'
import { translator } from './test/translate'

const id = 'e1000000-0000-4000-8000-000000000001'
// 09:00 to 09:30 UTC: the window opens at 08:50 and closes at 10:00.
const booked = appointment(id, '2026-11-09T09:00:00Z', '2026-11-09T09:30:00Z')
const at = (iso: string) => new Date(iso)

describe('joinWindow', () => {
  it.each([
    ['2026-11-09T08:49:59Z', 'before'],
    ['2026-11-09T08:50:00Z', 'open'],
    ['2026-11-09T09:15:00Z', 'open'],
    ['2026-11-09T09:59:59Z', 'open'],
    ['2026-11-09T10:00:00Z', 'closed'],
  ])('at %s is %s for a booked appointment', (now, want) => {
    expect(joinWindow(booked, at(now))).toBe(want)
  })

  it.each(['cancelled', 'fulfilled', 'noshow'] as const)('is closed for a %s appointment, also inside the window', (status) => {
    expect(joinWindow({ ...booked, status }, at('2026-11-09T09:15:00Z'))).toBe('closed')
  })
})

describe('joinMessage', () => {
  it.each([
    ['join_window_not_open', 'This consultation opens 10 minutes before the start.'],
    ['consultation_ended', 'This consultation has ended.'],
    ['practitioner_did_not_join', 'The time to join this consultation has passed.'],
    ['consent_required', "The patient's consent for this clinic is not active, so the consultation cannot take place."],
    ['consultation_not_started', 'You have not joined this consultation yet, so there is nothing to end.'],
    ['not_found', 'This consultation is not available.'],
    [undefined, 'Something went wrong. Try again.'],
    ['internal error', 'Something went wrong. Try again.'],
  ] as const)('maps %s to its approved text', (code, want) => {
    expect(joinMessage(code, translator('en'))).toBe(want)
  })

  it('writes the text in the UI language', () => {
    expect(joinMessage('consultation_ended', translator('bs'))).toBe('Ova konsultacija je završena.')
    expect(joinMessage(undefined, translator('sr-Cyrl'))).toBe('Нешто није у реду. Покушајте поново.')
  })
})
