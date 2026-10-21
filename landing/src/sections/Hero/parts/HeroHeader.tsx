import { Button } from '../../../components/Button'
import { nav } from '../../../content/nav'
import { ids } from '../../../content/sections'
import styles from './HeroHeader.module.css'

export function HeroHeader() {
  return (
    <header className={styles.header} style={{ '--delay': '1.5s' }}>
      <a className={styles.home} href={`#${ids.top}`} aria-label={nav.homeLabel}>
        <svg className={styles.logo} width="30" height="30" viewBox="0 0 30 30" aria-hidden="true">
          <circle className={styles.logoDisc} cx="15" cy="15" r="14" />
          <rect className={styles.logoUpright} x="12.5" y="7" width="5" height="16" rx="2.5" />
          <rect className={styles.logoBar} x="7" y="12.5" width="16" height="5" rx="2.5" />
        </svg>
        <span className={styles.wordmark}>{nav.brand}</span>
      </a>
      <nav className={styles.nav} aria-label={nav.label}>
        {nav.links.map((link) => (
          <a key={link.href} className={styles.link} href={link.href}>
            {link.label}
          </a>
        ))}
        <Button variant="ink" size="sm" href={nav.cta.href}>
          {nav.cta.label}
        </Button>
      </nav>
    </header>
  )
}
