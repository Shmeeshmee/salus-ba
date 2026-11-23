// VisitReceipt is a receipt of a visit as the desk and the visit page show it: its number, whether it was reversed, and
// its id for the PDF.
export type VisitReceipt = { id: string; number: number; reversed: boolean }

// mergeReceipts is the receipts read with those issued since, once each, by number.
export function mergeReceipts(read: VisitReceipt[], issued: VisitReceipt[]): VisitReceipt[] {
  const all = new Map<string, VisitReceipt>()
  for (const r of [...read, ...issued]) if (!all.has(r.id)) all.set(r.id, r)
  return [...all.values()].sort((a, b) => a.number - b.number)
}
