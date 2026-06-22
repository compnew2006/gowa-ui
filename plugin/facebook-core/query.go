package facebookcore

import (
	"strconv"
	"strings"

	"github.com/zerodha/fastglue"
)

func ParseIntQuery(
	r *fastglue.Request,
	name string,
	fallback int,
	minimum int,
	maximum int,
) (int, error) {
	raw := strings.TrimSpace(string(r.RequestCtx.QueryArgs().Peek(name)))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if value < minimum {
		value = minimum
	}
	if value > maximum {
		value = maximum
	}
	return value, nil
}
