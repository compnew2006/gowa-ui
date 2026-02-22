import { formatDate } from '@/lib/utils'
import type { ActivityLog } from '@/services/api'
import { extractContactIDFromPath } from './activity-log-route-utils'

export type ActivityEventClass = 'user' | 'auth' | 'system' | 'custom' | 'general'

export interface ActivityNarrative {
  sentence: string
  eventClass: ActivityEventClass
}

export interface ActivityNarratorContext {
  contactLabels?: Map<string, string>
}

export class ActivityLogNarrator {
  build(log: ActivityLog, context: ActivityNarratorContext = {}): ActivityNarrative {
    const eventClass = this.classify(log)
    return {
      eventClass,
      sentence: this.describe(log, eventClass, context)
    }
  }

  private describe(log: ActivityLog, eventClass: ActivityEventClass, context: ActivityNarratorContext): string {
    const timestamp = this.formatTimestamp(log.created_at)
    const actor = this.resolveActor(log)

    if (this.isMessageEvent(log)) {
      return this.describeMessageEvent(log, actor, timestamp, context)
    }

    if (log.event_type === 'auth.login') {
      return `${actor} signed in at ${timestamp}`
    }

    if (log.event_type === 'auth.login_failed') {
      const email = this.readMetadataText(log, 'email')
      const reason = this.readMetadataText(log, 'reason')
      if (email && reason) {
        return `Login failed for ${email} (${reason}) at ${timestamp}`
      }
      if (email) {
        return `Login failed for ${email} at ${timestamp}`
      }
      return `Login failed at ${timestamp}`
    }

    if (log.event_type === 'auth.logout') {
      return `${actor} signed out at ${timestamp}`
    }

    if (log.event_type === 'system.api_interaction') {
      return `${actor} ${this.describeSystemAction(log, context)} at ${timestamp}`
    }

    if (eventClass === 'custom') {
      return `${actor} ran custom event ${log.event_type} (${log.action}) at ${timestamp}`
    }

    const action = this.normalizeText(log.action) || this.normalizeText(log.event_type) || 'performed an action'
    return `${actor} ${action.replace(/_/g, ' ')} at ${timestamp}`
  }

  private describeMessageEvent(
    log: ActivityLog,
    actor: string,
    timestamp: string,
    context: ActivityNarratorContext
  ): string {
    const message = this.resolveMessageText(log)
    const chat = this.resolveChatLabel(log, context)
    if (message === 'a message') {
      return `${actor} sent a message to chat ${chat} at ${timestamp}`
    }
    return `${actor} sent message "${message}" to chat ${chat} at ${timestamp}`
  }

  private classify(log: ActivityLog): ActivityEventClass {
    if (this.isMessageEvent(log)) return 'user'
    if (log.category === 'auth') return 'auth'
    if (log.category === 'system') return 'system'
    if (log.category === 'custom') return 'custom'
    return 'general'
  }

  private isMessageEvent(log: ActivityLog): boolean {
    if (log.event_type === 'engagement.conversation_response' || log.action === 'send_message') {
      return true
    }
    const path = (log.path || '').toLowerCase()
    const method = (log.method || '').toUpperCase()
    return method === 'POST' && /\/api(?:\/v\d+)?\/(contacts|chats)\/[^/]+\/messages(?:\/|$)/.test(path)
  }

  private resolveActor(log: ActivityLog): string {
    const actorFromMetadata =
      this.readMetadataText(log, 'actor_name') ||
      this.readMetadataText(log, 'user_name') ||
      this.readMetadataText(log, 'email')
    if (actorFromMetadata) {
      return actorFromMetadata
    }
    return 'you'
  }

