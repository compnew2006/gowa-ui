package handlers

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// sanitizeComponentsWithPayload — pure function, no dependencies
// ---------------------------------------------------------------------------

func TestSanitizeComponentsWithPayload_NilEmpty_ReturnsEmpty(t *testing.T) {
	result := sanitizeComponentsWithPayload(nil, nil, nil)
	assert.Empty(t, result)

	result = sanitizeComponentsWithPayload([]interface{}{}, nil, nil)
	assert.Empty(t, result)
}

func TestSanitizeComponentsWithPayload_NonMapChildren_PassThrough(t *testing.T) {
	children := []interface{}{"hello", 42, true, nil}
	result := sanitizeComponentsWithPayload(children, nil, nil)
	require.Len(t, result, 4)
	assert.Equal(t, "hello", result[0])
	assert.Equal(t, 42, result[1])
	assert.Equal(t, true, result[2])
	assert.Nil(t, result[3])
}

func TestSanitizeComponentsWithPayload_ComponentsWithoutID_Stripped(t *testing.T) {
	for compType := range componentsWithoutID {
		t.Run(compType, func(t *testing.T) {
			children := []interface{}{
				map[string]interface{}{
					"type": compType,
					"id":   "remove_me",
					"name": "field",
				},
			}
			result := sanitizeComponentsWithPayload(children, nil, nil)
			comp := result[0].(map[string]interface{})
			_, hasID := comp["id"]
			assert.False(t, hasID, "type %s should have id removed", compType)
		})
	}
}

func TestSanitizeComponentsWithPayload_UnknownComponentType_KeepsID(t *testing.T) {
	children := []interface{}{
		map[string]interface{}{
			"type": "CustomWidget",
			"id":   "keep_me",
		},
	}
	result := sanitizeComponentsWithPayload(children, nil, nil)
	comp := result[0].(map[string]interface{})
	assert.Equal(t, "keep_me", comp["id"])
}

func TestSanitizeComponentsWithPayload_NameWithDigits_Sanitized(t *testing.T) {
	children := []interface{}{
		map[string]interface{}{
			"type": "TextInput",
			"name": "field123",
		},
	}
	result := sanitizeComponentsWithPayload(children, nil, nil)
	comp := result[0].(map[string]interface{})
	sanitized := comp["name"].(string)
	assert.NotEqual(t, "field123", sanitized, "digits should be replaced")
	assert.Equal(t, sanitizeID("field123"), sanitized)
}

func TestSanitizeComponentsWithPayload_NameAlreadyValid_Unchanged(t *testing.T) {
	children := []interface{}{
		map[string]interface{}{
			"type": "TextInput",
			"name": "email_field",
		},
	}
	result := sanitizeComponentsWithPayload(children, nil, nil)
	comp := result[0].(map[string]interface{})
	assert.Equal(t, "email_field", comp["name"])
}

func TestSanitizeComponentsWithPayload_DataSourceOptionIDs_Sanitized(t *testing.T) {
	children := []interface{}{
		map[string]interface{}{
			"type": "Dropdown",
			"name": "picker",
			"data-source": []interface{}{
				map[string]interface{}{"id": "opt_1", "label": "A"},
				map[string]interface{}{"id": "opt_2", "label": "B"},
				"non-map-entry",
			},
		},
	}
	result := sanitizeComponentsWithPayload(children, nil, nil)
	comp := result[0].(map[string]interface{})
	ds := comp["data-source"].([]interface{})
	require.Len(t, ds, 3)
	opt0 := ds[0].(map[string]interface{})
	assert.Equal(t, sanitizeID("opt_1"), opt0["id"])
	assert.Equal(t, "non-map-entry", ds[2])
}

func TestSanitizeComponentsWithPayload_CompleteAction_PayloadMapped(t *testing.T) {
	children := []interface{}{
		map[string]interface{}{
			"type": "TextInput",
			"name": "email",
			"on-click-action": map[string]interface{}{
				"name": "complete",
			},
		},
	}
	allFields := []string{"name_field", "email"}
	prevFields := []string{"name_field"}

	result := sanitizeComponentsWithPayload(children, allFields, prevFields)
	comp := result[0].(map[string]interface{})
	action := comp["on-click-action"].(map[string]interface{})
	payload := action["payload"].(map[string]interface{})

	assert.Equal(t, "${data.name_field}", payload["name_field"], "previous screen field uses data ref")
	assert.Equal(t, "${form.email}", payload["email"], "current screen field uses form ref")
}

