import { useEffect, useRef } from 'react'
import { hero } from '../../content/hero'
import { ids } from '../../content/sections'
import { specialties } from '../../content/specialties'
import { useCycle } from '../../hooks/useCycle'
import { useScrollJump } from '../../hooks/useScrollJump'
import { EcgMonitor } from './parts/EcgMonitor'
import { FloatingChips } from './parts/FloatingChips'
import { Headline } from './parts/Headline'
import { HeroHeader } from './parts/HeroHeader'
import { PhoneMockup } from './parts/PhoneMockup'
import { Ribbon } from './parts/Ribbon'
import { ScrollCue } from './parts/ScrollCue'
import { SpecialtyPicker } from './parts/SpecialtyPicker'
import styles from './Hero.module.css'

const CYCLE_MS = 4000
// The cycle waits for the intro zoom and the headline to land.
const INTRO_MS = 2200

export function Hero() {
  const heroRef = useRef<HTMLElement>(null)
  const { index, step, select } = useCycle(specialties.length, { interval: CYCLE_MS, initialDelay: INTRO_MS })
  const jumpToPage = useScrollJump(heroRef, ids.page)
  const { bg, ink, accent } = specialties[index].palette
  // Alternates on every change so the phone swap and the ribbon draw restart, even for a repeated pick.
  const swap = step % 2 === 1 ? 'a' : 'b'

  useEffect(() => {
    document.querySelector('meta[name="theme-color"]')?.setAttribute('content', bg)
  }, [bg])

  return (
    <section
      ref={heroRef}
      id={ids.top}
      className={styles.hero}
      aria-label={hero.label}
      style={{ '--bg': bg, '--ink': ink, '--acc': accent }}
    >
      <div className={styles.exit}>
        <div className={styles.scene}>
          <EcgMonitor />
          <HeroHeader />
          <div className={styles.row}>
            <Headline onSeeProduct={jumpToPage} />
            <div className={styles.stage}>
              <Ribbon layer="back" swap={swap} />
              <PhoneMockup specialties={specialties} active={index} swap={swap} />
              <Ribbon layer="front" swap={swap} />
              <FloatingChips />
            </div>
            <SpecialtyPicker specialties={specialties} active={index} onSelect={select} />
          </div>
        </div>
      </div>
      <ScrollCue onClick={jumpToPage} />
    </section>
  )
}
