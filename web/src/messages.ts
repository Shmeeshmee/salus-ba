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
  otherTab: 'The consultation is open in another tab or window.', // W40
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

// Questionnaires: the library and the builder (docs/questionnaire/COPY.md, approved by the owner, word for
// word; the W numbers are COPY.md's). [Item] is the item's text, or its type (W11) while it has none.
export const questionnaires = {
  title: 'Questionnaires', // W1
  create: 'New questionnaire', // W2
  filters: { all: 'All', draft: 'Drafts', active: 'Active', retired: 'Retired' }, // W3
  statuses: { draft: 'Draft', active: 'Active', retired: 'Retired' }, // W4
  versionTip: (n: number, date: string) => `Version ${n}, last changed ${date}`, // W5
  emptyLibrary: 'No questionnaires yet. Create one to ask patients questions before their appointments.', // W6
  emptyFilter: 'No questionnaires with this status.', // W7
  untitled: 'Untitled questionnaire', // W8
  titleLabel: 'Title', // W9
  descriptionLabel: 'Description (optional)', // W9
  addItem: 'Add an item', // W10
  addItemHint: 'Drag an item into the questionnaire, or select it to add it at the end.', // W10
  kinds: {
    group: 'Section',
    display: 'Instructions',
    string: 'Short answer',
    text: 'Long answer',
    boolean: 'Yes or no',
    single: 'Single choice',
    multiple: 'Multiple choice',
    integer: 'Whole number',
    decimal: 'Decimal number',
    date: 'Date',
  }, // W11
  empty: 'Drag items here, or select one on the left.', // W12
  question: 'Question', // W13
  instructions: 'Instructions', // W13
  sectionTitle: 'Section title', // W13
  required: 'Required', // W13
  templateHint: "This questionnaire is a template for all patients. Do not include any patient's details.", // W14
  options: 'Options', // W15
  addOption: 'Add option', // W15
  option: (n: number) => `Option ${n}`, // W15
  removeOption: 'Remove option', // W15
  moreOptions: 'More options', // W16
  addHelp: 'Add help text', // W16
  help: 'Help text', // W16
  helpHint: 'Patients see this when they tap the information icon.', // W16
  addCondition: 'Add condition', // W17
  showOnlyIf: 'Show this only if', // W17
  is: 'is', // W17
  removeCondition: 'Remove condition', // W17
  yes: 'Yes', // W38, the answers of a yes or no question
  no: 'No', // W38
  nothingToDependOn: 'A condition needs a yes or no question or a choice question above this item.', // W17
  moveUp: 'Move up', // W18
  moveDown: 'Move down', // W18
  delete: 'Delete', // W18
  drag: 'Drag to reorder', // W18
  deleteQuestion: 'Delete this question?', // W19
  dependents: (n: number) => `${n} items are shown only after an answer to it. Their conditions will be removed.`, // W19
  cancel: 'Cancel', // W19, W21, W22, W24
  save: 'Save draft', // W20
  saving: 'Saving…', // W20
  saved: 'Saved', // W20
  discard: 'Discard draft', // W20
  discardQuestion: 'Discard this draft?', // W21
  discardWarning: 'This cannot be undone.', // W21
  discardConfirm: 'Discard', // W21
  publish: 'Publish', // W22
  publishQuestion: 'Publish this questionnaire?', // W22
  publishWarning: 'Once published, it cannot be changed. To change it later, you create a new version.', // W22
  publishRetires: (n: number) => `Version ${n} will be retired. Patients who have already been asked keep that version.`, // W22
  published: 'This questionnaire is published and cannot be changed.', // W23
  revise: 'Edit as new version', // W23
  retire: 'Retire', // W23, W24
  retireQuestion: 'Retire this questionnaire?', // W24
  retireWarning:
    'It is no longer sent with new bookings or added to appointments, including as a default questionnaire. Patients who have already been asked can still submit it.', // W24
  retired: 'This questionnaire is retired. It is no longer sent to patients.', // W25
  leaveQuestion: 'You have unsaved changes. Leave without saving?', // W26
  leave: 'Leave', // W26
  stay: 'Stay', // W26
  itemCount: (n: number) => `${n} of 200 items`, // W27
  tooManyItems: 'A questionnaire can have up to 200 items.', // W27
  characters: (n: number, max: number) => `${n} of ${max} characters`, // W27
  fewOptions: 'A choice question needs at least 2 options.', // W27
  manyOptions: 'A choice question can have up to 50 options.', // W27
  sameOptions: 'Each option must be different.', // W27
  nestedSection: 'Sections cannot be placed inside other sections.', // W27
  noText: 'Add text to this item.', // W28
  conditionOrder: 'This condition depends on a question that is no longer above this item.', // W28
  conditionAnswer: "This condition's answer is no longer one of the options.", // W28
  draftExists: 'A draft of this questionnaire already exists.', // W29
  openDraft: 'Open draft', // W29
  changed: 'Someone else changed this questionnaire. Reload to see their changes. Your changes since the last save will be lost.', // W29
  reload: 'Reload', // W29
  notDraft: 'This version is published and can no longer be changed.', // W29
  needsAttention: 'Some items need attention.', // W29
  notAvailable: 'This questionnaire is not available.', // W29
  dragInstructions: 'To move this item, press space. Use the arrow keys to move it, space to drop it, or escape to cancel.', // W41
  moving: (item: string) => `Moving ${item}.`, // W42
  at: (item: string, n: number, total: number) => `${item} is at position ${n} of ${total}.`, // W42
  inSection: (item: string, section: string, n: number, total: number) => `${item} is in ${section}, at position ${n} of ${total}.`, // W42
  moved: (item: string, n: number, total: number) => `${item} moved to position ${n} of ${total}.`, // W42
  cancelled: (item: string, n: number, total: number) => `Move cancelled. ${item} is back at position ${n} of ${total}.`, // W42
} as const
