// Fixed messages shared by the screens. API error codes are mapped to plain sentences; nothing from a
// response body is shown.
export const somethingWrong = 'Something went wrong. Try again.'

// What is left of the consultation and questionnaire texts is only what area A's screens still import
// (Calendar, AppointmentDetail, App, Availability: package 1.5's files). Area B's screens read their text from the
// catalogs (web/src/i18n/consultation/, web/src/i18n/questionnaire/, web/src/i18n/attachments/), which hold these
// texts too; delete these blocks once nothing imports them. The W numbers are COPY.md's.
export const consultation = {
  join: 'Join consultation', // W1
  joinFrom: 'You can join from 10 minutes before the start.', // W2
  title: 'Consultation', // W35
  states: { not_started: 'Not started', open: 'Open', ended: 'Ended' }, // W36
  patientJoinedAt: (time: string) => `Patient joined at ${time}`, // W37
  patientDidNotJoin: 'Patient did not join', // W37
  practitionerJoinedAt: (time: string) => `Practitioner joined at ${time}`, // W38
  practitionerDidNotJoin: 'Practitioner did not join', // W38
} as const

export const questionnaires = {
  title: 'Questionnaires', // W1
  defaultLabel: 'Questionnaire for every booking', // W30
  none: 'None', // W30
  defaultHint: 'Patients who book from now on are asked to fill it in before their appointment.', // W30
  defaultNotActive: 'This questionnaire is not active. Choose an active one.', // W31
} as const
