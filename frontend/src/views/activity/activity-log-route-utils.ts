const uuidPattern =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i

export function isUUID(value: string): boolean {
  return uuidPattern.test(value.trim())
}

export function extractContactIDFromPath(path: string): string {
  const normalizedPath = path.split('?')[0]
  const match = normalizedPath.match(/\/api(?:\/v\d+)?\/(?:contacts|chats)\/([^/]+)/i)
  const candidate = match?.[1]?.trim() || ''
  if (!candidate || !isUUID(candidate)) return ''
  return candidate
}
