import { ids } from './sections'

export const pricing = {
  id: ids.pricing,
  eyebrow: 'Pricing',
  heading: 'A monthly membership per practitioner.',
  // [PRICE] is a placeholder until the plan prices are settled.
  plans: [
    {
      label: 'Basic',
      amount: '[PRICE]',
      unit: ' / practitioner / month',
      body: 'The clinic system: calendar, in-person visits, price list, receipts, questionnaires, documents, reminders, waitlist, chat and analytics.',
    },
    {
      label: 'Pro',
      amount: '[PRICE]',
      unit: ' / practitioner / month',
      body: 'Everything in Basic, plus video visits and online hours, live transcription and AI that drafts the visit note.',
    },
  ],
  notes: [
    'Every clinic starts with a 30-day trial. You pay each month only for the practitioners who are active.',
    'Patients pay at the clinic, as they do today. Live translation is an add-on, billed per use.',
  ],
} as const
