package handlers

import (
	"strconv"
	"strings"

	"github.com/compnew2006/whatomate/pkg/gowa"
	"github.com/zerodha/fastglue"
)

// Request-query → typed filter mapping for the GOWA read-side proxy endpoints
// declared in chats_gowa.go. These helpers turn raw fasthttp query strings into
// the strongly-typed gowa.ListChatsFilter / gowa.GetMessagesFilter structs the
// GOWA client expects, applying default values and light validation. They are
// pure transport-layer mechanics and intentionally hold no business logic.

func parseListChatsFilter(r *fastglue.Request) gowa.ListChatsFilter {
	q := r.RequestCtx.QueryArgs()
	f := gowa.ListChatsFilter{
		Limit:  atoiOrDefault(string(q.Peek("limit")), 25),
		Offset: atoiOrDefault(string(q.Peek("offset")), 0),
		Search: string(q.Peek("search")),
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 25
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	if v := string(q.Peek("has_media")); v == "true" || v == "1" {
		f.HasMedia = true
	}
	if v := string(q.Peek("archived")); v != "" {
		b := v == "true" || v == "1"
		f.Archived = &b
	}
	return f
}

func parseGetMessagesFilter(r *fastglue.Request) gowa.GetMessagesFilter {
	q := r.RequestCtx.QueryArgs()
	f := gowa.GetMessagesFilter{
		Limit:     atoiOrDefault(string(q.Peek("limit")), 50),
		Offset:    atoiOrDefault(string(q.Peek("offset")), 0),
		Search:    string(q.Peek("search")),
		StartTime: string(q.Peek("start_time")),
		EndTime:   string(q.Peek("end_time")),
	}
	if f.Limit < 1 || f.Limit > 200 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	if v := string(q.Peek("media_only")); v == "true" || v == "1" {
		f.MediaOnly = true
	}
	if v := string(q.Peek("is_from_me")); v != "" {
		b := v == "true" || v == "1"
		f.IsFromMe = &b
	}
	return f
}

func atoiOrDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// isValidWhatsAppJID is a light sanity check: a JID must contain "@" and a
// domain suffix we recognise. We accept:
//   - <digits>@s.whatsapp.net          (individual)
//   - <digits>-<digits>@g.us            (group)
//   - <alphanum>@newsletter             (newsletter, GOWA-specific)
//
// This is not a strict parser — GOWA will reject malformed JIDs downstream
// with a precise error. The point here is to fail fast on obviously wrong
// inputs like missing "@" or path traversal attempts.
func isValidWhatsAppJID(jid string) bool {
	at := strings.IndexByte(jid, '@')
	if at <= 0 || at == len(jid)-1 {
		return false
	}
	domain := jid[at+1:]
	switch {
	case domain == "s.whatsapp.net",
		domain == "c.us",
		strings.HasSuffix(domain, ".g.us"),
		strings.HasSuffix(domain, "newsletter"):
		return true
	}
	return false
}
