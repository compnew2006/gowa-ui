<script setup lang="ts" generic="T extends Record<string, any>">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Checkbox } from '@/components/ui/checkbox'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  Search,
  CheckCircle2
} from 'lucide-vue-next'
import { getPageNumbers } from '@/composables/usePagination'

export interface Column<__T = unknown> {
  key: string
  label: string
  width?: string
  align?: 'left' | 'center' | 'right'
}

const props = withDefaults(defineProps<{
  items: T[]
  columns: Column<T>[]
  isLoading?: boolean
  rowKey?: string
  
  // Pagination
  currentPage: number
  totalPages: number
  totalItems: number
  pageSize: number
  
  // Selection
  selectedIds: Set<string | number>
  isAllMatchingSelected: boolean
  isAllPageSelected: boolean
  selectedCount: number
  showSelection?: boolean
  showSelectAllMatching?: boolean
}>(), {
  rowKey: 'id',
  showSelection: true,
  showSelectAllMatching: true
})

const emit = defineEmits<{
  'update:currentPage': [page: number]
  'page-change': [page: number]
  'update:pageSize': [size: number]
  'pageSize-change': [size: number]
  'toggle-row': [item: T]
  'toggle-page': []
  'select-all-matching': []
  'clear-selection': []
}>()

const { t, locale } = useI18n()

const isRtl = computed(() => {
  return locale.value === 'ar' || document.documentElement.dir === 'rtl'
})

// Helper to determine if some, but not all, page items are selected
const isSomePageSelected = computed(() => {
  if (props.items.length === 0) return false
  const selectedOnPage = props.items.filter((item: any) => {
    const id = item[props.rowKey]
    return id !== undefined && props.selectedIds.has(id)
  })
  return selectedOnPage.length > 0 && selectedOnPage.length < props.items.length
})

// Bind checked state to the header checkbox
const isHeaderChecked = computed(() => {
  if (props.isAllPageSelected || props.isAllMatchingSelected) return true
  if (isSomePageSelected.value) return 'indeterminate'
  return false
})

function handlePageChange(page: number) {
  emit('update:currentPage', page)
  emit('page-change', page)
}

function handlePageSizeChange(sizeStr: any) {
  const size = parseInt(String(sizeStr), 10) || 25
  emit('update:pageSize', size)
  emit('pageSize-change', size)
}

function getRowKey(item: T, index: number): string | number {
  return item[props.rowKey] ?? `selectable-row-${index}`
}

const pageNumbers = computed(() => getPageNumbers(props.currentPage, props.totalPages))

// Current displayed item index range calculations
const startItemIndex = computed(() => {
  if (props.items.length === 0) return 0
  return (props.currentPage - 1) * props.pageSize + 1
})

const endItemIndex = computed(() => {
  return Math.min(props.currentPage * props.pageSize, props.totalItems)
})
</script>

