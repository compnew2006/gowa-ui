<script setup lang="ts">
import { ref, watch, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import { Badge } from "@/components/ui/badge";
import { PageHeader, SearchInput } from "@/components/shared";
import { useGroupParticipants } from "@/composables/useGroupParticipants";
import {
  Users,
  UserPlus,
  UserMinus,
  Shield,
  ShieldOff,
  Loader2,
  Crown,
  Search,
} from "lucide-vue-next";

const { t } = useI18n();
const gp = useGroupParticipants();

// Add participants dialog
const showAddDialog = ref(false);
const newParticipantsInput = ref("");
const isAdding = ref(false);

async function confirmAdd() {
  isAdding.value = true;
  await gp.addParticipants(newParticipantsInput.value);
  isAdding.value = false;
  showAddDialog.value = false;
  newParticipantsInput.value = "";
}

// Watch for instance change to load groups
watch(gp.selectedInstanceId, () => {
  gp.selectedGroupJid.value = "";
  gp.participants.value = [];
  gp.fetchGroups();
});

// Watch for group change to load participants
watch(gp.selectedGroupJid, () => {
  gp.fetchParticipants();
});

onMounted(() => {
  gp.fetchInstances();
});
</script>

<template>
  <div class="flex h-full flex-col gap-6 p-6">
    <PageHeader
      :title="t('groupParticipants.title')"
      :subtitle="t('groupParticipants.subtitle')"
    />

    <!-- Instance & Group Selection -->
    <div class="flex flex-wrap items-center gap-3">
      <Select v-model="gp.selectedInstanceId.value">
        <SelectTrigger class="w-[220px]">
          <SelectValue :placeholder="t('groupParticipants.selectInstance')" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem
            v-for="inst in gp.instances.value"
            :key="inst.id"
            :value="inst.id"
          >
            {{ inst.name }}
          </SelectItem>
        </SelectContent>
      </Select>

      <Select v-model="gp.selectedGroupJid.value" :disabled="!gp.selectedInstanceId.value">
        <SelectTrigger class="w-[280px]">
          <SelectValue :placeholder="t('groupParticipants.selectGroup')" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem
            v-for="g in gp.groups.value"
            :key="g.jid"
            :value="g.jid"
          >
            {{ g.name }} ({{ g.participant_count }})
          </SelectItem>
        </SelectContent>
      </Select>

      <Button
        variant="outline"
        size="sm"
        :disabled="!gp.selectedGroupJid.value || gp.isLoadingParticipants.value"
        @click="gp.fetchParticipants()"
      >
        <Loader2 v-if="gp.isLoadingParticipants.value" class="w-4 h-4 animate-spin mr-1" />
        <Search v-else class="w-4 h-4 mr-1" />
        {{ t("groupParticipants.refresh") }}
      </Button>
    </div>

    <!-- Participants Content -->
    <Card v-if="gp.selectedGroupJid.value">
      <CardContent class="p-0">
        <!-- Toolbar -->
        <div class="flex flex-wrap items-center gap-3 border-b p-4">
          <SearchInput
            v-model="gp.searchQuery.value"
            :placeholder="t('groupParticipants.searchPlaceholder')"
            class="w-[260px]"
          />

          <div class="flex-1" />

          <div class="flex items-center gap-2">
            <Badge v-if="gp.selectedCount.value > 0" variant="secondary">
              {{ gp.selectedCount.value }} {{ t("groupParticipants.selected") }}
            </Badge>

            <Button
              variant="outline"
              size="sm"
              :disabled="gp.selectedCount.value === 0"
              @click="gp.promoteSelected()"
            >
              <Shield class="w-4 h-4 mr-1" />
              {{ t("groupParticipants.promote") }}
            </Button>

            <Button
              variant="outline"
              size="sm"
              :disabled="gp.selectedCount.value === 0"
              @click="gp.demoteSelected()"
            >
              <ShieldOff class="w-4 h-4 mr-1" />
              {{ t("groupParticipants.demote") }}
            </Button>

            <Button
              variant="destructive"
              size="sm"
              :disabled="gp.selectedCount.value === 0"
              @click="gp.removeSelected()"
            >
              <UserMinus class="w-4 h-4 mr-1" />
              {{ t("groupParticipants.remove") }}
            </Button>

            <Button size="sm" @click="showAddDialog = true">
              <UserPlus class="w-4 h-4 mr-1" />
              {{ t("groupParticipants.addMembers") }}
            </Button>
          </div>
        </div>

        <!-- Loading -->
        <div v-if="gp.isLoadingParticipants.value" class="flex justify-center py-12">
          <Loader2 class="w-6 h-6 animate-spin text-muted-foreground" />
        </div>

        <!-- Empty State -->
        <div
          v-else-if="gp.filteredParticipants.value.length === 0"
          class="flex flex-col items-center justify-center py-12 text-muted-foreground"
        >
          <Users class="w-12 h-12 mb-3 opacity-50" />
          <p>{{ t("groupParticipants.noMembers") }}</p>
        </div>

        <!-- Participants Table -->
        <div v-else class="overflow-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b bg-muted/50">
                <th class="w-12 p-3 text-center">
                  <Checkbox
                    :checked="
                      gp.filteredParticipants.value.length > 0 &&
                      gp.selectedCount.value === gp.filteredParticipants.value.length
                    "
                    @update:checked="gp.toggleAll()"
                  />
                </th>
                <th class="p-3 text-left font-medium">{{ t("groupParticipants.phoneNumber") }}</th>
                <th class="p-3 text-left font-medium">{{ t("groupParticipants.jid") }}</th>
                <th class="p-3 text-center font-medium">{{ t("groupParticipants.status") }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="p in gp.filteredParticipants.value"
                :key="p.jid"
                class="border-b transition-colors hover:bg-muted/50"
              >
                <td class="p-3 text-center">
                  <Checkbox
                    :checked="gp.selectedJids.value.has(p.jid)"
                    @update:checked="gp.toggleParticipant(p.jid)"
                  />
                </td>
                <td class="p-3 font-mono text-xs">
                  {{ p.phone_number || p.jid.replace(/@.*$/, "") }}
                </td>
                <td class="p-3 font-mono text-xs text-muted-foreground">
                  {{ p.jid }}
                </td>
                <td class="p-3 text-center">
                  <Badge v-if="p.is_super_admin" variant="default" class="bg-amber-500">
                    <Crown class="w-3 h-3 mr-1" />
                    {{ t("groupParticipants.superAdmin") }}
                  </Badge>
                  <Badge v-else-if="p.is_admin" variant="default" class="bg-blue-500">
                    <Shield class="w-3 h-3 mr-1" />
                    {{ t("groupParticipants.admin") }}
                  </Badge>
                  <Badge v-else variant="secondary">
                    {{ t("groupParticipants.member") }}
                  </Badge>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>

    <!-- No Group Selected -->
    <div
      v-else
      class="flex flex-col items-center justify-center py-16 text-muted-foreground"
    >
      <Users class="w-16 h-16 mb-4 opacity-30" />
      <p class="text-lg">{{ t("groupParticipants.selectGroupHint") }}</p>
    </div>

    <!-- Add Participants Dialog -->
    <Dialog v-model:open="showAddDialog">
      <DialogContent class="max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ t("groupParticipants.addMembersTitle") }}</DialogTitle>
          <DialogDescription>
            {{ t("groupParticipants.addMembersDescription") }}
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-3">
          <Textarea
            v-model="newParticipantsInput"
            :placeholder="t('groupParticipants.addMembersPlaceholder')"
            :rows="6"
            class="font-mono text-sm"
          />
          <p class="text-xs text-muted-foreground">
            {{ t("groupParticipants.addMembersHint") }}
          </p>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="showAddDialog = false">
            {{ t("common.cancel") }}
          </Button>
          <Button :disabled="!newParticipantsInput.trim() || isAdding" @click="confirmAdd">
            <Loader2 v-if="isAdding" class="w-4 h-4 animate-spin mr-1" />
            {{ t("groupParticipants.addMembersConfirm") }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