func TestSanitizeComponentsWithPayload_NavigateAction_PayloadMapped(t *testing.T) {
	children := []interface{}{
		map[string]interface{}{
			"type": "TextInput",
			"name": "email",
			"on-click-action": map[string]interface{}{
				"name": "navigate",
			},
		},
	}
	prevFields := []string{"name_field"}

	result := sanitizeComponentsWithPayload(children, nil, prevFields)
	comp := result[0].(map[string]interface{})
	action := comp["on-click-action"].(map[string]interface{})
	payload := action["payload"].(map[string]interface{})

	assert.Equal(t, "${data.name_field}", payload["name_field"])
	assert.Equal(t, "${form.email}", payload["email"])
}

func TestSanitizeComponentsWithPayload_NavigateAction_NoCurrentFields_NoPayload(t *testing.T) {
	children := []interface{}{
		map[string]interface{}{
			"type": "TextHeading",
			"on-click-action": map[string]interface{}{
				"name": "navigate",
			},
		},
	}
	result := sanitizeComponentsWithPayload(children, nil, nil)
	comp := result[0].(map[string]interface{})
	action := comp["on-click-action"].(map[string]interface{})
	_, hasPayload := action["payload"]
	assert.False(t, hasPayload, "navigate with no current screen fields should not add payload")
}

func TestSanitizeComponentsWithPayload_DoesNotMutateOriginal(t *testing.T) {
	original := map[string]interface{}{
		"type": "TextInput",
		"name": "my_field",
		"id":   "orig_id",
	}
	children := []interface{}{original}

	_ = sanitizeComponentsWithPayload(children, nil, nil)

	assert.Equal(t, "my_field", original["name"])
	assert.Equal(t, "orig_id", original["id"])
}

func TestSanitizeComponentsWithPayload_PreservesOtherActionFields(t *testing.T) {
	children := []interface{}{
		map[string]interface{}{
			"type": "TextInput",
			"name": "email",
			"on-click-action": map[string]interface{}{
				"name":    "navigate",
				"next":    "SCREEN_B",
				"customk": "customv",
			},
		},
	}
	result := sanitizeComponentsWithPayload(children, nil, nil)
	comp := result[0].(map[string]interface{})
	action := comp["on-click-action"].(map[string]interface{})
	assert.Equal(t, "SCREEN_B", action["next"])
	assert.Equal(t, "customv", action["customk"])
}

// ---------------------------------------------------------------------------
// parseIncomingMessagePayload — pure branches (text/interactive/location/contacts)
// ---------------------------------------------------------------------------

func mustUnmarshalMsg(t *testing.T, raw string) IncomingTextMessage {
	t.Helper()
	var msg IncomingTextMessage
	require.NoError(t, json.Unmarshal([]byte(raw), &msg))
	return msg
}

func TestParseIncomingMessagePayload_TextMessage(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	msg := mustUnmarshalMsg(t, `{"type":"text","text":{"body":"Hello"}}`)
	payload := app.parseIncomingMessagePayload(&models.WhatsAppAccount{}, msg)
	assert.Equal(t, "Hello", payload.MessageText)
	assert.Equal(t, "text", payload.MessageType)
}

func TestParseIncomingMessagePayload_TextNilBody_EmptyText(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	msg := IncomingTextMessage{Type: "text"}
	payload := app.parseIncomingMessagePayload(&models.WhatsAppAccount{}, msg)
	assert.Equal(t, "", payload.MessageText)
}

func TestParseIncomingMessagePayload_ButtonReply(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	msg := mustUnmarshalMsg(t, `{
		"type":"interactive",
		"interactive":{"button_reply":{"id":"btn_1","title":"Yes"}}
	}`)
	payload := app.parseIncomingMessagePayload(&models.WhatsAppAccount{}, msg)
	assert.Equal(t, "Yes", payload.MessageText)
	assert.Equal(t, "button_reply", payload.MessageType)
	assert.Equal(t, "btn_1", payload.ButtonID)
}

func TestParseIncomingMessagePayload_ListReply(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	msg := mustUnmarshalMsg(t, `{
		"type":"interactive",
		"interactive":{"list_reply":{"id":"list_1","title":"Option A","description":"desc"}}
	}`)
	payload := app.parseIncomingMessagePayload(&models.WhatsAppAccount{}, msg)
	assert.Equal(t, "Option A", payload.MessageText)
	assert.Equal(t, "button_reply", payload.MessageType)
	assert.Equal(t, "list_1", payload.ButtonID)
}

