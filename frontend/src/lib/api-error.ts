export function basicAuthHeader(username?: string, password?: string): string {
  if (!username || !password) return ''
  try {
    return `Basic ${btoa(`${username}:${password}`)}`
  } catch {
    return ''
  }
}
