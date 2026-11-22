import { describe, expect, it } from 'vitest'
import { appointment } from './test/fixtures'
import { visitModeOf, visitModeURL } from './visitMode'

const at = (extension?: { url: string; valueCode: string }[]) =>
  appointment('e1000000-0000-4000-8000-000000000001', '2026-11-09T09:00:00Z', '2026-11-09T09:30:00Z', { extension })

describe('visitModeOf', () => {
  it('is in person only for an appointment whose extension says so', () => {
    expect(visitModeOf(at([{ url: visitModeURL, valueCode: 'in-person' }]))).toBe('in-person')
    expect(visitModeOf(at([{ url: visitModeURL, valueCode: 'video' }]))).toBe('video')
    expect(visitModeOf(at([{ url: 'https://salus.invalid/other', valueCode: 'in-person' }]))).toBe('video')
    expect(visitModeOf(at())).toBe('video')
  })
})
