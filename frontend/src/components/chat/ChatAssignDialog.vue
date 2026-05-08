<script setup lang="ts">
import { ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  Search,
  Check,
  User,
  UserMinus,
} from "lucide-vue-next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface AssignableUser {
  id: string;
  full_name: string;
  role?: { name: string } | null;
}

const props = defineProps<{
  open: boolean;
  assignedUserId: string | null;
  users: AssignableUser[];
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
  assign: [userId: string | null];
}>();

const { t } = useI18n();
const searchQuery = ref("");

const filteredUsers = computed(() => {
  const q = searchQuery.value.toLowerCase().trim();
  if (!q) return props.users;
  return props.users.filter((u) =>
    u.full_name.toLowerCase().includes(q),
  );
});

function handleAssign(userId: string | null) {
  emit("assign", userId);
  emit("update:open", false);
}

function handleOpenChange(open: boolean) {
  if (!open) searchQuery.value = "";
  emit("update:open", open);
}
</script>

<template>
  <Dialog :open="open" @update:open="handleOpenChange">
    <DialogContent class="max-w-sm">
      <DialogHeader>
        <DialogTitle>{{ t("chat.assignContact") }}</DialogTitle>
        <DialogDescription>
          {{ t("chat.assignContactDesc") }}
        </DialogDescription>
      </DialogHeader>
      <div class="py-4 space-y-3">
        <div class="relative">
          <Search
            class="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground"
          />
          <Input
            v-model="searchQuery"
            :placeholder="t('chat.searchUsers') + '...'"
            class="pl-9 h-9"
          />
        </div>
        <Button
          v-if="assignedUserId"
          variant="outline"
          class="w-full justify-start"
          @click="handleAssign(null)"
        >
          <UserMinus class="mr-2 h-4 w-4" />
          {{ t("chat.unassignContact") }}
        </Button>
        <Separator />
        <ScrollArea class="max-h-[280px]">
          <div class="space-y-1">
            <Button
              v-for="user in filteredUsers"
              :key="user.id"
              :variant="assignedUserId === user.id ? 'secondary' : 'ghost'"
              class="w-full justify-start"
              @click="handleAssign(user.id)"
            >
              <User class="mr-2 h-4 w-4" />
              <span>{{ user.full_name }}</span>
              <Check
                v-if="assignedUserId === user.id"
                class="ml-auto h-4 w-4 text-primary"
              />
              <Badge v-else variant="outline" class="ml-auto text-xs">
                {{ user.role?.name }}
              </Badge>
            </Button>
            <p
              v-if="filteredUsers.length === 0"
              class="text-sm text-muted-foreground text-center py-4"
            >
              {{ t("chat.noUsersFound") }}
            </p>
          </div>
        </ScrollArea>
      </div>
    </DialogContent>
  </Dialog>
</template>
