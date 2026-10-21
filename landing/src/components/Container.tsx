import type { ComponentPropsWithoutRef } from 'react'
import { cx } from '../lib/cx'
import styles from './Container.module.css'

type ContainerProps = ComponentPropsWithoutRef<'section'>

// A page section at the content width. Sections set their own vertical padding.
export function Container({ className, ...section }: ContainerProps) {
  return <section className={cx(styles.container, className)} {...section} />
}
