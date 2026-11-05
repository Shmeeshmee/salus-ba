import { DisconnectReason } from 'livekit-client'
import { describe, expect, it } from 'vitest'
import { endingOf, levelOf, problemOf } from './call'

describe('endingOf', () => {
  it.each([
    [DisconnectReason.CLIENT_INITIATED, 'left'],
    [DisconnectReason.DUPLICATE_IDENTITY, 'replaced'],
    [DisconnectReason.ROOM_DELETED, 'ended'],
    [DisconnectReason.ROOM_CLOSED, 'ended'],
    [DisconnectReason.PARTICIPANT_REMOVED, 'ended'],
    [DisconnectReason.SIGNAL_CLOSE, 'lost'],
    [DisconnectReason.SERVER_SHUTDOWN, 'lost'],
    [DisconnectReason.JOIN_FAILURE, 'lost'],
    [undefined, 'lost'],
  ])('maps LiveKit reason %s to %s', (reason, want) => {
    expect(endingOf(reason)).toBe(want)
  })
})

describe('problemOf', () => {
  it.each([
    ['NotAllowedError', 'denied'],
    ['SecurityError', 'denied'],
    ['NotFoundError', 'missing'],
    ['OverconstrainedError', 'missing'],
    ['NotReadableError', 'missing'],
  ])('maps a %s to %s', (name, want) => {
    expect(problemOf(new DOMException('synthetic', name))).toBe(want)
  })

  it('takes anything else as no device', () => {
    expect(problemOf(new Error('synthetic'))).toBe('missing')
  })
})

describe('levelOf', () => {
  it('is 0 for silence, which the analyser gives as 128', () => {
    expect(levelOf(new Uint8Array(32).fill(128))).toBe(0)
  })

  it('is the root mean square of the samples, from 0 to 1', () => {
    const loud = Uint8Array.from({ length: 32 }, (_, i) => (i % 2 ? 0 : 255))
    expect(levelOf(loud)).toBeGreaterThan(0.9)
    expect(levelOf(loud)).toBeLessThanOrEqual(1)
    const soft = Uint8Array.from({ length: 32 }, (_, i) => (i % 2 ? 120 : 136))
    expect(levelOf(soft)).toBeCloseTo(8 / 128, 5)
  })
})
