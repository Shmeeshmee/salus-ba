import { ids } from './sections'

export const howItWorks = {
  id: ids.how,
  eyebrow: 'How it works',
  heading: 'From booking to signed note.',
  steps: [
    { title: 'Patient books', body: 'They pick your clinic, a doctor and a free slot in the mobile app.' },
    { title: 'Questionnaire first', body: "Your custom pre-visit form lands in the doctor's view before the call starts." },
    { title: 'Consult and sign', body: 'Video, live transcript, a drafted note — the doctor reviews and signs.' },
  ],
} as const
