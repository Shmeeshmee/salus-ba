import { useSyncExternalStore } from 'react'
import { footer } from '../../content/footer'
import styles from './Footer.module.css'

const neverChanges = () => () => {}
const currentYear = () => new Date().getFullYear()
const buildYear = () => footer.year

export function Footer() {
  // The prerendered page carries the build year; the browser switches to the current one after hydration.
  const year = useSyncExternalStore(neverChanges, currentYear, buildYear)

  return (
    <footer className={styles.footer}>
      <span>
        © {year} {footer.brand}
      </span>
      <span className={styles.links}>
        {footer.links.map((link) => (
          <a key={link.href} className={styles.link} href={link.href}>
            {link.label}
          </a>
        ))}
      </span>
    </footer>
  )
}
