export interface AuditFormatOptions {
  // When set, objects rendered as JSON are truncated to this many characters
  // with an ellipsis. AuditLogPanel uses 120; the detail view shows them full.
  maxObjectChars?: number
}

// useAuditFormat exposes the shared audit-log value formatter used by both the
// compact AuditLogPanel timeline and the full AuditLogDetailView. The only
// behavioural difference between the two call sites is whether long JSON
// objects are truncated, so that is driven by maxObjectChars.
export function useAuditFormat(options: AuditFormatOptions = {}) {
  function formatValue(val: any): string {
    if (val === null || val === undefined || val === '') return '—'
    if (typeof val === 'boolean') return val ? 'Yes' : 'No'
    if (Array.isArray(val)) {
      if (val.length === 0) return '—'
      // Format array of objects (e.g. buttons) as readable text
      if (typeof val[0] === 'object' && val[0] !== null) {
        return val.map(item => item.text || item.name || item.title || JSON.stringify(item)).join(', ')
      }
      return val.join(', ') || '—'
    }
    if (typeof val === 'object') {
      // For simple objects with a "body" key (like response_content), show the body
      if (val.body) return String(val.body)
      const s = JSON.stringify(val)
      if (options.maxObjectChars !== undefined && s.length > options.maxObjectChars) {
        return s.slice(0, options.maxObjectChars) + '…'
      }
      return s
    }
    return String(val)
  }

  return { formatValue }
}
