<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, computed } from "vue";
import { RouterLink } from "vue-router";
import { useI18n } from "vue-i18n";
import { useAuthStore } from "@/stores/auth";
import { useContactsStore } from "@/stores/contacts";
import { usersService, chatbotService } from "@/services/api";
import { localeDirectionManager } from "@/i18n/locale-direction";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { LogOut, User } from "lucide-vue-next";
import { toast } from "vue-sonner";
import { getInitials } from "@/lib/utils";
import ThemeSwitcher from "./ThemeSwitcher.vue";
import LanguageSwitcher from "@/components/LanguageSwitcher.vue";

const { t, locale } = useI18n();

const props = withDefaults(
  defineProps<{
    expanded?: boolean;
    collapsed?: boolean;
  }>(),
  {
    expanded: undefined,
    collapsed: false,
  },
);

const emit = defineEmits<{
  logout: [];
}>();

const authStore = useAuthStore();
const contactsStore = useContactsStore();
const isUserMenuOpen = ref(false);
const isUpdatingAvailability = ref(false);
const isCheckingTransfers = ref(false);
const showAwayWarning = ref(false);
const awayWarningTransferCount = ref(0);
const isRTL = computed(() => localeDirectionManager.isRTL(locale.value));
const isExpanded = computed(() => props.expanded ?? !props.collapsed);

const handleAvailabilityChange = async (checked: boolean) => {
  if (!checked) {
    isCheckingTransfers.value = true;
    try {
      const response = await chatbotService.listTransfers({ status: "active" });
      const data = response.data.data || response.data;
      const transfers = data.transfers || [];
      const userId = authStore.user?.id;
      const myActiveTransfers = transfers.filter(
        (t: any) => t.agent_id === userId,
      );

      if (myActiveTransfers.length > 0) {
        awayWarningTransferCount.value = myActiveTransfers.length;
        showAwayWarning.value = true;
        return;
      }
    } catch (error) {
      console.error("Failed to check transfers:", error);
    } finally {
      isCheckingTransfers.value = false;
    }
  }

  await setAvailability(checked);
};

const confirmGoAway = async () => {
  showAwayWarning.value = false;
  await setAvailability(false);
};

const setAvailability = async (checked: boolean) => {
  isUpdatingAvailability.value = true;
  try {
    const response = await usersService.updateAvailability(checked);
    const data = response.data.data;
    authStore.setAvailability(checked, data.break_started_at);

    if (checked) {
      toast.success(t("userMenu.available"), {
        description: t("userMenu.availableDesc"),
      });
    } else {
      const transfersReturned = data.transfers_to_queue || 0;
      toast.success(t("userMenu.away"), {
        description:
          transfersReturned > 0
            ? t("userMenu.transfersReturned", { count: transfersReturned })
            : t("userMenu.awayDesc"),
      });

      if (transfersReturned > 0) {
        contactsStore.fetchContacts();
      }
    }
  } catch (error) {
    toast.error(t("common.error"), {
      description: t("userMenu.failedUpdateAvailability"),
    });
  } finally {
    isUpdatingAvailability.value = false;
  }
};

// Break duration tracking
const breakDuration = ref("");
let breakTimerInterval: ReturnType<typeof setInterval> | null = null;

const updateBreakDuration = () => {
  if (!authStore.breakStartedAt) {
    breakDuration.value = "";
    return;
  }
  const start = new Date(authStore.breakStartedAt);
  const now = new Date();
  const diffMs = now.getTime() - start.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const hours = Math.floor(diffMins / 60);
  const mins = diffMins % 60;

  if (hours > 0) {
    breakDuration.value = `${hours}h ${mins}m`;
  } else {
    breakDuration.value = `${mins}m`;
  }
};

watch(
  () => authStore.isAvailable,
  (available) => {
    if (!available && authStore.breakStartedAt) {
      updateBreakDuration();
      breakTimerInterval = setInterval(updateBreakDuration, 60000);
    } else if (breakTimerInterval) {
      clearInterval(breakTimerInterval);
      breakTimerInterval = null;
      breakDuration.value = "";
    }
  },
  { immediate: true },
);

onMounted(() => {
  authStore.restoreBreakTime();
  if (!authStore.isAvailable && authStore.breakStartedAt) {
    updateBreakDuration();
    breakTimerInterval = setInterval(updateBreakDuration, 60000);
  }
});

onUnmounted(() => {
  if (breakTimerInterval) {
    clearInterval(breakTimerInterval);
  }
});

const handleLogout = () => {
  emit("logout");
};
</script>

