import 'react'

// Lets style props set CSS custom properties, such as an animation's --delay.
declare module 'react' {
  interface CSSProperties {
    [property: `--${string}`]: string | number | undefined
  }
}
