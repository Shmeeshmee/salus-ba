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
  // The clinic's self-signup (docs/clinic-cms, O1): the web app's /signup, served beside the landing page.
  cta: { label: 'Sign up', href: '/signup' },
} as const
