import { DisconnectReason } from 'livekit-client'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { endingOf, levelOf, liveKit, problemOf } from './call'

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

// livekit-client logs its connection steps and errors to the console, which can carry the signalling URL with
// the token (SPEC §2 "Changed after package 6"); call.ts turns its log off.
// FakePeerConnection is just enough of WebRTC for livekit-client, which jsdom lacks, to try the signalling
// connection.
class FakePeerConnection extends EventTarget {
  addTransceiver() {}
  addTrack() {}
  close() {}
}

describe('liveKit', () => {
  it('prints nothing, and no token, when the connection fails', async () => {
    vi.stubGlobal('RTCPeerConnection', FakePeerConnection)
    const methods = ['log', 'info', 'warn', 'error', 'debug', 'trace'] as const
    const spies = methods.map((m) => vi.spyOn(console, m).mockImplementation(() => {}))
    const token = 'synthetic-token-that-must-not-print'

    await expect(liveKit.connect('ws://127.0.0.1:9', token, { camera: false, microphone: false }, () => {})).rejects.toBeTruthy()

    const printed = spies.flatMap((s) => s.mock.calls.map((args) => args.map((a) => (typeof a === 'string' ? a : JSON.stringify(a) ?? String(a))).join(' ')))
    expect(printed.filter((line) => line.includes(token))).toEqual([])
    expect(printed).toEqual([])
  }, 20_000)
})

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
