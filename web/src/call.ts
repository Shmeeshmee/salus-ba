import {
  createLocalAudioTrack,
  createLocalVideoTrack,
  DisconnectReason,
  Room as LiveKitRoom,
  RoomEvent,
  Track,
  type LocalAudioTrack,
  type LocalVideoTrack,
  type RemoteTrack,
} from 'livekit-client'

// call.ts is the one module that reaches LiveKit (livekit-client) and the browser's camera and microphone.
// The screens see only the types below, so they are tested with a fake (src/test/fakeCall.ts). Nothing here
// logs, and nothing about the people in the call goes to LiveKit: the token from our API is all it gets.

export type Kind = 'camera' | 'microphone'

// Device is a camera or microphone the browser offers, with its label (given once access is allowed).
export type Device = { id: string; label: string }

// Problem is why a device could not be opened: access denied, or no such device.
export type Problem = 'denied' | 'missing'

// Preview is a device opened for the check before joining: a camera shown in a video element, a microphone
// measured as a level from 0 to 1 (undefined while it cannot be measured, as before the page has had a click).
export interface Preview {
  attach(el: HTMLVideoElement): void
  level(): number | undefined
  stop(): void
}

export type Opened = { preview: Preview; problem?: undefined } | { problem: Problem; preview?: undefined }

// Peer is the other person in the room, as LiveKit shows them: present, and whether their camera and
// microphone are on.
export type Peer = { present: boolean; camera: boolean; microphone: boolean }

// Ending is why the connection to the room ended: this side left, the room was closed (deleted, or this side
// removed from it), the same person joined from somewhere else, or the network.
export type Ending = 'left' | 'ended' | 'replaced' | 'lost'

export type CallEvent =
  | { type: 'peer'; peer: Peer }
  | { type: 'reconnecting' }
  | { type: 'reconnected' }
  | { type: 'disconnected'; ending: Ending }

// Settings are what to publish on joining, and from which devices.
export type Settings = { camera: boolean; microphone: boolean; cameraId?: string; microphoneId?: string }

export interface Room {
  // show puts the other person's video and this side's own camera into the given elements.
  show(remote: HTMLVideoElement | null, local: HTMLVideoElement | null): void
  // setCamera and setMicrophone turn this side's camera or microphone on or off and answer whether it is on.
  setCamera(on: boolean): Promise<boolean>
  setMicrophone(on: boolean): Promise<boolean>
  leave(): Promise<void>
}

export interface Media {
  devices(kind: Kind): Promise<Device[]>
  open(kind: Kind, deviceId?: string): Promise<Opened>
  connect(url: string, token: string, settings: Settings, onEvent: (event: CallEvent) => void): Promise<Room>
}

// liveKit is Media on livekit-client. No LiveKit UI components; the screens draw their own.
export const liveKit: Media = {
  async devices(kind) {
    const all = await LiveKitRoom.getLocalDevices(kind === 'camera' ? 'videoinput' : 'audioinput', false)
    return all.filter((d) => d.deviceId && d.label).map((d) => ({ id: d.deviceId, label: d.label }))
  },
  async open(kind, deviceId) {
    try {
      const track = kind === 'camera' ? await createLocalVideoTrack({ deviceId }) : await createLocalAudioTrack({ deviceId })
      return { preview: previewOf(track) }
    } catch (err) {
      return { problem: problemOf(err) }
    }
  },
  connect: connectLiveKit,
}

