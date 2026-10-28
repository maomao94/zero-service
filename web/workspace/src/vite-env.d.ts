/// <reference types="vite/client" />

/** 构建期注入：package.json 版本号 */
declare const __APP_VERSION__: string
/** 构建期注入：构建时间（ISO 字符串） */
declare const __BUILD_TIME__: string

interface ImportMetaEnv {
  readonly VITE_LIVE_URL?: string
  readonly VITE_SOCKETIO_URL?: string
  readonly VITE_OPENAPI_URL?: string
  readonly VITE_GITLAB_URL?: string
  readonly VITE_GITHUB_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
