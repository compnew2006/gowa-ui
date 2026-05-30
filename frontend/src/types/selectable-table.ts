export interface SelectableTableColumn {
  key: string
  label: string
  width?: string
  align?: 'left' | 'center' | 'right'
}

export interface PaginationParams {
  page: number
  limit: number
  q?: string
  status?: string
}

export interface PaginationResponseEnvelope<T> {
  data: T[]
  total: number
  page: number
  limit: number
}

export interface SelectionState<T> {
  selectedIds: Set<string | number>
  selectedRecords: T[]
  isAllMatchingSelected: boolean
  isAllPageSelected: boolean
  selectedCount: number
}
