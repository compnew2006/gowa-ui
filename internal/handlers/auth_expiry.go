package handlers

import "time"

const minAccessTokenTTLSeconds = 1

// nextAccessTokenExpiry returns the next midnight in the same location as now.
func nextAccessTokenExpiry(now time.Time) time.Time {
	year, month, day := now.Date()
	return time.Date(year, month, day+1, 0, 0, 0, 0, now.Location())
}

func accessTokenTTLSeconds(now time.Time, expiresAt time.Time) int {
	ttlSeconds := int(expiresAt.Sub(now).Seconds())
	if ttlSeconds < minAccessTokenTTLSeconds {
		return minAccessTokenTTLSeconds
	}
	return ttlSeconds
}
