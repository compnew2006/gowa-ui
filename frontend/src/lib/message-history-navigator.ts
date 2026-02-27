export interface MessageHistoryNavigatorDependencies {
  hasMessage: (messageId: string) => boolean
  hasMoreMessages: () => boolean
  loadOlderMessages: () => Promise<void>
  getBoundaryToken?: () => string | null
}

export interface MessageHistoryNavigatorOptions {
  maxRequests?: number
  maxStagnantRequests?: number
}

const DEFAULT_MAX_REQUESTS = 64
const DEFAULT_MAX_STAGNANT_REQUESTS = 2

/**
 * Loads older chat history in bounded loops until a target message is available.
 */
export class MessageHistoryNavigator {
  private readonly dependencies: MessageHistoryNavigatorDependencies

  constructor(dependencies: MessageHistoryNavigatorDependencies) {
    this.dependencies = dependencies
  }

  async loadUntilMessage(
    messageId: string,
    options: MessageHistoryNavigatorOptions = {}
  ): Promise<boolean> {
    if (!messageId) {
      return false
    }

    if (this.dependencies.hasMessage(messageId)) {
      return true
    }

    const maxRequests = Math.max(1, options.maxRequests ?? DEFAULT_MAX_REQUESTS)
    const maxStagnantRequests = Math.max(
      1,
      options.maxStagnantRequests ?? DEFAULT_MAX_STAGNANT_REQUESTS
    )

    const hasBoundaryToken = typeof this.dependencies.getBoundaryToken === 'function'
    let previousBoundary = hasBoundaryToken
      ? this.dependencies.getBoundaryToken?.() ?? null
      : null
    let stagnantRequests = 0

    for (let requestCount = 0; requestCount < maxRequests; requestCount += 1) {
      if (!this.dependencies.hasMoreMessages()) {
        break
      }

      await this.dependencies.loadOlderMessages()

      if (this.dependencies.hasMessage(messageId)) {
        return true
      }

      if (!hasBoundaryToken) {
        continue
      }

      const nextBoundary = this.dependencies.getBoundaryToken?.() ?? null
      if (nextBoundary === previousBoundary) {
        stagnantRequests += 1
        if (stagnantRequests >= maxStagnantRequests) {
          break
        }
      } else {
        stagnantRequests = 0
      }
      previousBoundary = nextBoundary
    }

    return this.dependencies.hasMessage(messageId)
  }
}
