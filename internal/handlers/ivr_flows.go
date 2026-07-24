package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"gorm.io/gorm"
)

// IVRFlowRequest represents the request body for creating/updating an IVR flow
type IVRFlowRequest struct {
	WhatsAppAccount string       `json:"whatsapp_account"`
	Name            string       `json:"name"`
	Description     string       `json:"description"`
	IsActive        bool         `json:"is_active"`
	IsCallStart     bool         `json:"is_call_start"`
	IsOutgoingEnd   bool         `json:"is_outgoing_end"`
	Menu            models.JSONB `json:"menu"`
	WelcomeAudioURL string       `json:"welcome_audio_url"`
}

// ListIVRFlows returns all IVR flows for the organization

// GetIVRFlow returns a single IVR flow by ID

// CreateIVRFlow creates a new IVR flow

// UpdateIVRFlow updates an existing IVR flow

// diffIVRMenuNodes compares old and new IVR menu JSONB to find node-level changes
func diffIVRMenuNodes(db *gorm.DB, oldMenu, newMenu models.JSONB) []map[string]any {
	var changes []map[string]any

	type ivrNode struct {
		ID     string         `json:"id"`
		Type   string         `json:"type"`
		Label  string         `json:"label"`
		Config map[string]any `json:"config"`
	}

	extractNodes := func(menu models.JSONB) map[string]ivrNode {
		result := make(map[string]ivrNode)
		nodesRaw, ok := menu["nodes"]
		if !ok {
			return result
		}
		nodesSlice, ok := nodesRaw.([]any)
		if !ok {
			return result
		}
		for _, raw := range nodesSlice {
			m, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			b, err := json.Marshal(m)
			if err != nil {
				continue
			}
			var n ivrNode
			_ = json.Unmarshal(b, &n)
			if n.ID != "" {
				result[n.ID] = n
			}
		}
		return result
	}

	oldNodes := extractNodes(oldMenu)
	newNodes := extractNodes(newMenu)

	// Detect added nodes
	for id, n := range newNodes {
		if _, exists := oldNodes[id]; !exists {
			changes = append(changes, map[string]any{
				"field": "node_added", "old_value": nil, "new_value": n.Label + " (" + n.Type + ")",
			})
		}
	}

	// Detect removed nodes
	for id, n := range oldNodes {
		if _, exists := newNodes[id]; !exists {
			changes = append(changes, map[string]any{
				"field": "node_removed", "old_value": n.Label + " (" + n.Type + ")", "new_value": nil,
			})
		}
	}

	// Detect modified nodes
	for id, newN := range newNodes {
		oldN, exists := oldNodes[id]
		if !exists {
			continue
		}
		if oldN.Label != newN.Label {
			changes = append(changes, map[string]any{
				"field": newN.Label + " → label", "old_value": oldN.Label, "new_value": newN.Label,
			})
		}
		// Compare config fields — drill into nested maps for readable diffs
		label := newN.Label
		if label == "" {
			label = id
		}
		for key, newVal := range newN.Config {
			oldVal := oldN.Config[key]
			oldJSON, _ := json.Marshal(oldVal)
			newJSON, _ := json.Marshal(newVal)
			if string(oldJSON) == string(newJSON) {
				continue
			}
			// Try to diff nested maps (e.g. options: {"1": {"label": "Sales"}})
			oldMap, oldIsMap := oldVal.(map[string]any)
			newMap, newIsMap := newVal.(map[string]any)
			if oldIsMap && newIsMap {
				for subKey, subNew := range newMap {
					subOld := oldMap[subKey]
					sOldJSON, _ := json.Marshal(subOld)
					sNewJSON, _ := json.Marshal(subNew)
					if string(sOldJSON) != string(sNewJSON) {
						// Extract readable value from nested object
						oldLabel := extractLabel(subOld)
						newLabel := extractLabel(subNew)
						changes = append(changes, map[string]any{
							"field": label + " → " + key + "[" + subKey + "]", "old_value": oldLabel, "new_value": newLabel,
						})
					}
				}
				// Check for removed keys
				for subKey, subOld := range oldMap {
					if _, exists := newMap[subKey]; !exists {
						changes = append(changes, map[string]any{
							"field": label + " → " + key + "[" + subKey + "]", "old_value": extractLabel(subOld), "new_value": nil,
						})
					}
				}
			} else {
				displayOld := oldVal
				displayNew := newVal
				// Resolve team_id UUIDs to team names
				if key == "team_id" {
					displayOld = resolveTeamName(db, fmt.Sprintf("%v", oldVal))
					displayNew = resolveTeamName(db, fmt.Sprintf("%v", newVal))
				}
				changes = append(changes, map[string]any{
					"field": label + " → " + key, "old_value": displayOld, "new_value": displayNew,
				})
			}
		}
	}

	return changes
}

func resolveTeamName(db *gorm.DB, teamID string) string {
	if teamID == "" || teamID == "<nil>" {
		return "—"
	}
	var name string
	db.Model(&models.Team{}).Where("id = ?", teamID).Pluck("name", &name)
	if name == "" {
		return teamID
	}
	return name
}

