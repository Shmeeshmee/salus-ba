import { ids } from './sections'

export const howItWorks = {
  id: ids.how,
  eyebrow: 'How it works',
  heading: 'From booking to receipt.',
  steps: [
    {
      title: 'Patient books',
      body: 'In the mobile app or at the desk: a doctor, a free time, at the clinic or by video.',
    },
    { title: 'Questionnaire first', body: "Your pre-visit form lands in the doctor's view before the visit starts." },
    {
      title: 'Visit and receipt',
      body: 'The doctor sees the patient in the room or by video and signs the note; the desk issues the receipt.',
    },
  ],
} as const
