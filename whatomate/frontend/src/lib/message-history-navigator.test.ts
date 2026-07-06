import { describe, expect, it } from 'vitest'

import { MessageHistoryNavigator } from './message-history-navigator'

describe('MessageHistoryNavigator', () => {
  it('returns immediately when target message is already loaded', async () => {
    let loadCalls = 0
    const navigator = new MessageHistoryNavigator({
      hasMessage: (messageId) => messageId === 'target',
      hasMoreMessages: () => true,
      loadOlderMessages: async () => {
        loadCalls += 1
      },
      getBoundaryToken: () => 'm-3'
    })

    const found = await navigator.loadUntilMessage('target')

    expect(found).toBe(true)
    expect(loadCalls).toBe(0)
  })

  it('loads older history until target message is found', async () => {
    const loadedMessages = new Set<string>(['m-3', 'm-4'])
    const historyWindows = [
      ['m-1', 'm-2', 'm-3', 'm-4']
    ]

    let oldestBoundary = 'm-3'
    const navigator = new MessageHistoryNavigator({
      hasMessage: (messageId) => loadedMessages.has(messageId),
      hasMoreMessages: () => oldestBoundary !== 'm-1',
      getBoundaryToken: () => oldestBoundary,
      loadOlderMessages: async () => {
        const nextWindow = historyWindows.shift()
        if (!nextWindow) return
        nextWindow.forEach((messageId) => loadedMessages.add(messageId))
        oldestBoundary = nextWindow[0]
      }
    })

    const found = await navigator.loadUntilMessage('m-1', { maxRequests: 3 })

    expect(found).toBe(true)
  })

  it('stops after stagnant loads when history does not advance', async () => {
    let loadCalls = 0
    const navigator = new MessageHistoryNavigator({
      hasMessage: () => false,
      hasMoreMessages: () => true,
      getBoundaryToken: () => 'm-3',
      loadOlderMessages: async () => {
        loadCalls += 1
      }
    })

    const found = await navigator.loadUntilMessage('m-1', {
      maxRequests: 10,
      maxStagnantRequests: 2
    })

    expect(found).toBe(false)
    expect(loadCalls).toBe(2)
  })
})
