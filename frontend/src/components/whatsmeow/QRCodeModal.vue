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
import { Label } from "@/components/ui/label";
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
      class="flex min-h-[620px] w-[95vw] max-w-none flex-col rounded-[calc(var(--radius)+0.7rem)] shadow-2xl sm:max-w-2xl"
    >
      <DialogHeader>
        <DialogTitle class="text-xl font-semibold text-center">{{
          $t("instances.qr_modal.title")
        }}</DialogTitle>
        <DialogDescription class="text-center text-muted-foreground">
          {{ $t("instances.qr_modal.description") }}
        </DialogDescription>
      </DialogHeader>

      <div class="flex flex-1 flex-col space-y-8 p-6 sm:p-8">
        <div class="flex flex-1 items-center justify-center">
          <div v-if="qrCode" class="bg-white p-6 rounded-2xl shadow-lg mx-auto flex items-center justify-center">
            <img
              v-if="qrCode.startsWith('http://') || qrCode.startsWith('https://') || qrCode.startsWith('data:')"
              :src="qrCode"
              class="w-[320px] h-[320px]"
            />
            <QRCode v-else :value="qrCode" :size="320" level="M" />
          </div>
          <div
            v-else
            :class="[
              'flex flex-col items-center justify-center h-[360px] w-[360px] rounded-2xl border mx-auto',
              errorMessage
                ? 'border-destructive/30 bg-destructive/5'
                : 'border-border bg-muted/30 animate-pulse',
            ]"
          >
            <Loader2 class="h-8 w-8 text-primary animate-spin mb-2" />
            <span
              :class="[
                'text-sm text-center px-4',
                errorMessage ? 'text-destructive' : 'text-muted-foreground',
              ]"
            >
              {{ errorMessage || $t("instances.qr_modal.waitingCode") }}
            </span>
          </div>
        </div>
        <div class="w-full space-y-2">
          <div
            class="flex justify-between font-mono text-xs text-muted-foreground"
          >
            <span>{{ $t("instances.qr_modal.refreshingIn") }}</span>
            <span>{{ timeLeft }}s</span>
          </div>
          <Progress :value="(timeLeft / timeout) * 100" class="h-1 bg-muted" />
        </div>

        <Button
          variant="outline"
          class="w-full"
          :disabled="!!refreshing"
          @click="emit('refresh')"
        >
          <Loader2 v-if="refreshing" class="h-4 w-4 mr-2 animate-spin" />
          <RefreshCw v-else class="h-4 w-4 mr-2" />
          {{ $t("instances.qr_modal.regenerateQR") }}
        </Button>

        <div
          class="space-y-3 rounded-xl border border-border/70 bg-muted/20 p-4"
        >
          <div class="space-y-1">
            <Label for="pair-phone-number">{{
              $t("instances.qr_modal.pairPhone")
            }}</Label>
            <p class="text-sm text-muted-foreground">
              {{ $t("instances.qr_modal.phonePlaceholder") }}
            </p>
          </div>
          <div class="flex gap-2">
            <Input
              id="pair-phone-number"
              v-model="phoneNumber"
              :placeholder="$t('instances.qr_modal.phonePlaceholder')"
            />
            <Button
              variant="outline"
              class="whitespace-nowrap"
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
            class="rounded-xl border border-border/70 bg-background/80 p-3 text-center"
          >
            <p class="mb-1 text-xs text-muted-foreground">
              {{ $t("instances.qr_modal.enterCode") }}
            </p>
            <p class="text-2xl font-mono tracking-widest text-primary">
              {{ pairingCode }}
            </p>
            <p
              v-if="pairingPhoneNumber"
              class="mt-1 text-xs text-muted-foreground"
            >
              {{ $t("instances.qr_modal.phone") }}: {{ pairingPhoneNumber }}
            </p>
          </div>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
