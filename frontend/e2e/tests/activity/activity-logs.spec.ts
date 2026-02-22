import { test, expect } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'
import { ApiHelper } from '../../helpers/api'
import { ActivityLogsPage } from '../../pages'

test.describe('Activity Logs', () => {
  let activityLogsPage: ActivityLogsPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    activityLogsPage = new ActivityLogsPage(page)
    await activityLogsPage.goto()
  })

  test('should display activity logs page', async () => {
    await activityLogsPage.expectPageVisible()
    await expect(activityLogsPage.refreshButton).toBeVisible()
  })

  test('should show custom activity event created via API in history', async ({ request }) => {
    const suffix = Date.now()
    const eventType = `ui.e2e_activity_${suffix}`
    const action = `custom_action_${suffix}`
    const api = new ApiHelper(request)
    await api.loginAsAdmin()

    await api.createActivityLog({
      category: 'custom',
      event_type: eventType,
      action,
      source: 'playwright',
      test_case: 'activity-log-create',
      suffix
    })

    await activityLogsPage.clickRefresh()
    await activityLogsPage.expectRowContains(eventType)
    await activityLogsPage.expectRowContains(action)
  })

  test('should filter activity logs by event type', async ({ request }) => {
    const suffix = Date.now()
    const matchingEventType = `ui.e2e_match_${suffix}`
    const otherEventType = `ui.e2e_other_${suffix}`
    const api = new ApiHelper(request)
    await api.loginAsAdmin()

    await api.createActivityLog({
      category: 'custom',
      event_type: matchingEventType,
      action: `action_match_${suffix}`,
      metadata: { tag: 'match' }
    })
    await api.createActivityLog({
      category: 'custom',
      event_type: otherEventType,
      action: `action_other_${suffix}`,
      metadata: { tag: 'other' }
    })

    await activityLogsPage.clickRefresh()
    await activityLogsPage.applyEventTypeFilter(matchingEventType)
    await activityLogsPage.clickApplyFilters()

    await activityLogsPage.expectRowContains(matchingEventType)
    await activityLogsPage.expectRowNotContains(otherEventType)
  })
})