  private resolveMessageText(log: ActivityLog): string {
    const rawText =
      this.readMetadataText(log, 'message_content') ||
      this.readMetadataText(log, 'content') ||
      this.readMetadataText(log, 'message')

    if (rawText) {
      return rawText.replace(/"/g, '\\"')
    }

    const messageType = this.readMetadataText(log, 'message_type')
    if (messageType) {
      return `[${messageType}]`
    }
    return 'a message'
  }

  private resolveChatLabel(log: ActivityLog, context: ActivityNarratorContext): string {
    const chatName =
      this.readMetadataText(log, 'chat_name') ||
      this.readMetadataText(log, 'contact_name')
    const chatPhone =
      this.readMetadataText(log, 'chat_phone') ||
      this.readMetadataText(log, 'contact_phone')

    if (chatName && chatPhone) return `${chatName} ${chatPhone}`
    if (chatName) return chatName
    if (chatPhone) return chatPhone
    return this.resolveContactLabel(log, context)
  }

  private describeSystemAction(log: ActivityLog, context: ActivityNarratorContext): string {
    const method = (log.method || '').toUpperCase()
    const route = this.abstractRoute(log.path || '')
    const key = `${method} ${route}`
    const contactReference = this.resolveContactLabel(log, context)

    const scopedRouteAction = this.describeScopedRouteAction(method, route, contactReference)
    if (scopedRouteAction) {
      return this.withResultSuffix(log, scopedRouteAction)
    }

    const knownActions: Record<string, string> = {
      'GET /api/contacts': 'viewed the contacts list',
      'GET /api/notifications': 'checked notifications',
      'GET /api/me/organizations': 'checked available organizations',
      'GET /api/me': 'opened account profile',
      'GET /api/widgets/data-sources': 'loaded dashboard data sources',
      'GET /api/widgets/data': 'loaded dashboard analytics',
      'GET /api/widgets': 'opened dashboard widgets',
      'GET /api/auth/ws-token': 'refreshed the real-time connection token',
      'GET /api/chats': 'viewed the chat queue'
    }

    const known = knownActions[key]
    if (known) {
      return this.withResultSuffix(log, known)
    }

    const resource = this.humanizeRouteResource(route)
    if (method === 'GET') return this.withResultSuffix(log, `viewed ${resource}`)
    if (method === 'POST') return this.withResultSuffix(log, `created ${resource}`)
    if (method === 'PUT' || method === 'PATCH') return this.withResultSuffix(log, `updated ${resource}`)
    if (method === 'DELETE') return this.withResultSuffix(log, `deleted ${resource}`)

    return this.withResultSuffix(log, 'performed a system action')
  }

  private describeScopedRouteAction(method: string, route: string, contactReference: string): string {
    const contactTail = this.getRouteTail(route, '/api/contacts/:id')
    if (contactTail) {
      return this.describeContactScopedAction(method, contactTail, contactReference)
    }

    const chatTail = this.getRouteTail(route, '/api/chats/:id')
    if (chatTail) {
      return this.describeChatScopedAction(method, chatTail, contactReference)
    }

    return ''
  }

  private getRouteTail(route: string, prefix: string): string[] | null {
    if (route === prefix) return []
    if (!route.startsWith(`${prefix}/`)) return null
    return route.slice(prefix.length + 1).split('/').filter(Boolean)
  }

  private describeContactScopedAction(method: string, tail: string[], contactReference: string): string {
    if (method === 'GET' && tail.length === 0) {
      return `opened contact profile for ${contactReference}`
    }

    const scope = tail[0] || 'details'
    if (method === 'GET' && (scope === 'session-data' || scope === 'session')) {
      return `viewed session summary for ${contactReference}`
    }
    if (method === 'GET' && scope === 'messages') {
      return `viewed conversation with ${contactReference}`
    }
    if (method === 'POST' && scope === 'messages') {
      return `sent a message to ${contactReference}`
    }

    const scopeLabel = this.humanizeRouteScope(scope)
    if (method === 'GET') return `viewed ${scopeLabel} for ${contactReference}`
    if (method === 'POST') return `created ${scopeLabel} for ${contactReference}`
    if (method === 'PUT' || method === 'PATCH') return `updated ${scopeLabel} for ${contactReference}`
    if (method === 'DELETE') return `deleted ${scopeLabel} for ${contactReference}`
    return ''
  }

  private describeChatScopedAction(method: string, tail: string[], contactReference: string): string {
    if (method === 'GET' && tail.length === 0) {
      return `opened chat ${contactReference}`
    }

    const scope = tail[0] || 'details'
    if (method === 'PUT' && scope === 'claim') {
      return `claimed chat ${contactReference}`
    }
    if (method === 'PUT' && scope === 'close') {
      return `closed chat ${contactReference}`
    }
    if (method === 'PUT' && scope === 'reopen') {
      return `reopened chat ${contactReference}`
    }
    if (method === 'GET' && scope === 'messages') {
      return `viewed conversation with ${contactReference}`
    }
    if (method === 'POST' && scope === 'messages') {
      return `sent a message to ${contactReference}`
    }

    const scopeLabel = this.humanizeRouteScope(scope)
    if (method === 'GET') return `viewed ${scopeLabel} for chat ${contactReference}`
    if (method === 'POST') return `created ${scopeLabel} for chat ${contactReference}`
    if (method === 'PUT' || method === 'PATCH') return `updated ${scopeLabel} for chat ${contactReference}`
    if (method === 'DELETE') return `deleted ${scopeLabel} for chat ${contactReference}`
    return ''
  }

  private withResultSuffix(log: ActivityLog, baseText: string): string {
    const statusCode = this.readMetadataNumber(log, 'http_status')
    if (statusCode >= 400 || log.status === 'failure') {
      return `${baseText}, but it failed`
    }
    return baseText
  }

  private abstractRoute(path: string): string {
    const route = path.split('?')[0].trim().toLowerCase().replace(/\/+$/, '')
    if (!route) return ''
    const normalized: string[] = []
    for (const segment of route.split('/')) {
      if (!segment) continue
      const lastSegment = normalized[normalized.length - 1]
      if ((segment === 'v1' || segment === 'v2') && lastSegment === 'api') {
        continue
      }
      if (/^[0-9a-f]{8}-[0-9a-f-]{27}$/i.test(segment)) {
        normalized.push(':id')
        continue
      }
      if (/^\d+$/.test(segment)) {
        normalized.push(':id')
        continue
      }
      if (/^[A-Za-z0-9_-]{16,}$/.test(segment)) {
        normalized.push(':id')
        continue
      }
      normalized.push(segment)
    }
    if (normalized.length === 0) return ''
    return `/${normalized.join('/')}`
  }

  private humanizeRouteResource(route: string): string {
    if (!route) return 'system data'
    const segments = route.split('/').filter(Boolean)
    const apiIndex = segments.indexOf('api')
    const resourceSegment =
      apiIndex >= 0 && segments.length > apiIndex + 1 ? segments[apiIndex + 1] : segments[0]

    const cleaned = resourceSegment
      .replace(/-/g, ' ')
      .replace(/_/g, ' ')
      .replace(/\b\w/g, (char) => char.toLowerCase())

    if (!cleaned) return 'system data'
    if (cleaned.endsWith('s')) return `the ${cleaned}`
    return `a ${cleaned}`
  }

  private humanizeRouteScope(scope: string): string {
    const cleaned = scope.replace(/-/g, ' ').replace(/_/g, ' ').trim()
    if (!cleaned) return 'details'
    return cleaned
  }

  private resolveContactLabel(log: ActivityLog, context: ActivityNarratorContext): string {
    const contactID = this.resolveContactID(log)
    if (!contactID) return 'a contact'
    const label = context.contactLabels?.get(contactID)
    if (!label) return `contact ${contactID}`
    const normalized = this.normalizeText(label)
    if (!normalized) {
      return `contact ${contactID}`
    }
    return normalized
  }

  private resolveContactID(log: ActivityLog): string {
    const directContactID = this.normalizeText(log.contact_id || '')
    if (directContactID) return directContactID
    return extractContactIDFromPath(log.path || '')
  }

  private readMetadataText(log: ActivityLog, key: string): string {
    const value = log.metadata?.[key]
    return this.normalizeText(typeof value === 'string' ? value : '')
  }

  private readMetadataNumber(log: ActivityLog, key: string): number {
    const value = log.metadata?.[key]
    if (typeof value === 'number') return value
    if (typeof value === 'string') {
      const parsed = Number(value)
      return Number.isFinite(parsed) ? parsed : 0
    }
    return 0
  }

  private normalizeText(value: string): string {
    const cleaned = value.replace(/\s+/g, ' ').trim()
    if (!cleaned) return ''
    return cleaned.length > 140 ? `${cleaned.slice(0, 137)}...` : cleaned
  }

  private formatTimestamp(value: string): string {
    return formatDate(value, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  }
}
