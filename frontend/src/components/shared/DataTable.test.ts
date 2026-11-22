/**
 * Task 11: the shared settings table must keep its Actions column
 * reachable at narrow widths (978px).
 *
 * Today the DataTable wrapper only scrolls when maxHeight is set, so a wide
 * table propagates its min-content width up through the page (the settings
 * views sit inside a Radix ScrollArea viewport) and the page itself scrolls
 * horizontally — the row and its Actions column slide away together. The
 * table wrapper itself must be the horizontal scroll container, so the
 * Actions column is reachable by scrolling within the table while the row
 * stays in place.
 */
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import DataTable from './DataTable.vue'
import type { Column } from './types'

const columns: Column[] = [
  { key: 'name', label: 'Name' },
  { key: 'email', label: 'Email' },
  { key: 'role', label: 'Role' },
  { key: 'status', label: 'Status' },
  { key: 'created', label: 'Created' },
  { key: 'actions', label: 'Actions' },
]

const items = [
  { id: '1', name: 'Sara', email: 'sara@example.com', role: 'admin', status: 'active', created: '2026-01-01' },
]

describe('DataTable horizontal scroll', () => {
  it('the table wrapper scrolls horizontally in-table (no maxHeight needed)', () => {
    const wrapper = mount(DataTable, {
      props: { items, columns },
      slots: { 'cell-actions': '<button>Edit</button>' },
    })
    const scroller = wrapper.find('[data-testid="data-table-scroll"]')
    expect(scroller.exists()).toBe(true)
    expect(scroller.classes()).toContain('overflow-x-auto')
  })

  it('the Actions column header and its row actions render', () => {
    const wrapper = mount(DataTable, {
      props: { items, columns },
      slots: { 'cell-actions': '<button>Edit</button>' },
    })
    const headers = wrapper.findAll('th').map((th) => th.text())
    expect(headers[headers.length - 1]).toBe('Actions')
    expect(wrapper.find('tbody').text()).toContain('Edit')
  })

  it('with maxHeight the wrapper keeps vertical scroll behaviour', () => {
    const wrapper = mount(DataTable, {
      props: { items, columns, maxHeight: 'calc(100vh - 320px)' },
    })
    const scroller = wrapper.find('[data-testid="data-table-scroll"]')
    expect(scroller.exists()).toBe(true)
    expect(scroller.classes()).toContain('overflow-auto')
    expect(scroller.attributes('style')).toContain('calc(100vh - 320px)')
  })
})
