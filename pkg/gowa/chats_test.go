package gowa_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPagedChatsServer returns a mock GOWA server that serves GET /chats in
// pages of `pageSize`. It reports total=`totalChats` so ListChats must page
// until it has collected them all. The recorded requests let the test assert
// the offset progression.
func newPagedChatsServer(t *testing.T, totalChats, pageSize int) (*httptest.Server, *[]string) {
	t.Helper()
	paths := []string{}
	var pathsPtr = &paths
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		pathsPtr = &paths

		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

		// Build the page slice.
		end := offset + limit
		if end > totalChats {
			end = totalChats
		}
		var data []map[string]any
		for i := offset; i < end; i++ {
			data = append(data, map[string]any{
				"jid":               "1650555" + strconv.Itoa(1000+i) + "@s.whatsapp.net",
				"name":              "User " + strconv.Itoa(i),
				"last_message_time": "2026-07-15T10:30:00Z",
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    "SUCCESS",
			"message": "Success get chat list",
			"results": map[string]any{
				"data": data,
				"pagination": map[string]any{
					"limit":  pageSize,
					"offset": offset,
					"total":  totalChats,
				},
			},
		})
	}))
	return server, pathsPtr
}

func TestListChats_PagesUntilTotalReached(t *testing.T) {
	t.Parallel()
	const total, pageSize = 25, 10 // expect 3 pages: 10, 10, 5
	server, paths := newPagedChatsServer(t, total, 10)
	defer server.Close()

	c := gowa.New(server.URL, "", "")
	chats, reportedTotal, err := c.ListChats(context.Background(), "dev1", gowa.ListChatsOptions{Limit: pageSize})
	require.NoError(t, err)

	require.Len(t, chats, total, "all chats across pages must be aggregated")
	assert.Equal(t, total, reportedTotal)
	// 3 paginated requests. The first omits offset=0 (GOWA's default); the
	// subsequent two carry offset=10 and offset=20.
	require.Len(t, *paths, 3)
	assert.NotContains(t, (*paths)[0], "offset=", "first page omits the default offset=0")
	assert.Contains(t, (*paths)[1], "offset=10")
	assert.Contains(t, (*paths)[2], "offset=20")
}

func TestListChats_SinglePageWhenUnderLimit(t *testing.T) {
	t.Parallel()
	server, paths := newPagedChatsServer(t, 3, 10)
	defer server.Close()

	c := gowa.New(server.URL, "", "")
	chats, total, err := c.ListChats(context.Background(), "dev1", gowa.ListChatsOptions{})
	require.NoError(t, err)

	// A short page (< pageSize) must terminate the loop after one request.
	require.Len(t, chats, 3)
	assert.Equal(t, 3, total)
	require.Len(t, *paths, 1)
}

func TestListChats_SendsDeviceHeaderAndQuery(t *testing.T) {
	t.Parallel()
	var (
		gotDevice string
		gotSearch string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotDevice = r.Header.Get("X-Device-Id")
		gotSearch = r.URL.Query().Get("search")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":{"data":[{"jid":"1@s.whatsapp.net","name":"A"}],"pagination":{"total":1,"limit":25,"offset":0}}}`))
	}))
	defer server.Close()

	c := gowa.New(server.URL, "", "")
	_, _, err := c.ListChats(context.Background(), "dev-xyz", gowa.ListChatsOptions{Search: "alice"})
	require.NoError(t, err)
	assert.Equal(t, "dev-xyz", gotDevice)
	assert.Equal(t, "alice", gotSearch)
}

func TestListChats_GroupAndRegularJIDsPreserved(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":{"data":[
			{"jid":"16505551234@s.whatsapp.net","name":"Alice"},
			{"jid":"120363abc@g.us","name":"Team Group"}
		],"pagination":{"total":2,"limit":25,"offset":0}}}`))
	}))
	defer server.Close()

	c := gowa.New(server.URL, "", "")
	chats, _, err := c.ListChats(context.Background(), "dev1", gowa.ListChatsOptions{})
	require.NoError(t, err)
	require.Len(t, chats, 2)
	assert.Equal(t, "16505551234@s.whatsapp.net", chats[0].JID)
	assert.Equal(t, "Team Group", chats[1].Name)
}

func TestListChats_HasMediaFilter(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "true", r.URL.Query().Get("has_media"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":{"data":[{"jid":"1@s.whatsapp.net","name":"A"}],"pagination":{"total":1,"limit":25,"offset":0}}}`))
	}))
	defer server.Close()

	c := gowa.New(server.URL, "", "")
	mediaOnly := true
	_, _, err := c.ListChats(context.Background(), "dev1", gowa.ListChatsOptions{HasMedia: &mediaOnly})
	require.NoError(t, err)
}

func TestGetChatMessages_SendsFilterQueryParams(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "true", q.Get("media_only"))
		assert.Equal(t, "false", q.Get("is_from_me"))
		assert.Equal(t, "2026-08-01T00:00:00Z", q.Get("start_time"))
		assert.Equal(t, "2026-08-30T00:00:00Z", q.Get("end_time"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":{"data":[{"id":"m1"}],"pagination":{"total":1,"limit":50,"offset":0}}}`))
	}))
	defer server.Close()

	c := gowa.New(server.URL, "", "")
	fromMe := false
	msgs, _, err := c.GetChatMessages(context.Background(), "dev1", "628123@s.whatsapp.net", gowa.ChatMessagesOptions{
		Limit: 1, MediaOnly: true, IsFromMe: &fromMe,
		StartTime: "2026-08-01T00:00:00Z", EndTime: "2026-08-30T00:00:00Z",
	})
	require.NoError(t, err)
	require.Len(t, msgs, 1)
}