func TestParseIncomingMessagePayload_NFMReply_ValidJSON(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	msg := mustUnmarshalMsg(t, `{
		"type":"interactive",
		"interactive":{"nfm_reply":{"body":"Flow done","response_json":"{\"name\":\"John\",\"age\":30}"}}
	}`)
	payload := app.parseIncomingMessagePayload(&models.WhatsAppAccount{}, msg)
	assert.Equal(t, "Flow done", payload.MessageText)
	assert.Equal(t, "nfm_reply", payload.MessageType)
	require.NotNil(t, payload.FlowResponseData)
	assert.Equal(t, "John", payload.FlowResponseData["name"])
}

func TestParseIncomingMessagePayload_NFMReply_InvalidJSON_NoPanic(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	msg := mustUnmarshalMsg(t, `{
		"type":"interactive",
		"interactive":{"nfm_reply":{"body":"done","response_json":"{invalid}"}}
	}`)
	payload := app.parseIncomingMessagePayload(&models.WhatsAppAccount{}, msg)
	assert.Equal(t, "done", payload.MessageText)
	assert.Equal(t, "nfm_reply", payload.MessageType)
	assert.Nil(t, payload.FlowResponseData)
}

func TestParseIncomingMessagePayload_InteractiveNil_NoChange(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	msg := IncomingTextMessage{Type: "interactive"}
	payload := app.parseIncomingMessagePayload(&models.WhatsAppAccount{}, msg)
	assert.Equal(t, "interactive", payload.MessageType)
	assert.Equal(t, "", payload.MessageText)
}

func TestParseIncomingMessagePayload_Location_Full(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	msg := mustUnmarshalMsg(t, `{
		"type":"location",
		"location":{"latitude":1.23,"longitude":4.56,"name":"HQ","address":"123 St"}
	}`)
	payload := app.parseIncomingMessagePayload(&models.WhatsAppAccount{}, msg)
	var loc map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(payload.MessageText), &loc))
	assert.Equal(t, 1.23, loc["latitude"])
	assert.Equal(t, 4.56, loc["longitude"])
	assert.Equal(t, "HQ", loc["name"])
	assert.Equal(t, "123 St", loc["address"])
}

func TestParseIncomingMessagePayload_Location_Minimal(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	msg := mustUnmarshalMsg(t, `{
		"type":"location",
		"location":{"latitude":0,"longitude":0}
	}`)
	payload := app.parseIncomingMessagePayload(&models.WhatsAppAccount{}, msg)
	var loc map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(payload.MessageText), &loc))
	assert.Equal(t, 0.0, loc["latitude"])
	assert.Equal(t, 0.0, loc["longitude"])
	_, hasName := loc["name"]
	assert.False(t, hasName)
}

func TestParseIncomingMessagePayload_Contacts(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	msg := mustUnmarshalMsg(t, `{
		"type":"contacts",
		"contacts":[
			{"name":{"formatted_name":"Alice"},"phones":[{"phone":"+123"},{"phone":"+456"}]},
			{"name":{"formatted_name":"Bob"}}
		]
	}`)
	payload := app.parseIncomingMessagePayload(&models.WhatsAppAccount{}, msg)
	var contacts []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(payload.MessageText), &contacts))
	require.Len(t, contacts, 2)
	assert.Equal(t, "Alice", contacts[0]["name"])
	phones := contacts[0]["phones"].([]interface{})
	assert.Equal(t, "+123", phones[0])
	assert.Equal(t, "+456", phones[1])
	assert.Equal(t, "Bob", contacts[1]["name"])
}

func TestParseIncomingMessagePayload_UnknownType_NoPanic(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	msg := IncomingTextMessage{Type: "reaction"}
	payload := app.parseIncomingMessagePayload(&models.WhatsAppAccount{}, msg)
	assert.Equal(t, "reaction", payload.MessageType)
	assert.Equal(t, "", payload.MessageText)
}

func TestParseIncomingMessagePayload_Context_SetsReplyToWAMID(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	msg := mustUnmarshalMsg(t, `{
		"type":"text",
		"text":{"body":"reply"},
		"context":{"id":"wamid_abc123","from":"+123"}
	}`)
	payload := app.parseIncomingMessagePayload(&models.WhatsAppAccount{}, msg)
	assert.Equal(t, "wamid_abc123", payload.ReplyToWAMID)
}

