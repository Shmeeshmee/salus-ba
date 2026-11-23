import { appHref } from '../lib/appHref'
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
  // The clinic's self-signup (docs/clinic-cms, O1): the web app's /signup. [APP URL] is a placeholder until the web
  // app's address is set at build time (VITE_APP_URL).
  cta: { label: 'Sign up', href: appHref('/signup', import.meta.env.VITE_APP_URL) },
} as const
