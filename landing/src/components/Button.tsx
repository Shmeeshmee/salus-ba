import type { ComponentPropsWithoutRef, ReactNode } from 'react'
import { cx } from '../lib/cx'
import styles from './Button.module.css'

type Look = {
  variant: 'ink' | 'line' | 'mint'
  size?: 'md' | 'sm'
  children: ReactNode
}

type LinkProps = Look & { href: string } & Omit<ComponentPropsWithoutRef<'a'>, keyof Look | 'href'>
type ActionProps = Look & { href?: undefined } & Omit<ComponentPropsWithoutRef<'button'>, keyof Look>

export type ButtonProps = LinkProps | ActionProps

// A link when it has an href, otherwise a button. ink and line take their colours from the hero (--ink,
// --bg), mint is the light-on-dark call to action.
export function Button(props: ButtonProps) {
  if (props.href !== undefined) {
    const { variant, size = 'md', className, ...link } = props
    return <a className={cx(styles.button, styles[variant], styles[size], className)} {...link} />
  }
  const { variant, size = 'md', className, type = 'button', ...button } = props
  return <button type={type} className={cx(styles.button, styles[variant], styles[size], className)} {...button} />
}
