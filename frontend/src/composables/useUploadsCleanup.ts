import { computed, ref } from "vue";
import { toast } from "vue-sonner";
import { organizationService } from "@/services/api";
import { getErrorMessage, unwrapResponse } from "@/lib/api-utils";
import { useI18n } from "vue-i18n";

const MAX_UPLOADS_CLEANUP_RETENTION_DAYS = 3650;
const DEFAULT_UPLOADS_CLEANUP_SCHEDULE_HOUR = 3;

const isUploadsCleanupSubmitting = ref(false);
const isUploadsCleanupRunning = ref(false);
const uploadsCleanupConfirmOpen = ref(false);

interface UploadsCleanupSettingsForm {
  retention_days: string | number;
  schedule_hour: string;
  timezone: string;
}

const uploadsCleanupSettings = ref<UploadsCleanupSettingsForm>({
  retention_days: "0",
  schedule_hour: formatUploadsCleanupScheduleTime(
    DEFAULT_UPLOADS_CLEANUP_SCHEDULE_HOUR,
  ),
  timezone: "UTC",
});

function parseUploadsCleanupRetentionDaysInput(value: unknown): number | null {
  if (typeof value === "number") {
    if (
      !Number.isInteger(value) ||
      value < 0 ||
      value > MAX_UPLOADS_CLEANUP_RETENTION_DAYS
    ) {
      return null;
    }

    return value;
  }

  if (typeof value !== "string") {
    return null;
  }

  const trimmed = value.trim();
  if (trimmed === "") return 0;
  if (!/^\d+$/.test(trimmed)) {
    return null;
  }

  const parsed = Number(trimmed);
  if (
    !Number.isInteger(parsed) ||
    parsed < 0 ||
    parsed > MAX_UPLOADS_CLEANUP_RETENTION_DAYS
  ) {
    return null;
  }

  return parsed;
}

function parseUploadsCleanupScheduleHourInput(value: unknown): number | null {
  if (typeof value === "number") {
    if (!Number.isInteger(value) || value < 0 || value > 23) {
      return null;
    }

    return value;
  }

  if (typeof value !== "string") {
    return null;
  }

  const trimmed = value.trim();
  if (trimmed === "") {
    return DEFAULT_UPLOADS_CLEANUP_SCHEDULE_HOUR;
  }

  const match = /^(\d{1,2})(?::([0-5]\d))?$/.exec(trimmed);
  if (!match) {
    return null;
  }

  const parsed = Number(match[1]);
  const minutes = match[2];
  if (
    !Number.isInteger(parsed) ||
    parsed < 0 ||
    parsed > 23 ||
    (minutes !== undefined && minutes !== "00")
  ) {
    return null;
  }

  return parsed;
}

function formatUploadsCleanupScheduleTime(value: unknown): string {
  const hour =
    parseUploadsCleanupScheduleHourInput(value) ??
    DEFAULT_UPLOADS_CLEANUP_SCHEDULE_HOUR;
  return `${String(hour).padStart(2, "0")}:00`;
}

export function useUploadsCleanup() {
  const { t } = useI18n();

  const uploadsCleanupScheduleLabel = computed(() =>
    formatUploadsCleanupScheduleTime(uploadsCleanupSettings.value.schedule_hour),
  );

  function buildUploadsCleanupPayload() {
    const retentionDays = parseUploadsCleanupRetentionDaysInput(
      uploadsCleanupSettings.value.retention_days,
    );
    if (retentionDays === null) {
      toast.error(t("settings.uploadsCleanupRetentionDaysInvalid"));
      return null;
    }

    const scheduleHour = parseUploadsCleanupScheduleHourInput(
      uploadsCleanupSettings.value.schedule_hour,
    );
    if (scheduleHour === null) {
      toast.error(t("settings.uploadsCleanupScheduleHourInvalid"));
      return null;
    }

    return {
      uploads_cleanup_retention_days: retentionDays,
      uploads_cleanup_schedule_hour: scheduleHour,
    };
  }

  async function saveUploadsCleanupSettings() {
    const payload = buildUploadsCleanupPayload();
    if (!payload) {
      return;
    }

    isUploadsCleanupSubmitting.value = true;
    try {
      await organizationService.updateSettings(payload);
      uploadsCleanupSettings.value.schedule_hour =
        formatUploadsCleanupScheduleTime(payload.uploads_cleanup_schedule_hour);
      uploadsCleanupSettings.value.retention_days = String(
        payload.uploads_cleanup_retention_days,
      );
      toast.success(t("settings.uploadsCleanupSaved"));
    } catch (error) {
      toast.error(getErrorMessage(error, t("settings.uploadsCleanupSaveFailed")));
    } finally {
      isUploadsCleanupSubmitting.value = false;
    }
  }

  async function runUploadsCleanupNow(canEdit: boolean) {
    let payload: {
      uploads_cleanup_retention_days: number;
      uploads_cleanup_schedule_hour: number;
    } | null = null;

    if (canEdit) {
      payload = buildUploadsCleanupPayload();
      if (!payload) {
        return;
      }
    }

    isUploadsCleanupRunning.value = true;
    try {
      if (payload) {
        await organizationService.updateSettings(payload);
        uploadsCleanupSettings.value.schedule_hour = String(
          payload.uploads_cleanup_schedule_hour,
        );
        uploadsCleanupSettings.value.retention_days = String(
          payload.uploads_cleanup_retention_days,
        );
      }

      const response = await organizationService.runUploadsCleanupNow();
      const result = unwrapResponse<{
        message: string;
        deleted_files: number;
        retention_days: number;
      }>(response);
      toast.success(
        result.message ||
          t("settings.uploadsCleanupRunSuccess", {
            count: result.deleted_files,
            days: result.retention_days,
          }),
      );
    } catch (error) {
      toast.error(getErrorMessage(error, t("settings.uploadsCleanupRunFailed")));
    } finally {
      isUploadsCleanupRunning.value = false;
    }
  }

  function requestRunUploadsCleanupNow() {
    uploadsCleanupConfirmOpen.value = true;
  }

  return {
    MAX_UPLOADS_CLEANUP_RETENTION_DAYS,
    DEFAULT_UPLOADS_CLEANUP_SCHEDULE_HOUR,
    isUploadsCleanupSubmitting,
    isUploadsCleanupRunning,
    uploadsCleanupConfirmOpen,
    uploadsCleanupSettings,
    uploadsCleanupScheduleLabel,
    formatUploadsCleanupScheduleTime,
    buildUploadsCleanupPayload,
    saveUploadsCleanupSettings,
    runUploadsCleanupNow,
    requestRunUploadsCleanupNow,
  };
}