<template>
  <div class="border-t border-sidebar-border p-2">
    <Popover v-model:open="isUserMenuOpen">
      <PopoverTrigger as-child>
        <Button
          variant="ghost"
          data-testid="sidebar-user-menu-trigger"
          :title="authStore.user?.email || $t('userMenu.myAccount')"
          :class="[
            'flex h-auto w-full items-center gap-2 px-2 py-1.5 hover:bg-sidebar-accent/70',
            !isExpanded
              ? 'md:justify-center'
              : isRTL
                ? 'justify-end'
                : 'justify-start',
          ]"
          aria-label="User menu"
        >
          <Avatar class="h-7 w-7 ring-2 ring-sidebar-border">
            <AvatarImage :src="undefined" />
            <AvatarFallback class="bg-primary text-xs text-primary-foreground">
              {{ getInitials(authStore.user?.full_name || "U") }}
            </AvatarFallback>
          </Avatar>
          <div
            :class="[
              'flex flex-col overflow-hidden transition-[max-width,opacity] duration-200',
              isExpanded ? 'max-w-[140px] opacity-100' : 'max-w-0 opacity-0',
              isRTL && isExpanded ? 'items-end text-right' : 'items-start text-left',
            ]"
          >
            <span
              class="max-w-[140px] truncate text-[13px] font-medium text-sidebar-foreground"
            >
              {{ authStore.user?.full_name }}
            </span>
            <span
              class="max-w-[140px] truncate text-[11px] text-sidebar-foreground/55"
            >
              {{ authStore.user?.email }}
            </span>
          </div>
        </Button>
      </PopoverTrigger>
      <PopoverContent side="top" align="start" class="w-56 p-1.5">
        <div class="px-2 py-1 text-xs font-medium text-muted-foreground">
          {{ $t("userMenu.myAccount") }}
        </div>
        <Separator class="my-1" />
        <!-- Availability Toggle -->
        <div class="flex items-center justify-between px-2 py-1.5">
          <div class="flex items-center gap-2">
            <span class="text-[13px] text-foreground/80">{{
              $t("userMenu.status")
            }}</span>
            <Badge
              :class="
                'text-[10px] px-1.5 py-0 ' +
                (authStore.isAvailable
                  ? 'bg-emerald-500/20 text-emerald-400 light:bg-emerald-100 light:text-emerald-700'
                  : 'bg-muted text-muted-foreground')
              "
            >
              {{
                authStore.isAvailable
                  ? $t("userMenu.available")
                  : $t("userMenu.away")
              }}
            </Badge>
            <span
              v-if="!authStore.isAvailable && breakDuration"
              class="text-[10px] text-muted-foreground/80"
            >
              {{ breakDuration }}
            </span>
          </div>
          <Switch
            :checked="authStore.isAvailable"
            :disabled="isUpdatingAvailability || isCheckingTransfers"
            aria-label="Toggle availability status"
            @update:checked="handleAvailabilityChange"
          />
        </div>
        <Separator class="my-1" />
        <RouterLink to="/profile">
          <Button
            variant="ghost"
            class="h-auto w-full justify-start px-2 py-1 text-[13px] font-normal"
            @click="isUserMenuOpen = false"
          >
            <User class="mr-2 h-3.5 w-3.5" aria-hidden="true" />
            <span>{{ $t("userMenu.profile") }}</span>
          </Button>
        </RouterLink>
        <Separator class="my-1" />
        <div class="px-2 py-1 text-xs font-medium text-muted-foreground">
          {{ $t("userMenu.theme") }}
        </div>
        <ThemeSwitcher />
        <Separator class="my-1" />
        <div class="px-2 py-1 text-xs font-medium text-muted-foreground">
          {{ $t("userMenu.language") }}
        </div>
        <div class="px-1.5 py-1">
          <LanguageSwitcher />
        </div>
        <Separator class="my-1" />
        <Button
          variant="ghost"
          class="h-auto w-full justify-start px-2 py-1 text-[13px] font-normal"
          @click="handleLogout"
        >
          <LogOut class="mr-2 h-3.5 w-3.5" aria-hidden="true" />
          <span>{{ $t("userMenu.logOut") }}</span>
        </Button>
      </PopoverContent>
    </Popover>
  </div>

  <!-- Away Warning Dialog -->
  <AlertDialog :open="showAwayWarning">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{
          $t("userMenu.awayWarningTitle")
        }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{
            $t("userMenu.awayWarningDesc", { count: awayWarningTransferCount })
          }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <Button variant="outline" @click="showAwayWarning = false">{{
          $t("common.cancel")
        }}</Button>
        <Button @click="confirmGoAway" :disabled="isUpdatingAvailability">{{
          $t("userMenu.goAway")
        }}</Button>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>
