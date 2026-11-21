import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { CalendarDate } from '@internationalized/date'
import { useDateRange } from './useDateRange'

describe('useDateRange custom picker', () => {
  it('auto-opens the date picker when Custom Range is selected', async () => {
    const { selectedRange, isDatePickerOpen } = useDateRange()
    expect(isDatePickerOpen.value).toBe(false)

    selectedRange.value = 'custom'
    await nextTick()

    expect(isDatePickerOpen.value).toBe(true)
  })

  it('does not open the picker for non-custom presets', async () => {
    const { selectedRange, isDatePickerOpen } = useDateRange()
    selectedRange.value = '7days'
    await nextTick()

    expect(isDatePickerOpen.value).toBe(false)
  })

  it('closes the picker when a complete custom range is applied', async () => {
    const { selectedRange, customDateRange, isDatePickerOpen, applyCustomRange } = useDateRange()
    selectedRange.value = 'custom'
    await nextTick()
    expect(isDatePickerOpen.value).toBe(true)

    customDateRange.value = {
      start: new CalendarDate(2026, 11, 1),
      end: new CalendarDate(2026, 11, 21),
    }
    applyCustomRange()

    expect(isDatePickerOpen.value).toBe(false)
  })
})
