package handlers

import "time"

const minAccessTokenTTLSeconds = 1

const defaultAccessTokenExpiryMinutes = 15

// nextAccessTokenExpiry returns expiry relative to now based on configured TTL minutes.
func nextAccessTokenExpiry(now time.Time, ttlMinutes int) time.Time {
	if ttlMinutes <= 0 {
		ttlMinutes = defaultAccessTokenExpiryMinutes
	}
	return now.Add(time.Duration(ttlMinutes) * time.Minute)
}

func accessTokenTTLSeconds(now time.Time, expiresAt time.Time) int {
	ttlSeconds := int(expiresAt.Sub(now).Seconds())
	if ttlSeconds < minAccessTokenTTLSeconds {
		return minAccessTokenTTLSeconds
	}
	return ttlSeconds
}
