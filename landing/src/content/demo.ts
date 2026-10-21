import { ids } from './sections'

export const demo = {
  id: ids.demo,
  heading: 'Bring your clinic online.',
  // [YOUR EMAIL] is a placeholder until the demo address is settled.
  cta: { label: 'Book a demo', href: 'mailto:[YOUR EMAIL]' },
} as const
