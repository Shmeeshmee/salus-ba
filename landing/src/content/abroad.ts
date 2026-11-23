import { ids } from './sections'

export type AbroadPoint = {
  text: string
  coming: boolean
}

export const abroad = {
  id: ids.abroad,
  eyebrow: 'Patients abroad',
  heading: 'A doctor they know, wherever they live.',
  body: 'Patients from the diaspora and abroad find your clinic in the Salus app and see your doctors by video, without travelling home for a follow-up.',
  comingLabel: 'Coming',
  points: [
    {
      text: 'The patient app is in ten languages, among them German, Turkish, Italian, French and Arabic.',
      coming: false,
    },
    { text: 'Live translation during a video visit, as an add-on billed per use.', coming: false },
    { text: 'Booking and paying in advance from abroad.', coming: true },
  ] satisfies AbroadPoint[],
}
