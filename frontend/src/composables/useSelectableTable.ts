import { ref, computed, watch, type Ref, type ComputedRef } from 'vue'
import type { PaginationParams } from '@/types/selectable-table'

export interface UseSelectableTableOptions {
  initialPageSize?: number
  rowKey?: string
  debounceMs?: number
}

export interface UseSelectableTableResult<T> {
  items: Ref<T[]>
  totalItems: Ref<number>
  isLoading: Ref<boolean>
  currentPage: Ref<number>
  pageSize: Ref<number>
  totalPages: ComputedRef<number>
  searchQuery: Ref<string>
  debouncedSearchQuery: Ref<string>
  filterStatus: Ref<string>
  selectedIds: Ref<Set<string | number>>
  selectedRecords: ComputedRef<T[]>
  isAllPageSelected: ComputedRef<boolean>
  isAllMatchingSelected: Ref<boolean>
  selectedCount: ComputedRef<number>
  toggleRow: (item: T) => void
  togglePageSelection: () => void
  selectAllMatching: () => void
  clearSelection: () => void
  goToPage: (page: number) => void
  changePageSize: (size: number) => void
  loadData: () => Promise<void>
  resetTable: () => void
}

export function useSelectableTable<T>(
  fetchDataFn: (params: PaginationParams) => Promise<{ data: T[]; total: number }>,
  options: UseSelectableTableOptions = {}
): UseSelectableTableResult<T> {
  const {
    initialPageSize = 25,
    rowKey = 'id',
    debounceMs = 300
  } = options

  // Core Data States
  const items = ref<T[]>([]) as Ref<T[]>
  const totalItems = ref(0)
  const isLoading = ref(false)

  // Pagination & Filtering States
  const currentPage = ref(1)
  const pageSize = ref(initialPageSize)
  const searchQuery = ref('')
  const debouncedSearchQuery = ref('')
  const filterStatus = ref('all')

  // Selection States
  const selectedIds = ref<Set<string | number>>(new Set())
  const selectedData = ref<Map<string | number, any>>(new Map())
  const isAllMatchingSelected = ref(false)

  const totalPages = computed(() => {
    return Math.ceil(totalItems.value / pageSize.value) || 1
  })

  // Computed array of selected record objects
  const selectedRecords = computed<T[]>(() => {
    return Array.from(selectedData.value.values()) as T[]
  })

  // Check if every item on the current page is selected
  const isAllPageSelected = computed(() => {
    if (items.value.length === 0) return false
    return items.value.every((item: any) => {
      const id = item[rowKey]
      return id !== undefined && selectedIds.value.has(id)
    })
  })

  // running count of selections
  const selectedCount = computed(() => {
    if (isAllMatchingSelected.value) {
      return totalItems.value
    }
    return selectedIds.value.size
  })

  // Toggle selection for a single row
  function toggleRow(item: T): void {
    const id = (item as any)[rowKey]
    if (id === undefined) return

    // If 'all matching' across pages was selected, individual toggle drops that state
    if (isAllMatchingSelected.value) {
      isAllMatchingSelected.value = false
    }

    if (selectedIds.value.has(id)) {
      selectedIds.value.delete(id)
      selectedData.value.delete(id)
    } else {
      selectedIds.value.add(id)
      selectedData.value.set(id, item)
    }
  }

  // Toggle selection for every item on the current page
  function togglePageSelection(): void {
    if (isAllMatchingSelected.value) {
      clearSelection()
      return
    }

    const allOnPageChecked = isAllPageSelected.value

    items.value.forEach((item: any) => {
      const id = item[rowKey]
      if (id === undefined) return

      if (allOnPageChecked) {
        // Deselect
        selectedIds.value.delete(id)
        selectedData.value.delete(id)
      } else {
        // Select
        selectedIds.value.add(id)
        selectedData.value.set(id, item)
      }
    })
  }

  // Flag selection as server-side all matching
  function selectAllMatching(): void {
    isAllMatchingSelected.value = true
  }

  // Clear every selection
  function clearSelection(): void {
    selectedIds.value.clear()
    selectedData.value.clear()
    isAllMatchingSelected.value = false
  }

  // Go to page
  function goToPage(page: number): void {
    const clampedPage = Math.min(Math.max(1, page), totalPages.value)
    currentPage.value = clampedPage
  }

  // Change page size (resets page to 1)
  function changePageSize(size: number): void {
    pageSize.value = size
    currentPage.value = 1
  }

  // Core Data Fetch Coordinator
  async function loadData(): Promise<void> {
    isLoading.value = true
    try {
      const params: PaginationParams = {
        page: currentPage.value,
        limit: pageSize.value,
        q: debouncedSearchQuery.value.trim() || undefined,
        status: filterStatus.value !== 'all' ? filterStatus.value : undefined
      }
      const response = await fetchDataFn(params)
      items.value = response.data || []
      totalItems.value = response.total || 0
    } catch (err) {
      console.error('Failed to load selectable table data:', err)
      items.value = []
      totalItems.value = 0
    } finally {
      isLoading.value = false
    }
  }

  // Reset pagination and clear selections on major filter reset
  function resetTable(): void {
    currentPage.value = 1
    clearSelection()
  }

  // Debounced search logic watcher
  let searchDebounceTimer: any = null
  watch(searchQuery, (newVal) => {
    if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
    searchDebounceTimer = setTimeout(() => {
      debouncedSearchQuery.value = newVal
    }, debounceMs)
  })

  // Watchers to trigger automatic data reloading
  watch([currentPage, pageSize, debouncedSearchQuery, filterStatus], () => {
    void loadData()
  })

  // When filters or search keywords change, reset current page to 1 and reset selections
  watch([debouncedSearchQuery, filterStatus], () => {
    resetTable()
  })

  return {
    items,
    totalItems,
    isLoading,
    currentPage,
    pageSize,
    totalPages,
    searchQuery,
    debouncedSearchQuery,
    filterStatus,
    selectedIds,
    selectedRecords,
    isAllPageSelected,
    isAllMatchingSelected,
    selectedCount,
    toggleRow,
    togglePageSelection,
    selectAllMatching,
    clearSelection,
    goToPage,
    changePageSize,
    loadData,
    resetTable
  }
}
