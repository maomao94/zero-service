/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_SOCKET_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

interface ImportMetaEnv {
  readonly VITE_SOCKET_URL?: string
  readonly VITE_JWT_SECRET?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
