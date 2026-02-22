import { contactsService, type ActivityLog } from '@/services/api'
import { extractContactIDFromPath, isUUID } from './activity-log-route-utils'

function normalizeLabelPart(value: unknown): string {
  if (typeof value !== 'string') return ''
  return value.replace(/\s+/g, ' ').trim()
}

export class ActivityLogContactResolver {
  private readonly labelCache = new Map<string, string>()

  async resolve(logs: ActivityLog[]): Promise<Map<string, string>> {
    const ids = new Set<string>()

    for (const log of logs) {
      if (log.contact_id && isUUID(log.contact_id)) {
        ids.add(log.contact_id)
      }
      const pathID = extractContactIDFromPath(log.path || '')
      if (pathID) {
        ids.add(pathID)
      }
    }

    if (ids.size === 0) {
      return new Map()
    }

    const missing = Array.from(ids).filter((id) => !this.labelCache.has(id))
    await Promise.all(missing.map(async (id) => this.resolveOne(id)))

    const resolved = new Map<string, string>()
    for (const id of ids) {
      const label = this.labelCache.get(id)
      if (label) resolved.set(id, label)
    }
    return resolved
  }

  private async resolveOne(contactID: string): Promise<void> {
    try {
      const response = await contactsService.get(contactID)
      const envelope = (response.data as any)?.data || response.data
      const contact = (envelope?.contact || envelope) as Record<string, unknown>
      const label = this.toContactLabel(contact)
      if (label) {
        this.labelCache.set(contactID, label)
      }
    } catch {
      // Do not fail the activity table if contact resolution fails.
    }
  }

  private toContactLabel(contact: Record<string, unknown>): string {
    const name =
      normalizeLabelPart(contact.profile_name) ||
      normalizeLabelPart(contact.name)
    const phone = normalizeLabelPart(contact.phone_number)

    if (name && phone) return `${name} (${phone})`
    if (name) return name
    if (phone) return phone
    return ''
  }
}