func TestParseIncomingMessagePayload_NoContext_EmptyReplyToWAMID(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	msg := mustUnmarshalMsg(t, `{"type":"text","text":{"body":"hello"}}`)
	payload := app.parseIncomingMessagePayload(&models.WhatsAppAccount{}, msg)
	assert.Equal(t, "", payload.ReplyToWAMID)
}

// ---------------------------------------------------------------------------
// enforceStrictSendRestrictions — guard clauses only (main logic needs DB)
// ---------------------------------------------------------------------------

func TestEnforceStrictSendRestrictions_NilApp_ReturnsNil(t *testing.T) {
	err := (*App)(nil).enforceStrictSendRestrictions(
		context.Background(),
		OutgoingMessageRequest{Contact: &models.Contact{}},
		MessageSendOptions{},
	)
	assert.NoError(t, err)
}

func TestEnforceStrictSendRestrictions_NilDB_ReturnsNil(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	err := app.enforceStrictSendRestrictions(
		context.Background(),
		OutgoingMessageRequest{Contact: &models.Contact{}},
		MessageSendOptions{},
	)
	assert.NoError(t, err)
}

func TestEnforceStrictSendRestrictions_NilContact_ReturnsNil(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	err := app.enforceStrictSendRestrictions(
		context.Background(),
		OutgoingMessageRequest{},
		MessageSendOptions{},
	)
	assert.NoError(t, err)
}

func TestEnforceStrictSendRestrictions_GroupContactViaPhoneNumber_ReturnsNil(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	contact := &models.Contact{
		PhoneNumber:    "123456789@g.us",
		OrganizationID: uuid.New(),
	}
	err := app.enforceStrictSendRestrictions(
		context.Background(),
		OutgoingMessageRequest{Contact: contact},
		MessageSendOptions{},
	)
	assert.NoError(t, err)
}

func TestEnforceStrictSendRestrictions_GroupContactViaMetadata_ReturnsNil(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	contact := &models.Contact{
		PhoneNumber:    "+1234567890",
		OrganizationID: uuid.New(),
		Metadata:       models.JSONB{"is_group_chat": true},
	}
	err := app.enforceStrictSendRestrictions(
		context.Background(),
		OutgoingMessageRequest{Contact: contact},
		MessageSendOptions{},
	)
	assert.NoError(t, err)
}

