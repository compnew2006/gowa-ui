package whatsmeow

import (
	crand "crypto/rand"
	"math/big"
	"time"
)

func secureRandomInt63n(n int64) int64 {
	if n <= 1 {
		return 0
	}

	max := big.NewInt(n)
	value, err := crand.Int(crand.Reader, max)
	if err != nil {
		// Fallback keeps behavior deterministic enough for jitter/backoff while avoiding panic.
		return time.Now().UnixNano() % n
	}

	return value.Int64()
}

func secureRandomIntn(n int) int {
	if n <= 1 {
		return 0
	}
	return int(secureRandomInt63n(int64(n)))
}
