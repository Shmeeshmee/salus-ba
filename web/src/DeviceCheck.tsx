import { useEffect, useRef, useState } from 'react'
import type { Device, Kind, Media, Opened, Preview, Settings } from './call'
import { consultation as text } from './messages'
import { Alert } from './ui'

type Props = { media: Media; busy: boolean; error?: string; onJoin: (settings: Settings, camera: boolean) => void }

// DeviceCheck is the step before joining (SPEC §2 "Client states"): the camera's preview, the microphone's
// level, a picker for each, and what the browser does not allow. Join waits until both devices have opened or
// failed, then the user joins with what remains; onJoin gets the settings and whether a camera is there to turn
// on later.
export function DeviceCheck({ media, busy, error, onJoin }: Props) {
  const camera = useDevice(media, 'camera')
  const microphone = useDevice(media, 'microphone')
  const [cameraOff, setCameraOff] = useState(false)
  const [microphoneOff, setMicrophoneOff] = useState(false)
  const { level, silent } = useLevel(microphone.opened)

  function join() {
    const settings: Settings = { camera: !cameraOff && camera.usable, microphone: !microphoneOff && microphone.usable }
    if (camera.usable) settings.cameraId = camera.chosen
    if (microphone.usable) settings.microphoneId = microphone.chosen
    onJoin(settings, camera.usable)
  }

  return (
    <section className="panel device-check">
      <h1>{text.checkTitle}</h1>
      <p className="muted">{text.checkIntro}</p>
      <div className="check-media">
        <div className="preview">
          {camera.opened?.preview && <CameraPreview preview={camera.opened.preview} />}
        </div>
        <div className="check-devices">
          <Picker kind="camera" state={camera} />
          <Problem kind="camera" problem={camera.opened?.problem} />
          <Picker kind="microphone" state={microphone} />
          <Problem kind="microphone" problem={microphone.opened?.problem} />
          {microphone.opened?.preview && (
            <>
              <meter className="level" aria-label={text.microphone} min={0} max={1} value={level} />
              {silent && <p className="hint">{text.noSound}</p>}
            </>
          )}
          <label className="check-option">
            <input type="checkbox" checked={cameraOff || !camera.usable} disabled={!camera.usable} onChange={(e) => setCameraOff(e.target.checked)} />
            {text.joinCameraOff}
          </label>
          <label className="check-option">
            <input
              type="checkbox"
              checked={microphoneOff || !microphone.usable}
              disabled={!microphone.usable}
              onChange={(e) => setMicrophoneOff(e.target.checked)}
            />
            {text.joinMicrophoneOff}
          </label>
          <Alert message={error} />
          <div className="actions">
            <button type="button" className="primary" disabled={busy || !camera.opened || !microphone.opened} onClick={join}>
              {busy ? text.joining : text.joinNow}
            </button>
          </div>
        </div>
      </div>
    </section>
  )
}

function CameraPreview({ preview }: { preview: Preview }) {
  const ref = useRef<HTMLVideoElement>(null)
  useEffect(() => {
    if (ref.current) preview.attach(ref.current)
  }, [preview])
  return <video ref={ref} muted playsInline autoPlay />
}

type DeviceState = { opened?: Opened; devices: Device[]; chosen?: string; usable: boolean; choose: (id: string) => void }

// useDevice opens the camera or microphone for the preview (the browser's default until one is picked), stops it
// when another is picked or the check is left, and lists the devices once one has opened (browsers name them
// only then).
function useDevice(media: Media, kind: Kind): DeviceState {
  const [picked, setPicked] = useState<string>()
  const [opened, setOpened] = useState<Opened>()
  const [devices, setDevices] = useState<Device[]>([])

  useEffect(() => {
    let current = true
    let stop = () => {}
    media.open(kind, picked).then(async (o) => {
      if (!current) return o.preview?.stop()
      stop = () => o.preview?.stop()
      setOpened(o)
      const list = o.preview ? await media.devices(kind) : []
      if (current) setDevices(list)
    })
    return () => {
      current = false
      stop()
    }
  }, [media, kind, picked])

  return { opened, devices, chosen: picked ?? devices[0]?.id, usable: opened?.preview !== undefined, choose: setPicked }
}

const silence = 0.02
const silentPolls = 50 // 5 seconds at one reading every 100 ms

// useLevel reads the microphone's level ten times a second, and says when it has stayed silent for 5 seconds. A
// level that cannot be measured is neither sound nor silence.
function useLevel(opened?: Opened) {
  const [level, setLevel] = useState(0)
  const [quiet, setQuiet] = useState(0)
  useEffect(() => {
    const preview = opened?.preview
    if (!preview) return
    const timer = setInterval(() => {
      const l = preview.level()
      if (l === undefined) return
      setLevel(l)
      setQuiet((n) => (l < silence ? n + 1 : 0))
    }, 100)
    return () => clearInterval(timer)
  }, [opened])
  return { level, silent: quiet >= silentPolls }
}

function Picker({ kind, state }: { kind: Kind; state: DeviceState }) {
  if (state.devices.length === 0) return null
  const id = `device-${kind}`
  return (
    <div className="field">
      <label htmlFor={id}>{kind === 'camera' ? text.camera : text.microphone}</label>
      <select id={id} value={state.chosen ?? ''} onChange={(e) => state.choose(e.target.value)}>
        {state.devices.map((d) => (
          <option key={d.id} value={d.id}>
            {d.label}
          </option>
        ))}
      </select>
    </div>
  )
}

const problems = {
  camera: { denied: text.cameraDenied, missing: text.noCamera },
  microphone: { denied: text.microphoneDenied, missing: text.noMicrophone },
} as const

function Problem({ kind, problem }: { kind: Kind; problem?: Opened['problem'] }) {
  if (!problem) return null
  return <p className="alert">{problems[kind][problem]}</p>
}
