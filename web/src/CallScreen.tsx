import { useEffect, useRef, useState, type ReactNode } from 'react'
import type { ApiClient } from './api/client'
import { attempt } from './api/result'
import type { components } from './api/schema'
import { liveKit, type CallEvent, type Media, type Peer, type Room, type Settings } from './call'
import { joinMessage } from './consultation'
import { DeviceCheck } from './DeviceCheck'
import { useT } from './i18n/useT'
import { CameraIcon, LeaveIcon, MicrophoneIcon } from './icons'
import { Alert, Notice } from './ui'

type Status = components['schemas']['ConsultationStatus']

type Phase =
  | { name: 'check'; error?: string }
  | { name: 'joining' }
  | { name: 'call' }
  | { name: 'reconnecting' }
  | { name: 'lost' }
  | { name: 'left'; busy?: boolean; otherTab?: boolean }
  | { name: 'ended'; status?: Status }

// Gone is why the patient is no longer in the room, as the status's reason says: they left, or the server
// removed them because the consent was withdrawn or the appointment cancelled.
type Gone = 'left' | 'consent' | 'cancelled'

const goneOf: Partial<Record<NonNullable<Status['reason']>, Gone>> = { consent_withdrawn: 'consent', cancelled: 'cancelled' }

const nobody: Peer = { present: false, camera: false, microphone: false }

type Props = { api: ApiClient; id: string; media?: Media }

