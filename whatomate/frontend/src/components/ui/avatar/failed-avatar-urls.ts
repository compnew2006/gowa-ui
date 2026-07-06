const MAX_FAILED_AVATAR_URLS = 1000
const failedAvatarURLs = new Set<string>()

export function hasFailedAvatarURL(url: string): boolean {
  return failedAvatarURLs.has(url)
}

export function markFailedAvatarURL(url: string): void {
  if (url === '') return
  if (failedAvatarURLs.has(url)) return

  if (failedAvatarURLs.size >= MAX_FAILED_AVATAR_URLS) {
    const oldestURL = failedAvatarURLs.values().next().value
    if (typeof oldestURL === 'string') {
      failedAvatarURLs.delete(oldestURL)
    }
  }

  failedAvatarURLs.add(url)
}
