<script setup lang="ts">
import { onUnmounted, ref, watch } from "vue";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Loader2, RefreshCw } from "lucide-vue-next";
import QRCode from "qrcode.vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue-sonner";

const { t } = useI18n();

const props = defineProps<{
  open: boolean;
  qrCode: string;
  timeout: number;
  errorMessage?: string;
  refreshing?: boolean;
  pairingCode?: string;
  pairingPhoneNumber?: string;
  requestingPairCode?: boolean;
}>();

const emit = defineEmits<{
  (e: "update:open", value: boolean): void;
  (e: "refresh"): void;
  (e: "requestPairCode", phoneNumber: string): void;
}>();

const timeLeft = ref(props.timeout);
const phoneNumber = ref("");
const timer = setInterval(() => {
  if (timeLeft.value > 0) timeLeft.value--;
}, 1000);

watch(
  () => props.timeout,
  (nextTimeout) => {
    timeLeft.value = nextTimeout;
  },
);

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      timeLeft.value = props.timeout;
    }
  },
);

onUnmounted(() => {
  clearInterval(timer);
});

function requestPairCode() {
  const value = phoneNumber.value.trim();
  if (!value) {
    toast.error(t("instances.qr_modal.validation.phoneRequired"));
    return;
  }
  emit("requestPairCode", value);
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent
      class="bg-[#1a1a1c] border-white/[0.1] text-white light:bg-white light:border-gray-200 shadow-2xl w-[95vw] sm:max-w-2xl min-h-[620px] rounded-[1rem] flex flex-col"
    >
      <DialogHeader>
        <DialogTitle class="text-xl font-semibold text-center">{{
          $t("instances.qr_modal.title")
        }}</DialogTitle>
        <DialogDescription
          class="text-center text-white/60 light:text-gray-500"
        >
          {{ $t("instances.qr_modal.description") }}
        </DialogDescription>
      </DialogHeader>

      <div class="flex flex-1 flex-col p-8 space-y-8">
        <div class="flex flex-1 items-center justify-center">
          <div v-if="qrCode" class="bg-white p-6 rounded-2xl shadow-lg mx-auto">
            <QRCode :value="qrCode" :size="320" level="M" />
          </div>
          <div
            v-else
            :class="[
              'flex flex-col items-center justify-center h-[360px] w-[360px] rounded-2xl border mx-auto',
              errorMessage
                ? 'bg-red-500/5 border-red-400/30'
                : 'bg-white/5 border-white/10 animate-pulse',
            ]"
          >
            <Loader2 class="h-8 w-8 text-emerald-500 animate-spin mb-2" />
            <span
              :class="[
                'text-sm text-center px-4',
                errorMessage ? 'text-red-300' : 'text-white/40',
              ]"
            >
              {{ errorMessage || $t("instances.qr_modal.waitingCode") }}
            </span>
          </div>
        </div>
        <div class="w-full space-y-2">
          <div class="flex justify-between text-xs text-white/40 font-mono">
            <span>{{ $t("instances.qr_modal.refreshingIn") }}</span>
            <span>{{ timeLeft }}s</span>
          </div>
          <Progress
            :value="(timeLeft / timeout) * 100"
            class="h-1 bg-white/10"
          />
        </div>

        <Button
          variant="outline"
          class="w-full border-white/10 hover:bg-white/5 text-white/80"
          :disabled="!!refreshing"
          @click="emit('refresh')"
        >
          <Loader2 v-if="refreshing" class="h-4 w-4 mr-2 animate-spin" />
          <RefreshCw v-else class="h-4 w-4 mr-2" />
          {{ $t("instances.qr_modal.regenerateQR") }}
        </Button>

        <div class="rounded-lg border border-white/[0.12] p-4 space-y-3">
          <p class="text-sm text-white/75 light:text-gray-700">
            {{ $t("instances.qr_modal.pairPhone") }}
          </p>
          <div class="flex gap-2">
            <Input
              v-model="phoneNumber"
              :placeholder="$t('instances.qr_modal.phonePlaceholder')"
              class="bg-white/[0.04] border-white/[0.12] text-white placeholder:text-white/25 light:bg-white light:border-gray-300 light:text-gray-900 light:placeholder:text-gray-400"
            />
            <Button
              variant="outline"
              class="border-white/10 hover:bg-white/5 text-white/80 whitespace-nowrap"
              :disabled="!!requestingPairCode || !phoneNumber.trim()"
              @click="requestPairCode"
            >
              <Loader2
                v-if="requestingPairCode"
                class="h-4 w-4 mr-2 animate-spin"
              />
              {{ $t("instances.qr_modal.getCode") }}
            </Button>
          </div>

          <div
            v-if="pairingCode"
            class="rounded-md bg-white/[0.04] border border-white/[0.1] p-3 text-center"
          >
            <p class="text-xs text-white/60 mb-1">
              {{ $t("instances.qr_modal.enterCode") }}
            </p>
            <p class="text-2xl font-mono tracking-widest text-emerald-400">
              {{ pairingCode }}
            </p>
            <p v-if="pairingPhoneNumber" class="text-xs text-white/45 mt-1">
              {{ $t("instances.qr_modal.phone") }}: {{ pairingPhoneNumber }}
            </p>
          </div>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