// CallScreen is the practitioner's video consultation (SPEC §2 "Client states"): the device check, joining,
// waiting for the patient, the call, reconnecting, and the end. The token comes from our API on every join and
// rejoin; the server decides whether one is issued. Events of a room this screen has left are ignored
// (generation), so a late event of an old connection changes nothing.
export function CallScreen({ api, id, media = liveKit }: Props) {
  const t = useT()
  const [phase, setPhase] = useState<Phase>({ name: 'check' })
  const [room, setRoom] = useState<Room>()
  const [peer, setPeer] = useState<Peer>(nobody)
  const [peerName, setPeerName] = useState('')
  const [gone, setGone] = useState<Gone>()
  const [local, setLocal] = useState({ camera: false, microphone: false })
  const [cameraUsable, setCameraUsable] = useState(false)
  const [patientJoined, setPatientJoined] = useState(false)
  const [reconnected, setReconnected] = useState(false)
  const generation = useRef(0)
  const settings = useRef<Settings>({ camera: false, microphone: false })
  const present = useRef(false)
  const params = { params: { path: { id } } }

  useEffect(() => () => void room?.leave(), [room])
  // Unmounting makes every connection still in flight late, so a room that connects afterwards is left.
  useEffect(
    () => () => {
      generation.current++
    },
    [],
  )
  useEffect(() => {
    if (!reconnected) return
    const timer = setTimeout(() => setReconnected(false), 4000)
    return () => clearTimeout(timer)
  }, [reconnected])

  async function readStatus(): Promise<Status | undefined> {
    const res = await attempt(api.GET('/appointments/{id}/consultation', params))
    return res.ok ? res.data : undefined
  }

  // connect asks for a token and joins the room; again is a rejoin after the connection was lost.
  async function connect(again: boolean) {
    const gen = ++generation.current
    const res = await attempt(api.POST('/appointments/{id}/consultation/join', params))
    if (gen !== generation.current) return
    if (!res.ok) return refused(again, res.code)
    let joined: Room
    try {
      joined = await media.connect(res.data.url, res.data.token, { ...settings.current }, (e) => gen === generation.current && onEvent(e))
    } catch {
      if (gen === generation.current) setPhase(again ? { name: 'lost' } : { name: 'check', error: t('consultation.cannotConnect') })
      return
    }
    if (gen !== generation.current) return void joined.leave()
    setRoom(joined)
    setPeerName(res.data.peerName)
    setLocal({ camera: settings.current.camera, microphone: settings.current.microphone })
    setPhase({ name: 'call' })
    setReconnected(again)
    const status = await readStatus()
    if (status?.patientJoinedAt) setPatientJoined(true)
  }

  function refused(again: boolean, code?: Parameters<typeof joinMessage>[0]) {
    if (again && !code) return setPhase({ name: 'lost' })
    if (again && (code === 'consultation_ended' || code === 'practitioner_did_not_join')) return void finish(false)
    setPhase({ name: 'check', error: joinMessage(code, t) })
  }

  // finish shows the end once the room is gone. A room closed while the consultation is still open is rejoined
  // (rejoin); after a refused rejoin it is not: the window has closed while the room is still occupied, so the
  // screen shows the end without the status's line (SPEC §2 "Client states": rejoin only while the window is open).
  async function finish(rejoin = true) {
    const gen = ++generation.current
    const status = await readStatus()
    if (gen !== generation.current) return
    if (status && status.state !== 'ended') return rejoin ? void connect(true) : setPhase({ name: 'ended' })
    setPhase({ name: 'ended', status })
  }

  function onEvent(e: CallEvent) {
    if (e.type === 'peer') return onPeer(e.peer)
    if (e.type === 'reconnecting') {
      setReconnected(false)
      return setPhase({ name: 'reconnecting' })
    }
    if (e.type === 'reconnected') {
      setReconnected(true)
      return setPhase({ name: 'call' })
    }
    if (e.ending === 'left') return
    setRoom(undefined)
    setPeer(nobody)
    if (e.ending === 'replaced') {
      generation.current++
      return setPhase({ name: 'left', otherTab: true })
    }
    if (e.ending === 'ended') return void finish()
    setPhase({ name: 'reconnecting' })
    void connect(true)
  }

  // onPeer follows the patient; when they go (present before, not now), the status says why, read once.
  function onPeer(p: Peer) {
    setPeer(p)
    const was = present.current
    present.current = p.present
    if (p.present) {
      setPatientJoined(true)
      return setGone(undefined)
    }
    if (!was) return
    setGone('left')
    void readStatus().then((status) => {
      const gone = status?.reason && goneOf[status.reason]
      if (gone) setGone((g) => (g === 'left' ? gone : g))
    })
  }

  function join(s: Settings, camera: boolean) {
    settings.current = s
    setCameraUsable(camera)
    setPhase({ name: 'joining' })
    void connect(false)
  }

  function leave() {
    generation.current++
    setRoom(undefined)
    setPeer(nobody)
    setPhase({ name: 'left' })
  }

  async function toggle(kind: 'camera' | 'microphone') {
    if (!room) return
    const on = await (kind === 'camera' ? room.setCamera(!local.camera) : room.setMicrophone(!local.microphone))
    settings.current = { ...settings.current, [kind]: on }
    setLocal((l) => ({ ...l, [kind]: on }))
  }

  async function end(): Promise<string | undefined> {
    const res = await attempt(api.POST('/appointments/{id}/consultation/end', params))
    if (!res.ok) return joinMessage(res.code, t)
    generation.current++
    setRoom(undefined)
    setPhase({ name: 'ended', status: res.data })
  }

  switch (phase.name) {
    case 'check':
    case 'joining':
      return <DeviceCheck media={media} busy={phase.name === 'joining'} error={phase.name === 'check' ? phase.error : undefined} onJoin={join} />
    case 'left':
      return (
        <Message text={t(phase.otherTab ? 'consultation.otherTab' : 'consultation.left')}>
          <button
            type="button"
            className="primary"
            disabled={phase.busy}
            onClick={() => {
              setPhase({ name: 'left', busy: true, otherTab: phase.otherTab })
              void connect(false)
            }}
          >
            {t('consultation.joinAgain')}
          </button>
        </Message>
      )
    case 'lost':
      return (
        <Message text={t('consultation.notRestored')}>
          <button
            type="button"
            className="primary"
            onClick={() => {
              setPhase({ name: 'reconnecting' })
              void connect(true)
            }}
          >
            {t('consultation.tryAgain')}
          </button>
        </Message>
      )
    case 'ended':
      return <Ended status={phase.status} />
  }

  return (
    <section className="call">
      <Stage room={room} peer={peer} name={peerName} gone={gone} reconnecting={phase.name === 'reconnecting'} cameraOn={local.camera} />
      <Notice message={reconnected ? t('consultation.reconnected') : undefined} />
      <div className="call-controls">
        <button
          type="button"
          className={`round${local.microphone ? '' : ' off'}`}
          aria-label={t(local.microphone ? 'consultation.mute' : 'consultation.unmute')}
          title={t(local.microphone ? 'consultation.mute' : 'consultation.unmute')}
          disabled={!room}
          onClick={() => toggle('microphone')}
        >
          <MicrophoneIcon off={!local.microphone} />
        </button>
        <button
          type="button"
          className={`round${local.camera ? '' : ' off'}`}
          aria-label={t(local.camera ? 'consultation.cameraOff' : 'consultation.cameraOn')}
          title={t(local.camera ? 'consultation.cameraOff' : 'consultation.cameraOn')}
          disabled={!room || !cameraUsable}
          onClick={() => toggle('camera')}
        >
          <CameraIcon off={!local.camera} />
        </button>
        <button type="button" className="leave" onClick={leave}>
          <LeaveIcon />
          {t('consultation.leave')}
        </button>
        <EndConsultation patientJoined={patientJoined} onEnd={end} />
      </div>
    </section>
  )
}

