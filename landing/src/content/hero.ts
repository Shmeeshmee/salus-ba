import { ids } from './sections'

export const hero = {
  label: 'Introduction',
  eyebrow: 'Telemedicine for private clinics',
  headline: [['Your', 'doctor,'], ['one', 'call'], ['away.']],
  primaryCta: { label: 'Book a demo', href: `#${ids.demo}` },
  secondaryCta: 'See the product',
  note: 'Illustrative screens with synthetic data.',
  scrollCue: { label: 'Scroll', ariaLabel: 'Scroll to content' },
  phone: {
    label: 'The Salus patient app during a video consult',
    brand: 'Salus',
    live: 'Live · 12:48',
    self: 'You',
    transcription: 'Live transcription',
  },
  chips: {
    booking: { title: 'Booking confirmed', detail: 'Thu 10:30' },
    form: { title: 'Pre-visit form', progress: '4/6', percent: 66 },
  },
  monitor: {
    readouts: [
      { label: 'HR', value: '72', unit: 'bpm', beat: true },
      { label: 'SpO₂', value: '98', unit: ' %', beat: false },
      { label: 'NIBP', value: '118/76', unit: '', beat: false },
    ],
    lead: 'II · 25 mm/s · 10 mm/mV',
  },
} as const
