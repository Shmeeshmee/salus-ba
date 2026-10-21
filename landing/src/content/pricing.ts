import { ids } from './sections'

export const pricing = {
  id: ids.pricing,
  eyebrow: 'Pricing',
  heading: 'One plan per clinic. You grow, we grow.',
  // [PRICE] and [X] are placeholders until pricing is settled.
  plans: [
    {
      label: 'Yearly subscription',
      amount: '[PRICE]',
      unit: ' / year',
      body: 'Web app for your whole team, patient app listing, video and transcription.',
    },
    {
      label: 'Per paid service',
      amount: '[X]%',
      unit: '',
      body: 'A small share of each paid service booked through Salus. Nothing when nothing is booked.',
    },
  ],
} as const
