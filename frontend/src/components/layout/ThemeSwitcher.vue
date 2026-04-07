<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { Sun, Moon, Monitor } from "lucide-vue-next";
import { useColorMode } from "@/composables/useColorMode";
import { usersService } from "@/services/api";
import { unwrapResponse } from "@/lib/api-utils";
import { useAuthStore } from "@/stores/auth";
import type { ThemeMode, UserSettings } from "@/types/auth";

const authStore = useAuthStore();
const {
  colorMode,
  themePreset,
  previewAppearance,
  hydrateFromUserSettings,
  restorePersistedAppearance,
} = useColorMode();

async function handleModeChange(mode: ThemeMode) {
  previewAppearance(mode, themePreset.value);

  if (!authStore.user) {
    hydrateFromUserSettings({
      theme_mode: mode,
      theme_preset: themePreset.value,
    });
    return;
  }

  try {
    const response = await usersService.updateSettings({
      theme_mode: mode,
    });
    const payload = unwrapResponse<{
      message: string;
      settings: UserSettings;
    }>(response);
    authStore.replaceUserSettings(payload.settings);
    hydrateFromUserSettings(payload.settings);
  } catch {
    restorePersistedAppearance();
  }
}
</script>

<template>
  <div
    class="flex gap-1 rounded-full border border-border bg-muted/60 p-1"
    role="radiogroup"
    aria-label="Color theme"
  >
    <Button
      variant="ghost"
      size="icon"
      class="h-7 w-7"
      :class="
        colorMode === 'light' && 'bg-background text-foreground shadow-sm'
      "
      :aria-checked="colorMode === 'light'"
      aria-label="Light theme"
      role="radio"
      @click="handleModeChange('light')"
    >
      <Sun class="h-3.5 w-3.5" aria-hidden="true" />
    </Button>
    <Button
      variant="ghost"
      size="icon"
      class="h-7 w-7"
      :class="colorMode === 'dark' && 'bg-background text-foreground shadow-sm'"
      :aria-checked="colorMode === 'dark'"
      aria-label="Dark theme"
      role="radio"
      @click="handleModeChange('dark')"
    >
      <Moon class="h-3.5 w-3.5" aria-hidden="true" />
    </Button>
    <Button
      variant="ghost"
      size="icon"
      class="h-7 w-7"
      :class="
        colorMode === 'system' && 'bg-background text-foreground shadow-sm'
      "
      :aria-checked="colorMode === 'system'"
      aria-label="System theme"
      role="radio"
      @click="handleModeChange('system')"
    >
      <Monitor class="h-3.5 w-3.5" aria-hidden="true" />
    </Button>
  </div>
</template>