type StageProps = { room?: Room; peer: Peer; name: string; gone?: Gone; reconnecting: boolean; cameraOn: boolean }

// Stage is the other person's video, large, with this side's own camera small in a corner. LiveKit's tracks are
// put into the two video elements by the room. Lines that need the patient's name are left out without one.
function Stage({ room, peer, name, gone, reconnecting, cameraOn }: StageProps) {
  const t = useT()
  const remote = useRef<HTMLVideoElement>(null)
  const own = useRef<HTMLVideoElement>(null)
  useEffect(() => room?.show(remote.current, own.current), [room])

  return (
    <div className="stage">
      <video ref={remote} className={peer.present && peer.camera ? '' : 'hidden'} playsInline autoPlay />
      <div className="stage-text">
        {!peer.present && !reconnecting && <Absent name={name} gone={gone} />}
        {peer.present && !peer.camera && name && <p>{t('consultation.peerCameraOff', { name })}</p>}
      </div>
      {peer.present && !peer.microphone && name && <p className="badge">{t('consultation.peerMuted', { name })}</p>}
      <video ref={own} className={cameraOn ? 'own' : 'own hidden'} muted playsInline autoPlay />
      {reconnecting && (
        <p className="reconnecting" role="status">
          {t('consultation.reconnecting')}
        </p>
      )}
    </div>
  )
}

function Absent({ name, gone }: { name: string; gone?: Gone }) {
  const t = useT()
  if (gone === 'consent') return <p>{t('consultation.peerWithdrewConsent')}</p>
  if (gone === 'cancelled') return <p>{t('consultation.cancelled')}</p>
  if (gone === 'left') return name ? <p>{t('consultation.peerLeft', { name })}</p> : null
  return (
    <>
      <h2>{t('consultation.waiting')}</h2>
      {name && <p>{t('consultation.notJoinedYet', { name })}</p>}
    </>
  )
}

// EndConsultation asks to confirm, saying what the end does to the appointment, then ends it for both sides.
function EndConsultation({ patientJoined, onEnd }: { patientJoined: boolean; onEnd: () => Promise<string | undefined> }) {
  const t = useT()
  const [confirming, setConfirming] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()

  async function end() {
    setBusy(true)
    setError(undefined)
    const message = await onEnd()
    setBusy(false)
    setError(message)
  }

  if (!confirming) {
    return (
      <button type="button" className="danger" onClick={() => setConfirming(true)}>
        {t('consultation.end')}
      </button>
    )
  }
  return (
    <div className="confirm end-confirm">
      <p className="strong">{t('consultation.endQuestion')}</p>
      <p>{t(patientJoined ? 'consultation.endFulfilled' : 'consultation.endNoShow')}</p>
      <Alert message={error} />
      <div className="actions">
        <button type="button" className="danger" disabled={busy} onClick={end}>
          {t('consultation.end')}
        </button>
        <button type="button" disabled={busy} onClick={() => setConfirming(false)}>
          {t('consultation.keep')}
        </button>
      </div>
    </div>
  )
}

function Message({ text: message, children }: { text: string; children: ReactNode }) {
  return (
    <section className="panel call-message">
      <p>{message}</p>
      <div className="actions">{children}</div>
    </section>
  )
}

function Ended({ status }: { status?: Status }) {
  const t = useT()
  return (
    <section className="panel call-message">
      <h1>{t('consultation.ended')}</h1>
      {status && <p>{t(endLine(status))}</p>}
    </section>
  )
}

// endLine is the message id of how the consultation ended: a cancellation by the patient's revocation (the
// reason) is the withdrawn consent.
function endLine(status: Status): string {
  if (status.endReason === 'cancelled') return status.reason === 'consent_withdrawn' ? 'consultation.peerWithdrewConsent' : 'consultation.cancelled'
  if (status.endReason === 'time_limit') return 'consultation.timeLimit'
  return status.patientJoinedAt ? 'consultation.markedFulfilled' : 'consultation.markedNoShow'
}
