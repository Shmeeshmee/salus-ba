import type { CallEvent, Device, Kind, Media, Opened, Preview, Room, Settings } from '../call'

// fakeMedia stands in for call.ts: devices that open (or fail as told), a microphone at a fixed level, and
// rooms the test drives by emitting LiveKit's events as call.ts maps them.
export type FakeRoom = Room & {
  url: string
  token: string
  settings: Settings
  emit: (event: CallEvent) => void
  camera: boolean[] // every setCamera call
  microphone: boolean[]
  left: boolean
  shown: boolean
}

// level: the microphone's level, null when it cannot be measured.
type Options = { camera?: Opened['problem']; microphone?: Opened['problem']; level?: number | null; connectFails?: boolean }

export const devices: Record<Kind, Device[]> = {
  camera: [
    { id: 'cam-1', label: 'Synthetic camera' },
    { id: 'cam-2', label: 'Second synthetic camera' },
  ],
  microphone: [{ id: 'mic-1', label: 'Synthetic microphone' }],
}

export function fakeMedia(options: Options = {}) {
  const rooms: FakeRoom[] = []
  const opened: { kind: Kind; deviceId?: string }[] = []
  const previews: { kind: Kind; attached: number; stopped: boolean }[] = []

  function preview(kind: Kind): Preview {
    const p = { kind, attached: 0, stopped: false }
    previews.push(p)
    return {
      attach: () => void p.attached++,
      level: () => (options.level === null ? undefined : (options.level ?? 0.4)),
      stop: () => void (p.stopped = true),
    }
  }

  const media: Media = {
    devices: async (kind) => (options[kind] ? [] : devices[kind]),
    open: async (kind, deviceId) => {
      opened.push({ kind, deviceId })
      const problem = options[kind]
      return problem ? { problem } : { preview: preview(kind) }
    },
    connect: async (url, token, settings, onEvent) => {
      if (options.connectFails) throw new Error('synthetic connection failure')
      const room: FakeRoom = {
        url,
        token,
        settings,
        emit: onEvent,
        camera: [],
        microphone: [],
        left: false,
        shown: false,
        show: () => void (room.shown = true),
        setCamera: async (on) => {
          room.camera.push(on)
          return on
        },
        setMicrophone: async (on) => {
          room.microphone.push(on)
          return on
        },
        leave: async () => void (room.left = true),
      }
      rooms.push(room)
      return room
    },
  }

  return { media, rooms, opened, previews, last: () => rooms[rooms.length - 1] }
}
