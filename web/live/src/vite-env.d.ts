/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_ROOT?: string
  readonly VITE_SOCKET_URL?: string
  readonly VITE_LIVEKIT_URL?: string
  readonly VITE_JWT_SECRET?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
