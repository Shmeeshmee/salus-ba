// Last build step: renders the app with the server build (dist-ssr) and writes it into dist/index.html, so
// the served page carries its full content before any JavaScript runs.
import { readFile, writeFile } from 'node:fs/promises'
import { injectApp } from '../src/lib/injectApp.ts'

type ServerEntry = { render: () => string }

const page = new URL('../dist/index.html', import.meta.url)
const entry = new URL('../dist-ssr/entry-server.js', import.meta.url)

const { render } = (await import(entry.href)) as ServerEntry
await writeFile(page, injectApp(await readFile(page, 'utf8'), render()))
