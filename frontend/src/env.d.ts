/// <reference types="vite/client" />

declare module "vue3-emoji-picker/css";

interface ImportMetaEnv {
  readonly VITE_API_URL: string;
  readonly VITE_WS_URL: string;
  readonly VITE_POSTHOG_KEY: string;
  readonly VITE_POSTHOG_HOST: string;
  readonly VITE_PUBLIC_MARKETING_BASE_URL: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
