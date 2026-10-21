const MARKER = '<!--app-->'

// Puts the server-rendered app into the HTML template, in place of the marker in index.html.
export function injectApp(template: string, appHtml: string): string {
  if (!template.includes(MARKER)) {
    throw new Error(`the HTML template has no ${MARKER} marker`)
  }
  return template.replace(MARKER, () => appHtml)
}
