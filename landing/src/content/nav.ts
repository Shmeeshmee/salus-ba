import { ids } from './sections'

export const nav = {
  skipLink: 'Skip to content',
  brand: 'Salus',
  homeLabel: 'Salus home',
  label: 'Main',
  links: [
    { label: 'Product', href: `#${ids.product}` },
    { label: 'How it works', href: `#${ids.how}` },
    { label: 'Standards', href: `#${ids.standards}` },
    { label: 'Pricing', href: `#${ids.pricing}` },
  ],
  cta: { label: 'Book a demo', href: `#${ids.demo}` },
} as const