func TestEnforceStrictSendRestrictions_ChannelContactViaPhoneNumber_ReturnsNil(t *testing.T) {
	app := &App{Log: testutil.NopLogger()}
	contact := &models.Contact{
		PhoneNumber:    "12345@newsletter",
		OrganizationID: uuid.New(),
	}
	err := app.enforceStrictSendRestrictions(
		context.Background(),
		OutgoingMessageRequest{Contact: contact},
		MessageSendOptions{},
	)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// buildMessagesResponse — data transformation, no DB for non-reply messages
// ---------------------------------------------------------------------------

func newMinimalApp() *App {
	return &App{Log: testutil.NopLogger()}
}

func makeTextMessage(overrides ...func(*models.Message)) models.Message {
	now := time.Now()
	m := models.Message{
		BaseModel: models.BaseModel{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		ContactID:         uuid.New(),
		ConversationID:    "conv_1",
		OrganizationID:    uuid.New(),
		Direction:         models.DirectionIncoming,
		MessageType:       models.MessageTypeText,
		Content:           "Hello",
		Status:            models.MessageStatusDelivered,
		WhatsAppMessageID: "wamid_123",
	}
	for _, fn := range overrides {
		fn(&m)
	}
	return m
}

func TestBuildMessagesResponse_EmptyList_ReturnsEmpty(t *testing.T) {
	app := newMinimalApp()
	result := app.buildMessagesResponse([]models.Message{}, false)
	assert.Empty(t, result)
}

func TestBuildMessagesResponse_TextMessage_CorrectFields(t *testing.T) {
	app := newMinimalApp()
	msg := makeTextMessage()
	result := app.buildMessagesResponse([]models.Message{msg}, false)
	require.Len(t, result, 1)
	r := result[0]
	assert.Equal(t, msg.ID, r.ID)
	assert.Equal(t, msg.ContactID, r.ContactID)
	assert.Equal(t, msg.Direction, r.Direction)
	assert.Equal(t, msg.MessageType, r.MessageType)
	assert.Equal(t, msg.Status, r.Status)
	assert.Equal(t, msg.WhatsAppMessageID, r.WAMID)
	content, ok := r.Content.(map[string]string)
	require.True(t, ok, "content should be map[string]string")
	assert.Equal(t, "Hello", content["body"])
}

func TestBuildMessagesResponse_MultipleMessages(t *testing.T) {
	app := newMinimalApp()
	msgs := []models.Message{
		makeTextMessage(func(m *models.Message) { m.Content = "First" }),
		makeTextMessage(func(m *models.Message) { m.Content = "Second"; m.ID = uuid.New(); m.WhatsAppMessageID = "wamid_456" }),
	}
	result := app.buildMessagesResponse(msgs, false)
	require.Len(t, result, 2)
	assert.Equal(t, "First", result[0].Content.(map[string]string)["body"])
	assert.Equal(t, "Second", result[1].Content.(map[string]string)["body"])
}

func TestBuildMessagesResponse_MediaMessage_HasMediaFields(t *testing.T) {
	app := newMinimalApp()
	msg := makeTextMessage(func(m *models.Message) {
		m.MessageType = models.MessageTypeImage
		m.Content = "check this"
		m.MediaURL = "/media/img.png"
		m.MediaMimeType = "image/png"
		m.MediaFilename = "img.png"
	})
	result := app.buildMessagesResponse([]models.Message{msg}, false)
	require.Len(t, result, 1)
	assert.Equal(t, "/media/img.png", result[0].MediaURL)
	assert.Equal(t, "image/png", result[0].MediaMimeType)
	assert.Equal(t, "img.png", result[0].MediaFilename)
}

func TestBuildMessagesResponse_NonMediaMessage_NoMediaFields(t *testing.T) {
	app := newMinimalApp()
	msg := makeTextMessage()
	result := app.buildMessagesResponse([]models.Message{msg}, false)
	require.Len(t, result, 1)
	assert.Equal(t, "", result[0].MediaURL)
	assert.Equal(t, "", result[0].MediaMimeType)
}

func TestBuildMessagesResponse_InstanceID_Populated(t *testing.T) {
	app := newMinimalApp()
	instID := uuid.New()
	msg := makeTextMessage(func(m *models.Message) {
		m.InstanceID = &instID
	})
	result := app.buildMessagesResponse([]models.Message{msg}, false)
	require.Len(t, result, 1)
	require.NotNil(t, result[0].InstanceID)
	assert.Equal(t, instID.String(), *result[0].InstanceID)
}

func TestBuildMessagesResponse_NoInstanceID_Nil(t *testing.T) {
	app := newMinimalApp()
	msg := makeTextMessage()
	result := app.buildMessagesResponse([]models.Message{msg}, false)
	require.Len(t, result, 1)
	assert.Nil(t, result[0].InstanceID)
}

func TestBuildMessagesResponse_ReplyWithPreloadedMessage(t *testing.T) {
	app := newMinimalApp()
	replyID := uuid.New()
	replyMsg := makeTextMessage(func(m *models.Message) {
		m.Content = "original"
		m.Direction = models.DirectionIncoming
	})
	msg := makeTextMessage(func(m *models.Message) {
		m.IsReply = true
		m.ReplyToMessageID = &replyID
		m.ReplyToMessage = &replyMsg
	})
	result := app.buildMessagesResponse([]models.Message{msg}, false)
	require.Len(t, result, 1)
	assert.True(t, result[0].IsReply)
	require.NotNil(t, result[0].ReplyToMessageID)
	assert.Equal(t, replyID.String(), *result[0].ReplyToMessageID)
	require.NotNil(t, result[0].ReplyToMessage)
	assert.Equal(t, "original", result[0].ReplyToMessage.Content.(map[string]string)["body"])
}

func TestBuildMessagesResponse_DeletedContent_Normalized(t *testing.T) {
	app := newMinimalApp()
	msg := makeTextMessage(func(m *models.Message) {
		m.Content = "This message was deleted"
	})
	result := app.buildMessagesResponse([]models.Message{msg}, false)
	require.Len(t, result, 1)
	body := result[0].Content.(map[string]string)["body"]
	assert.Equal(t, "(This message was deleted)", body)
}