async function connectLiveKit(url: string, token: string, settings: Settings, onEvent: (event: CallEvent) => void): Promise<Room> {
  const room = new LiveKitRoom({
    adaptiveStream: true,
    dynacast: true,
    videoCaptureDefaults: { deviceId: settings.cameraId },
    audioCaptureDefaults: { deviceId: settings.microphoneId },
  })
  let views: { remote: HTMLVideoElement | null; local: HTMLVideoElement | null } = { remote: null, local: null }
  let connected = false
  const audio = new Set<HTMLMediaElement>()
  const peer = () => onEvent({ type: 'peer', peer: peerOf(room) })
  const showRemote = () => {
    const track = [...room.remoteParticipants.values()][0]?.getTrackPublication(Track.Source.Camera)?.track
    if (track && views.remote) track.attach(views.remote)
  }
  const showLocal = () => {
    const track = room.localParticipant.getTrackPublication(Track.Source.Camera)?.track
    if (track && views.local) track.attach(views.local)
  }
  const subscribed = (track: RemoteTrack) => {
    if (track.kind === Track.Kind.Audio) audio.add(document.body.appendChild(track.attach()))
    else showRemote()
    peer()
  }
  const unsubscribed = (track: RemoteTrack) => {
    for (const el of track.detach()) if (audio.delete(el)) el.remove()
    peer()
  }

  room
    .on(RoomEvent.ParticipantConnected, peer)
    .on(RoomEvent.ParticipantDisconnected, peer)
    .on(RoomEvent.TrackSubscribed, subscribed)
    .on(RoomEvent.TrackUnsubscribed, unsubscribed)
    .on(RoomEvent.TrackMuted, peer)
    .on(RoomEvent.TrackUnmuted, peer)
    .on(RoomEvent.LocalTrackPublished, showLocal)
    .on(RoomEvent.Reconnecting, () => onEvent({ type: 'reconnecting' }))
    .on(RoomEvent.Reconnected, () => onEvent({ type: 'reconnected' }))
    .on(RoomEvent.Disconnected, (reason?: DisconnectReason) => {
      for (const el of audio) el.remove()
      audio.clear()
      if (connected) onEvent({ type: 'disconnected', ending: endingOf(reason) })
    })

  try {
    await room.connect(url, token)
  } catch (err) {
    await room.disconnect()
    throw err
  }
  connected = true
  // A device that fails now leaves that side off; the call goes on with what remains.
  if (settings.microphone) await room.localParticipant.setMicrophoneEnabled(true).catch(() => undefined)
  if (settings.camera) await room.localParticipant.setCameraEnabled(true).catch(() => undefined)
  peer()

  return {
    show(remote, local) {
      views = { remote, local }
      showRemote()
      showLocal()
    },
    async setCamera(on) {
      await room.localParticipant.setCameraEnabled(on).catch(() => undefined)
      return room.localParticipant.isCameraEnabled
    },
    async setMicrophone(on) {
      await room.localParticipant.setMicrophoneEnabled(on).catch(() => undefined)
      return room.localParticipant.isMicrophoneEnabled
    },
    leave: () => room.disconnect(),
  }
}

// peerOf is the other participant: the room holds at most two (SPEC §2 "Rooms").
function peerOf(room: LiveKitRoom): Peer {
  const p = [...room.remoteParticipants.values()][0]
  if (!p) return { present: false, camera: false, microphone: false }
  const on = (source: Track.Source) => {
    const publication = p.getTrackPublication(source)
    return publication !== undefined && !publication.isMuted
  }
  return { present: true, camera: on(Track.Source.Camera), microphone: on(Track.Source.Microphone) }
}

function previewOf(track: LocalVideoTrack | LocalAudioTrack): Preview {
  if (track.kind !== Track.Kind.Audio) {
    return { attach: (el) => void track.attach(el), level: () => undefined, stop: () => track.stop() }
  }
  const context = new AudioContext()
  const analyser = context.createAnalyser()
  analyser.fftSize = 512
  context.createMediaStreamSource(new MediaStream([track.mediaStreamTrack])).connect(analyser)
  const samples = new Uint8Array(analyser.fftSize)
  return {
    attach: () => {},
    level: () => {
      if (context.state !== 'running') {
        void context.resume().catch(() => undefined)
        return undefined
      }
      analyser.getByteTimeDomainData(samples)
      return levelOf(samples)
    },
    stop: () => {
      track.stop()
      void context.close().catch(() => undefined)
    },
  }
}

// endingOf maps LiveKit's reason for a disconnection to what the screen does about it.
export function endingOf(reason?: DisconnectReason): Ending {
  switch (reason) {
    case DisconnectReason.CLIENT_INITIATED:
      return 'left'
    case DisconnectReason.DUPLICATE_IDENTITY:
      return 'replaced'
    case DisconnectReason.ROOM_DELETED:
    case DisconnectReason.ROOM_CLOSED:
    case DisconnectReason.PARTICIPANT_REMOVED:
      return 'ended'
    default:
      return 'lost'
  }
}

// problemOf is why getUserMedia refused a device: denied by the user or the browser, or anything else, which
// the screen shows as no device.
export function problemOf(err: unknown): Problem {
  const name = typeof err === 'object' && err !== null && 'name' in err ? err.name : undefined
  return name === 'NotAllowedError' || name === 'PermissionDeniedError' || name === 'SecurityError' ? 'denied' : 'missing'
}

// levelOf is the root mean square of an analyser's time-domain bytes (silence is 128), from 0 to 1.
export function levelOf(samples: Uint8Array): number {
  let sum = 0
  for (const s of samples) sum += ((s - 128) / 128) ** 2
  return Math.min(1, Math.sqrt(sum / samples.length))
}
