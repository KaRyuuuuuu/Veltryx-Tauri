/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_BRIDGE_SHARED_KEY_B64?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
