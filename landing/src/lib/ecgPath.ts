const BASELINE = 120
const BEAT_WIDTH = 200
// Slightly uneven amplitudes so the trace does not look machine-repeated.
const AMPLITUDES = [1, 0.94, 1.04, 0.97, 1.02, 0.95, 1, 0.98, 1.03, 0.96, 1.01, 0.97]

function beat(x: number, a: number): string {
  const t = (92 - 6 * a).toFixed(1)
  return (
    ` L${x + 24} ${BASELINE}` +
    ` C${x + 30} ${BASELINE} ${x + 32} 108 ${x + 39} 108` +
    ` C${x + 46} 108 ${x + 48} ${BASELINE} ${x + 54} ${BASELINE}` +
    ` L${x + 66} ${BASELINE} L${x + 70} 127` +
    ` L${x + 78} ${(BASELINE - 100 * a).toFixed(1)}` +
    ` L${x + 86} ${(150 + 6 * a).toFixed(1)}` +
    ` L${x + 91} 117 L${x + 108} 116` +
    ` C${x + 120} 115 ${x + 124} ${t} ${x + 138} ${t}` +
    ` C${x + 152} ${t} ${x + 156} ${BASELINE} ${x + 168} ${BASELINE}` +
    ` L${x + BEAT_WIDTH} ${BASELINE}`
  )
}

// SVG path of an ECG trace: twelve PQRST beats on a 2400 × 240 view box.
export function ecgPath(): string {
  return AMPLITUDES.reduce((d, a, i) => d + beat(i * BEAT_WIDTH, a), `M0 ${BASELINE}`)
}
