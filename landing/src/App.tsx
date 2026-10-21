import { SkipLink } from './components/SkipLink'
import { nav } from './content/nav'
import { ids } from './content/sections'
import { Demo } from './sections/Demo/Demo'
import { Footer } from './sections/Footer/Footer'
import { Hero } from './sections/Hero/Hero'
import { Highlights } from './sections/Highlights/Highlights'
import { HowItWorks } from './sections/HowItWorks/HowItWorks'
import { Pricing } from './sections/Pricing/Pricing'
import { Product } from './sections/Product/Product'
import { Standards } from './sections/Standards/Standards'

export function App() {
  return (
    <>
      <SkipLink targetId={ids.page}>{nav.skipLink}</SkipLink>
      <Hero />
      <main id={ids.page}>
        <Highlights />
        <Product />
        <HowItWorks />
        <Standards />
        <Pricing />
        <Demo />
      </main>
      <Footer />
    </>
  )
}