<template>
  <div class="space-y-4">
    <!-- Selected Count Banner / Server-side select all matching controls -->
    <div
      v-if="showSelection && selectedCount > 0"
      class="flex flex-wrap items-center justify-between gap-3 px-4 py-3 rounded-lg border bg-emerald-500/5 dark:bg-emerald-500/10 border-emerald-500/20 text-sm animate-fade-in"
    >
      <div class="flex items-center gap-2">
        <CheckCircle2 class="h-4 w-4 text-emerald-500 shrink-0" />
        <span class="font-medium">
          {{ t('selectableTable.selectedCount', { count: selectedCount }) }}
        </span>
      </div>

      <div class="flex items-center gap-4">
        <!-- "Select all matching records" prompt -->
        <Button
          v-if="showSelectAllMatching && isAllPageSelected && !isAllMatchingSelected && totalItems > items.length"
          variant="link"
          size="sm"
          class="h-auto p-0 font-semibold text-emerald-600 dark:text-emerald-400 hover:text-emerald-700 dark:hover:text-emerald-300 transition-colors"
          @click="emit('select-all-matching')"
        >
          {{ t('selectableTable.selectAllMatching', { total: totalItems }) }}
        </Button>
        <span
          v-else-if="isAllMatchingSelected"
          class="text-xs text-muted-foreground font-medium"
        >
          {{ t('selectableTable.allMatchingSelected', { total: totalItems }) }}
        </span>

        <!-- Clear selection trigger -->
        <Button
          variant="ghost"
          size="sm"
          class="h-7 text-xs font-semibold text-rose-600 hover:text-rose-700 hover:bg-rose-500/10 dark:hover:bg-rose-500/20 transition-all rounded"
          @click="emit('clear-selection')"
        >
          {{ t('selectableTable.clearSelection') }}
        </Button>
      </div>
    </div>

    <!-- Main Table Representation -->
    <div class="rounded-md border bg-card shadow-sm overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow class="hover:bg-transparent">
            <!-- First Checkbox Column -->
            <TableHead v-if="showSelection" class="w-[48px] px-4 text-center">
              <Checkbox
                :checked="isHeaderChecked"
                @update:checked="() => emit('toggle-page')"
                aria-label="Select all current page"
                class="translate-y-[2px]"
              />
            </TableHead>

            <TableHead
              v-for="col in columns"
              :key="col.key"
              :class="[
                col.width,
                col.align === 'right' && 'text-right',
                col.align === 'center' && 'text-center',
              ]"
            >
              {{ col.label }}
            </TableHead>
          </TableRow>
        </TableHeader>

        <TableBody>
          <!-- Skeleton Loading Screen -->
          <template v-if="isLoading">
            <TableRow v-for="i in Math.min(pageSize, 5)" :key="'skeleton-' + i">
              <TableCell v-if="showSelection" class="px-4 text-center">
                <Skeleton class="h-4 w-4 rounded-sm mx-auto" />
              </TableCell>
              <TableCell v-for="col in columns" :key="'skeleton-col-' + col.key">
                <Skeleton class="h-4 w-full max-w-[150px] rounded" />
              </TableCell>
            </TableRow>
          </template>

          <!-- Empty State Presentation -->
          <TableRow v-else-if="items.length === 0">
            <TableCell :colspan="columns.length + (showSelection ? 1 : 0)" class="h-32 text-center text-muted-foreground p-6">
              <div class="flex flex-col items-center justify-center space-y-2">
                <Search class="h-8 w-8 opacity-40 animate-pulse" />
                <p class="font-medium text-sm">{{ t('selectableTable.noMatchingData') }}</p>
                <slot name="empty-actions" />
              </div>
            </TableCell>
          </TableRow>

          <!-- Fully Populated Data Rows -->
          <template v-else>
            <TableRow
              v-for="(row, idx) in items"
              :key="getRowKey(row, idx)"
              :class="[
                'transition-colors hover:bg-slate-500/5',
                showSelection && selectedIds.has(row[rowKey]) && 'bg-emerald-500/5 dark:bg-emerald-500/10 hover:bg-emerald-500/10'
              ]"
            >
              <!-- Cell Checkbox -->
              <TableCell v-if="showSelection" class="px-4 text-center">
                <Checkbox
                  :checked="selectedIds.has(row[rowKey])"
                  @update:checked="() => emit('toggle-row', row)"
                  :aria-label="`Select row ${idx + 1}`"
                  class="translate-y-[2px]"
                />
              </TableCell>

              <!-- Mapped Columns Content -->
              <TableCell
                v-for="col in columns"
                :key="col.key"
                :class="[
                  col.align === 'right' && 'text-right',
                  col.align === 'center' && 'text-center',
                ]"
              >
                <slot :name="`cell-${col.key}`" :item="row" :index="idx">
                  {{ row[col.key] }}
                </slot>
              </TableCell>
            </TableRow>
          </template>
        </TableBody>
      </Table>
    </div>

    <!-- Paginated Navigation & Limits Section -->
    <div v-if="totalPages > 1 && !isLoading" class="flex flex-wrap items-center justify-between gap-4 py-2 px-1">
      <!-- range metrics -->
      <p class="text-sm text-muted-foreground">
        Showing {{ startItemIndex }} to {{ endItemIndex }} of {{ totalItems }} items
      </p>

      <!-- pagination button panel -->
      <div class="flex items-center gap-4 flex-wrap">
        <!-- Per page dropdown list -->
        <div class="flex items-center gap-2">
          <span class="text-xs text-muted-foreground font-semibold">Rows per page:</span>
          <Select :model-value="pageSize.toString()" @update:model-value="handlePageSizeChange">
            <SelectTrigger class="h-8 w-[72px]">
              <SelectValue :placeholder="pageSize.toString()" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="25">25</SelectItem>
              <SelectItem value="50">50</SelectItem>
              <SelectItem value="100">100</SelectItem>
              <SelectItem value="200">200</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="flex items-center gap-1.5">
          <!-- First Page -->
          <Button
            variant="outline"
            size="icon"
            class="h-8 w-8 shadow-sm"
            :disabled="currentPage === 1"
            @click="handlePageChange(1)"
            aria-label="First page"
          >
            <component :is="isRtl ? ChevronsRight : ChevronsLeft" class="h-4 w-4" />
          </Button>

          <!-- Previous Page -->
          <Button
            variant="outline"
            size="icon"
            class="h-8 w-8 shadow-sm"
            :disabled="currentPage === 1"
            @click="handlePageChange(currentPage - 1)"
            aria-label="Previous page"
          >
            <component :is="isRtl ? ChevronRight : ChevronLeft" class="h-4 w-4" />
          </Button>

          <!-- Surrounding pages window -->
          <div class="flex items-center gap-1 mx-1.5">
            <template v-for="(p, index) in pageNumbers" :key="index">
              <Button
                v-if="p !== '...'"
                :variant="p === currentPage ? 'default' : 'outline'"
                size="icon"
                class="h-8 w-8 shadow-sm"
                @click="handlePageChange(p as number)"
              >
                {{ p }}
              </Button>
              <span v-else class="px-1.5 text-muted-foreground text-xs font-bold">...</span>
            </template>
          </div>

          <!-- Next Page -->
          <Button
            variant="outline"
            size="icon"
            class="h-8 w-8 shadow-sm"
            :disabled="currentPage === totalPages"
            @click="handlePageChange(currentPage + 1)"
            aria-label="Next page"
          >
            <component :is="isRtl ? ChevronLeft : ChevronRight" class="h-4 w-4" />
          </Button>

          <!-- Last Page -->
          <Button
            variant="outline"
            size="icon"
            class="h-8 w-8 shadow-sm"
            :disabled="currentPage === totalPages"
            @click="handlePageChange(totalPages)"
            aria-label="Last page"
          >
            <component :is="isRtl ? ChevronsLeft : ChevronsRight" class="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.animate-fade-in {
  animation: fadeIn 0.2s ease-out forwards;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(-4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
