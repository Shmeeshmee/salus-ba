import { describe, expect, it } from 'vitest'
import { ecgPath } from './ecgPath'

const FIRST_BEAT =
  'M0 120 L24 120 C30 120 32 108 39 108 C46 108 48 120 54 120 L66 120 L70 127 L78 20.0 L86 156.0' +
  ' L91 117 L108 116 C120 115 124 86.0 138 86.0 C152 86.0 156 120 168 120 L200 120'

describe('ecgPath', () => {
  it('starts at the baseline with the first PQRST beat', () => {
    expect(ecgPath().startsWith(FIRST_BEAT + ' ')).toBe(true)
  })

  it('draws twelve beats across the 2400-wide trace', () => {
    const rPeaks = ecgPath().match(/ L\d+ [12]\d\.\d/g) ?? []

    expect(rPeaks).toHaveLength(12)
    expect(ecgPath().endsWith(' L2400 120')).toBe(true)
  })

  it('is deterministic', () => {
    expect(ecgPath()).toBe(ecgPath())
  })
})
