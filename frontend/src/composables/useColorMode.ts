import { ref, watch, onMounted } from "vue";

export type ColorMode = "light" | "dark" | "system";

const colorMode = ref<ColorMode>("system");
const isDark = ref(false);
let mediaListenerAttached = false;

function getSystemTheme(): boolean {
  return window.matchMedia("(prefers-color-scheme: dark)").matches;
}

function applyTheme(dark: boolean) {
  document.documentElement.classList.toggle("dark", dark);
  document.documentElement.classList.toggle("light", !dark);
  document.documentElement.style.colorScheme = dark ? "dark" : "light";
}

function updateTheme() {
  if (colorMode.value === "system") {
    isDark.value = getSystemTheme();
  } else {
    isDark.value = colorMode.value === "dark";
  }

  applyTheme(isDark.value);
}

function ensureSystemListener() {
  if (mediaListenerAttached) {
    return;
  }

  const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
  mediaQuery.addEventListener("change", () => {
    if (colorMode.value === "system") {
      updateTheme();
    }
  });
  mediaListenerAttached = true;
}

export function useColorMode() {
  onMounted(() => {
    const saved = localStorage.getItem("color-mode") as ColorMode | null;
    colorMode.value = saved ?? "system";
    ensureSystemListener();
    updateTheme();
  });

  watch(colorMode, (newMode) => {
    localStorage.setItem("color-mode", newMode);
    updateTheme();
  });

  function setColorMode(mode: ColorMode) {
    colorMode.value = mode;
  }

  return {
    colorMode,
    isDark,
    setColorMode,
  };
}