// extractLabel returns a readable string from a value — if it's a map with a "label" key, return that
func extractLabel(val any) any {
	if m, ok := val.(map[string]any); ok {
		if label, exists := m["label"]; exists {
			return label
		}
	}
	return val
}

// DeleteIVRFlow soft-deletes an IVR flow

// getAudioDir returns the configured audio directory path.
func (a *App) getAudioDir() string {
	dir := a.Config.Calling.AudioDir
	if dir == "" {
		dir = "./audio"
	}
	return dir
}

// UploadIVRAudio handles multipart audio file uploads for IVR greetings.

// ServeIVRAudio serves audio files from the IVR audio directory.

// UploadOrgAudio handles multipart audio file uploads for org-level hold music and ringback tones.
// The "type" query parameter must be "hold_music" or "ringback".
func (a *App) UploadOrgAudio(r *fastglue.Request) error {
	orgID, _, err := a.requireAuth(r, models.ResourceOrganizations, models.ActionWrite)
	if err != nil {
		return nil
	}

	audioType := string(r.RequestCtx.QueryArgs().Peek("type"))
	if audioType != "hold_music" && audioType != "ringback" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Query parameter 'type' must be 'hold_music' or 'ringback'", nil, "")
	}

	// Parse multipart form
	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid multipart form: "+err.Error(), nil, "")
	}

	files := form.File["file"]
	if len(files) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "No file provided", nil, "")
	}

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Failed to open file", nil, "")
	}
	defer func() { _ = file.Close() }()

	// Read file content (limit to 5MB)
	const maxAudioSize = 5 << 20
	data, err := io.ReadAll(io.LimitReader(file, maxAudioSize+1))
	if err != nil {
		a.Log.Error("Failed to read org audio file", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to read file", nil, "")
	}
	if len(data) > maxAudioSize {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "File too large. Maximum size is 5MB", nil, "")
	}

	// Validate MIME type
	mimeType := fileHeader.Header.Get("Content-Type")
	allowedAudio := map[string]bool{
		"audio/ogg": true, "audio/opus": true,
		"audio/mpeg": true, "audio/mp3": true,
		"audio/wav": true, "audio/x-wav": true, "audio/wave": true,
		"application/ogg": true, "application/octet-stream": true,
		"video/ogg": true,
	}
	if !allowedAudio[mimeType] {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Unsupported audio type: "+mimeType, nil, "")
	}

	// Ensure audio directory exists
	audioDir := a.getAudioDir()
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		a.Log.Error("Failed to create org audio directory", "error", err, "audio_dir", audioDir)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create audio directory", nil, "")
	}

	// Save uploaded file to a temp location for transcoding
	tmpInput, err := os.CreateTemp("", "org-audio-input-*")
	if err != nil {
		a.Log.Error("Failed to create org temp file", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to create temp file", nil, "")
	}
	defer func() { _ = os.Remove(tmpInput.Name()) }()

	if _, err := tmpInput.Write(data); err != nil {
		_ = tmpInput.Close()
		a.Log.Error("Failed to write org temp file", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to write temp file", nil, "")
	}
	_ = tmpInput.Close()

	// Transcode to OGG/Opus 48kHz mono using ffmpeg
	filename := fmt.Sprintf("org_%s_%s.ogg", orgID.String(), audioType)
	filePath := filepath.Join(audioDir, filename)

	if err := transcodeToOpus(tmpInput.Name(), filePath); err != nil {
		a.Log.Error("Audio transcoding failed", "error", err, "org_id", orgID, "type", audioType)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to transcode audio to Opus format", nil, "")
	}

	// Update org settings with the new filename
	var org models.Organization
	if err := a.DB.Where("id = ?", orgID).First(&org).Error; err != nil {
		a.Log.Error("Failed to load organization for audio update", "error", err, "org_id", orgID)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load organization", nil, "")
	}
	if org.Settings == nil {
		org.Settings = models.JSONB{}
	}
	settingsKey := audioType + "_file"
	org.Settings[settingsKey] = filename
	if err := a.DB.Save(&org).Error; err != nil {
		a.Log.Error("Failed to update organization audio settings", "error", err, "org_id", orgID, "audio_type", audioType)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to update organization settings", nil, "")
	}

	a.Log.Info("Org audio uploaded", "org_id", orgID, "type", audioType, "filename", filename, "size", len(data))

	return r.SendEnvelope(map[string]any{
		"filename":  filename,
		"type":      audioType,
		"mime_type": mimeType,
		"size":      len(data),
	})
}

// transcodeToOpus converts any audio file to OGG/Opus 48kHz mono using ffmpeg.
// This ensures the file is compatible with the WebRTC AudioPlayer.
func transcodeToOpus(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg",
		"-y",            // overwrite output
		"-i", inputPath, // input file
		"-ac", "1", // mono
		"-ar", "48000", // 48kHz (Opus standard)
		"-c:a", "libopus",
		"-b:a", "48k", // bitrate
		"-application", "audio",
		"-frame_duration", "20", // 20ms frames (matches RTP packetization)
		"-vn", // strip video/cover art
		outputPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}

// generateIVRAudio iterates the flat v2 nodes array and generates TTS audio
// for any node with a non-empty "greeting_text" in its config. The generated
// audio filename is set as the node's "audio_file" config field.
func (a *App) generateIVRAudio(menu models.JSONB) error {
	nodesRaw, ok := menu["nodes"]
	if !ok {
		return nil
	}

	// toSlice may produce a copy (via re-marshal), so we always write
	// the potentially-modified slice back into menu["nodes"].
	nodesSlice, ok := toSlice(nodesRaw)
	if !ok {
		return nil
	}

	for i, nodeRaw := range nodesSlice {
		nodeMap, ok := nodeRaw.(map[string]any)
		if !ok {
			continue
		}
		configRaw, ok := nodeMap["config"]
		if !ok {
			continue
		}
		config, ok := configRaw.(map[string]any)
		if !ok {
			continue
		}
		greetingText, _ := config["greeting_text"].(string)
		if greetingText == "" {
			continue
		}
		filename, err := a.TTS.Generate(greetingText)
		if err != nil {
			return err
		}
		config["audio_file"] = filename
		nodeMap["config"] = config
		nodesSlice[i] = nodeMap
	}

	// Write the modified nodes back so changes reach the DB.
	menu["nodes"] = nodesSlice
	return nil
}

// menuHasGreetingText checks if any node in the v2 flow graph uses greeting_text.
func menuHasGreetingText(menu models.JSONB) bool {
	nodesRaw, ok := menu["nodes"]
	if !ok {
		return false
	}
	nodesSlice, ok := toSlice(nodesRaw)
	if !ok {
		return false
	}

	for _, nodeRaw := range nodesSlice {
		nodeMap, ok := nodeRaw.(map[string]any)
		if !ok {
			continue
		}
		configRaw, ok := nodeMap["config"]
		if !ok {
			continue
		}
		config, ok := configRaw.(map[string]any)
		if !ok {
			continue
		}
		if text, _ := config["greeting_text"].(string); text != "" {
			return true
		}
	}
	return false
}

// toSlice converts an any to []any, handling JSON re-marshal if needed.
func toSlice(v any) ([]any, bool) {
	if s, ok := v.([]any); ok {
		return s, true
	}
	// Handle case where JSONB was deserialized differently
	b, err := json.Marshal(v)
	if err != nil {
		return nil, false
	}
	var s []any
	if json.Unmarshal(b, &s) == nil {
		return s, true
	}
	return nil, false
}

// validateFlowGraph validates a v2 IVR flow graph for structural correctness.
func validateFlowGraph(menu models.JSONB) error {
	versionRaw := menu["version"]
	var version int
	switch v := versionRaw.(type) {
	case float64:
		version = int(v)
	case int:
		version = v
	}
	if version != 2 {
		return fmt.Errorf("unsupported flow version: %v (expected 2)", versionRaw)
	}

	nodesRaw, ok := menu["nodes"]
	if !ok {
		return fmt.Errorf("missing nodes array")
	}
	nodesSlice, ok := toSlice(nodesRaw)
	if !ok {
		return fmt.Errorf("nodes must be an array")
	}

	// Empty flow (no nodes yet) is valid
	if len(nodesSlice) == 0 {
		return nil
	}

	entryNode, _ := menu["entry_node"].(string)
	if entryNode == "" {
		return fmt.Errorf("missing entry_node (required when nodes exist)")
	}

	// Build node ID set and terminal node set
	nodeIDs := make(map[string]bool, len(nodesSlice))
	terminalNodes := make(map[string]bool)
	terminalTypes := map[string]bool{"goto_flow": true, "hangup": true}

	for _, nodeRaw := range nodesSlice {
		nodeMap, ok := nodeRaw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := nodeMap["id"].(string)
		if id == "" {
			return fmt.Errorf("node missing id")
		}
		if nodeIDs[id] {
			return fmt.Errorf("duplicate node id: %s", id)
		}
		nodeIDs[id] = true

		nodeType, _ := nodeMap["type"].(string)
		if terminalTypes[nodeType] {
			terminalNodes[id] = true
		}
	}

	if !nodeIDs[entryNode] {
		return fmt.Errorf("entry_node %q does not reference a valid node", entryNode)
	}

	// Validate edges
	edgesRaw := menu["edges"]
	if edgesRaw != nil {
		edgesSlice, ok := toSlice(edgesRaw)
		if !ok {
			return fmt.Errorf("edges must be an array")
		}
		for _, edgeRaw := range edgesSlice {
			edgeMap, ok := edgeRaw.(map[string]any)
			if !ok {
				continue
			}
			from, _ := edgeMap["from"].(string)
			to, _ := edgeMap["to"].(string)
			if !nodeIDs[from] {
				return fmt.Errorf("edge from %q references non-existent node", from)
			}
			if !nodeIDs[to] {
				return fmt.Errorf("edge to %q references non-existent node", to)
			}
			if terminalNodes[from] {
				return fmt.Errorf("terminal node %q must not have outgoing edges", from)
			}
		}
	}

	return nil
}
