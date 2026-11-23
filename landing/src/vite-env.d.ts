/// <reference types="vite/client" />

interface ImportMetaEnv {
  // The web app's address, e.g. https://app.example.test, for the sign-up link; read at build time.
  readonly VITE_APP_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
