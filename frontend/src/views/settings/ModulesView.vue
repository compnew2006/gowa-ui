<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { Boxes, Loader2 } from "lucide-vue-next";
import { toast } from "vue-sonner";
import { useI18n } from "vue-i18n";
import { PageHeader } from "@/components/shared";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useAuthStore } from "@/stores/auth";
import { useConfigStore } from "@/stores/config";
import {
  modulesService,
  type ManagedModule,
} from "@/services/modules";
import { unwrapResponse } from "@/lib/api-utils";

const { t } = useI18n();
const authStore = useAuthStore();
const configStore = useConfigStore();
const modules = ref<ManagedModule[]>([]);
const loading = ref(false);
const updatingKey = ref("");

const organizationId = computed(() => authStore.organizationId);
const canManageGlobal = computed(
  () => authStore.user?.is_super_admin === true,
);

async function loadModules() {
  loading.value = true;
  try {
    const response = organizationId.value
      ? await modulesService.listOrganization(organizationId.value)
      : await modulesService.listGlobal();
    modules.value = unwrapResponse<ManagedModule[]>(response);
  } catch {
    toast.error(t("modules.loadFailed"));
  } finally {
    loading.value = false;
  }
}

async function updateOrganization(module: ManagedModule) {
  if (!organizationId.value) return;
  updatingKey.value = `organization:${module.key}`;
  try {
    await modulesService.updateOrganization(
      organizationId.value,
      module.key,
      !module.organization_enabled,
    );
    await Promise.all([loadModules(), configStore.fetchModules(true)]);
    toast.success(t("modules.updated"));
  } catch {
    toast.error(t("modules.updateFailed"));
  } finally {
    updatingKey.value = "";
  }
}

async function updateGlobal(module: ManagedModule) {
  updatingKey.value = `global:${module.key}`;
  try {
    await modulesService.updateGlobal(module.key, !module.global_enabled);
    await Promise.all([loadModules(), configStore.fetchModules(true)]);
    toast.success(t("modules.updated"));
  } catch {
    toast.error(t("modules.updateFailed"));
  } finally {
    updatingKey.value = "";
  }
}

onMounted(loadModules);
</script>

<template>
  <div class="flex h-full flex-col">
    <PageHeader
      :title="t('modules.title')"
      :description="t('modules.description')"
    />

    <div class="flex-1 space-y-4 overflow-auto p-6">
      <div
        v-if="loading"
        class="flex min-h-40 items-center justify-center text-muted-foreground"
      >
        <Loader2 class="h-6 w-6 animate-spin" />
      </div>

      <div
        v-else-if="modules.length === 0"
        class="rounded-lg border border-dashed p-8 text-center text-muted-foreground"
      >
        {{ t("modules.empty") }}
      </div>

      <Card v-for="module in modules" v-else :key="module.key">
        <CardHeader class="gap-2">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="flex items-start gap-3">
              <Boxes class="mt-1 h-5 w-5 text-muted-foreground" />
              <div>
                <CardTitle>{{ module.display_name }}</CardTitle>
                <CardDescription>
                  {{ module.key }} · v{{ module.version }}
                </CardDescription>
              </div>
            </div>
            <Badge :variant="module.effective_enabled ? 'default' : 'secondary'">
              {{
                module.effective_enabled
                  ? t("modules.enabled")
                  : t("modules.disabled")
              }}
            </Badge>
          </div>
        </CardHeader>

        <CardContent class="space-y-4">
          <p
            v-if="module.dependencies?.length"
            class="text-sm text-muted-foreground"
          >
            {{ t("modules.dependencies") }}:
            {{ module.dependencies.join(", ") }}
          </p>

          <div class="flex flex-wrap gap-2">
            <Button
              v-if="organizationId"
              :data-testid="`organization-toggle-${module.key}`"
              variant="outline"
              :disabled="updatingKey !== '' || !module.global_enabled"
              @click="updateOrganization(module)"
            >
              <Loader2
                v-if="updatingKey === `organization:${module.key}`"
                class="mr-2 h-4 w-4 animate-spin"
              />
              {{
                module.organization_enabled
                  ? t("modules.disableForOrganization")
                  : t("modules.enableForOrganization")
              }}
            </Button>

            <Button
              v-if="canManageGlobal"
              :data-testid="`global-toggle-${module.key}`"
              variant="outline"
              :disabled="updatingKey !== ''"
              @click="updateGlobal(module)"
            >
              <Loader2
                v-if="updatingKey === `global:${module.key}`"
                class="mr-2 h-4 w-4 animate-spin"
              />
              {{
                module.global_enabled
                  ? t("modules.disableGlobally")
                  : t("modules.enableGlobally")
              }}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
