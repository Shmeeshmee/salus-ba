// Fixed messages shared by the screens. API error codes are mapped to plain sentences; nothing from a
// response body is shown.
export const somethingWrong = 'Something went wrong. Try again.'
export const wrongCode = 'The code is wrong. Check the time on your device and try again.'

export function tooManyAttempts(retryAfter: string | null): string {
  const seconds = Number(retryAfter)
  if (!Number.isFinite(seconds) || seconds <= 0) return 'Too many attempts. Try again later.'
  const minutes = Math.ceil(seconds / 60)
  return `Too many attempts. Try again in ${minutes} ${minutes === 1 ? 'minute' : 'minutes'}.`
}

export const statusLabels = { booked: 'Booked', cancelled: 'Cancelled', fulfilled: 'Fulfilled', noshow: 'No-show' } as const

// The video consultation (docs/consultation/COPY.md, approved by the owner, word for word; the W numbers are
// COPY.md's). [Name] is the other person's name from the join response.
export const consultation = {
  join: 'Join consultation', // W1
  joinFrom: 'You can join from 10 minutes before the start.', // W2
  checkTitle: 'Check your camera and microphone', // W3
  checkIntro: 'The patient will see and hear you once you join.', // W4
  camera: 'Camera', // W5
  microphone: 'Microphone', // W5
  noSound: 'No sound detected. Check that your microphone is on and not muted.', // W6
  joinCameraOff: 'Join with camera off', // W7
  joinMicrophoneOff: 'Join with microphone off', // W7
  joinNow: 'Join', // W8
  joining: 'Joining…', // W9
  cameraDenied: 'Your browser is not allowing the camera. Allow it in the site settings, then reload the page. You can still join without video.', // W10
  microphoneDenied:
    'Your browser is not allowing the microphone. Allow it in the site settings, then reload the page. Without it, the patient cannot hear you.', // W11
  noCamera: 'No camera found.', // W12
  noMicrophone: 'No microphone found.', // W12
  waiting: 'Waiting for the patient', // W13
  notJoinedYet: (name: string) => `${name} has not joined yet.`, // W14
  mute: 'Mute microphone', // W15
  unmute: 'Unmute microphone', // W15
  cameraOff: 'Turn camera off', // W15
  cameraOn: 'Turn camera on', // W15
  leave: 'Leave', // W15
  peerCameraOff: (name: string) => `${name}'s camera is off`, // W16
  peerMuted: (name: string) => `${name} is muted`, // W17
  peerLeft: (name: string) => `${name} has left the consultation. They can join again until the consultation ends.`, // W18
  peerWithdrewConsent: 'The patient withdrew consent for this clinic, so the consultation has ended.', // W18a
  cancelled: 'This appointment was cancelled, so the consultation has ended.', // W18b
  end: 'End consultation', // W19, W22
  endQuestion: 'End the consultation?', // W20
  endFulfilled: 'The patient will be disconnected and cannot join again. The appointment will be marked as fulfilled.', // W21
  endNoShow: 'The patient has not joined. The appointment will be marked as a no-show.', // W21b
  keep: 'Cancel', // W22
  left: 'You left the consultation. You can join again until it ends.', // W23
  joinAgain: 'Join again', // W23
  reconnecting: 'Connection lost. Reconnecting…', // W24
  notRestored: 'The connection could not be restored.', // W25
  tryAgain: 'Try again', // W25
  reconnected: 'Reconnected', // W26
  ended: 'The consultation has ended', // W27
  markedFulfilled: 'The appointment is marked as fulfilled.', // W28
  markedNoShow: 'The appointment is marked as a no-show.', // W28
  timeLimit: 'The consultation reached its time limit and has ended.', // W29
  windowNotOpen: 'This consultation opens 10 minutes before the start.', // W30
  consultationEnded: 'This consultation has ended.', // W31
  consentRequired: "The patient's consent for this clinic is not active, so the consultation cannot take place.", // W32
  notStarted: 'You have not joined this consultation yet, so there is nothing to end.', // W32b
  notAvailable: 'This consultation is not available.', // W33
  cannotConnect: 'The video connection could not be started. Check your connection and try again.', // W34
  title: 'Consultation', // W35
  states: { not_started: 'Not started', open: 'Open', ended: 'Ended' }, // W36
  patientJoinedAt: (time: string) => `Patient joined at ${time}`, // W37
  patientDidNotJoin: 'Patient did not join', // W37
  practitionerJoinedAt: (time: string) => `Practitioner joined at ${time}`, // W38
  practitionerDidNotJoin: 'Practitioner did not join', // W38
  timePassed: 'The time to join this consultation has passed.', // W39
} as const
