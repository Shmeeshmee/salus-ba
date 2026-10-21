import { cx } from '../../../lib/cx'
import styles from './Ribbon.module.css'

type RibbonProps = {
  layer: 'back' | 'front'
  swap: 'a' | 'b'
}

const PATH =
  'M0 210 H560 L600 210 L630 120 L668 280 L705 50 L742 240 L770 210 H930 L960 210 L985 160 L1015 255 L1040 210 H1600'

// A heartbeat line that runs behind the phone and, through the front layer, across its left edge.
export function Ribbon({ layer, swap }: RibbonProps) {
  return (
    <svg
      className={cx(styles.ribbon, layer === 'front' && styles.front)}
      viewBox="0 0 1600 300"
      preserveAspectRatio="none"
      aria-hidden="true"
    >
      <path className={cx(styles.line, swap === 'a' ? styles.drawA : styles.drawB)} d={PATH} />
    </svg>
  )
}
