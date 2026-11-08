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
  transcriptionOn: 'Transcription is on', // W41, proposed: waiting for the owner (COPY.md)
} as const

// Questionnaires: the library, the builder, the role's default and the appointment's requests (docs/questionnaire/COPY.md, approved by the owner, word for
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
    attachment: 'File upload', // attachments W1 (docs/attachments/COPY.md)
  }, // W11
  empty: 'Drag items here, or select one on the left.', // W12
  question: 'Question', // W13
  instructions: 'Instructions', // W13
  sectionTitle: 'Section title', // W13
  required: 'Required', // W13
  type: 'Type', // W13 (added on review of PR #24, owner)
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
  emptyOption: 'Fill in this option.', // W28 (added on review of PR #24, owner)
  emptySection: 'Add at least one item to this section.', // W28 (added on review of PR #24, owner)
  // A file upload item in the builder (docs/attachments/COPY.md, approved by the owner, word for word; W2 to W6
  // of that file).
  acceptLabel: 'Accepted files', // attachments W2
  accept: { photo: 'Photos', document: 'Documents', video: 'Videos' }, // attachments W2
  acceptHint: 'Photos: JPEG or PNG, up to 20 MB. Documents: PDF, up to 20 MB. Videos: MP4, up to 200 MB.', // attachments W3
  maxFilesLabel: 'Number of files', // attachments W4
  maxFilesHint: 'From 1 to 10.', // attachments W4
  maxFilesSummary: (n: number) => (n === 1 ? '1 file' : `Up to ${n} files`), // attachments W5
  noFileKind: 'Choose at least one kind of file.', // attachments W6
  fileCount: 'The number of files must be from 1 to 10.', // attachments W6
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
  defaultLabel: 'Questionnaire for every booking', // W30
  none: 'None', // W30
  defaultHint: 'Patients who book from now on are asked to fill it in before their appointment.', // W30
  defaultNotActive: 'This questionnaire is not active. Choose an active one.', // W31
  requests: 'Questionnaires', // W32
  noRequests: 'No questionnaires requested.', // W33
  notSubmitted: 'Not submitted yet', // W34
  submitted: (when: string) => `Submitted ${when}`, // W34
  withdrawn: 'Withdrawn', // W34
  notSubmittedBefore: 'Not submitted before the appointment', // W34
  add: 'Add questionnaire', // W35
  picker: 'Questionnaire', // W35
  addConfirm: 'Add', // W35
  addHint: 'The patient sees it in the Salus app and is asked to submit it before the appointment starts.', // W35
  withdraw: 'Withdraw', // W36
  withdrawQuestion: 'Withdraw this questionnaire?', // W36
  withdrawWarning: 'The patient no longer needs to fill it in, and any answers they have saved are deleted.', // W36
  viewAnswers: 'View answers', // W37
  hideAnswers: 'Hide answers', // W37
  notAnswered: 'Not answered', // W38
  answersPrivate: "Only the appointment's practitioner can see the answers.", // W39
  requestErrors: {
    already_requested: 'This questionnaire has already been requested for this appointment.',
    not_active: 'This questionnaire is no longer active.',
    appointment_started: 'The appointment has started, so questionnaires can no longer be added.',
    consent_required: "The patient's consent for this clinic is not active, so a questionnaire cannot be requested.",
    submitted: 'The patient has already submitted this questionnaire.',
  } as Record<string, string | undefined>, // W40
} as const

// Files attached to a submitted questionnaire, on the appointment detail (docs/attachments/COPY.md, approved by the
// owner, word for word; the W numbers are that file's). [Title] is the file's name as the patient uploaded it.
export const files = {
  count: (n: number) => (n === 1 ? '1 file' : `${n} files`), // W7
  kinds: { photo: 'Photo', document: 'Document', video: 'Video' }, // W8
  kb: (n: number) => `${n} KB`, // W8
  mb: (n: string) => `${n} MB`, // W8
  details: (kind: string, size: string) => `${kind}, ${size}`, // W8
  open: 'Open', // W9
  download: 'Download', // W9, W12
  close: 'Close', // W10
  previous: 'Previous file', // W10
  next: 'Next file', // W10
  position: (n: number, total: number) => `File ${n} of ${total}`, // W10
  label: { photo: (title: string) => `Photo: ${title}`, document: (title: string) => `Document: ${title}`, video: (title: string) => `Video: ${title}` }, // W11
  cannotShow: 'This file cannot be shown here.', // W12
  gone: 'This file is no longer available.', // W13
  failed: 'The file could not be loaded.', // W13
  tryAgain: 'Try again', // W13
} as const
