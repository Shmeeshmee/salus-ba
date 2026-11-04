import qrcode from 'qrcode-generator'
import { useMemo } from 'react'

const quiet = 4 // modules of white around the code, as the QR standard asks

// QRCode draws value as an SVG path. The value itself is not written into the DOM.
export function QRCode({ value, label }: { value: string; label: string }) {
  const { size, path } = useMemo(() => {
    const qr = qrcode(0, 'M')
    qr.addData(value)
    qr.make()
    const n = qr.getModuleCount()
    let d = ''
    for (let row = 0; row < n; row++) {
      for (let col = 0; col < n; col++) if (qr.isDark(row, col)) d += `M${col} ${row}h1v1h-1z`
    }
    return { size: n, path: d }
  }, [value])

  const box = size + 2 * quiet
  return (
    <svg
      className="qr"
      role="img"
      aria-label={label}
      viewBox={`${-quiet} ${-quiet} ${box} ${box}`}
      shapeRendering="crispEdges"
    >
      <rect x={-quiet} y={-quiet} width={box} height={box} fill="#FFFFFF" />
      <path d={path} fill="#0E1512" />
    </svg>
  )
}
