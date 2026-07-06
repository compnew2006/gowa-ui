package whatsmeow

import (
	"fmt"

	"github.com/compnew2006/whatomate/internal/models"
)

const (
	// InstanceSettingAutoDownloadIncomingMedia toggles background media prefetch for incoming messages.
	InstanceSettingAutoDownloadIncomingMedia = "auto_download_incoming_media"
)

// IsAutoDownloadIncomingMediaEnabled returns whether incoming media prefetch is enabled in instance settings.
// Missing/invalid values default to false.
func IsAutoDownloadIncomingMediaEnabled(settings models.JSONB) bool {
	return boolSetting(settings, InstanceSettingAutoDownloadIncomingMedia, false)
}

func injectAutoDownloadIncomingMediaDefault(settings models.JSONB) models.JSONB {
	normalized := cloneSettings(settings)
	if _, ok := normalized[InstanceSettingAutoDownloadIncomingMedia]; !ok {
		normalized[InstanceSettingAutoDownloadIncomingMedia] = false
	}
	return normalized
}

func ValidateAutoDownloadIncomingMediaSetting(raw any) error {
	switch raw.(type) {
	case nil:
		return nil
	case bool, string, int, int64, float64:
		// Allowed primitive types; parsing happens via boolSetting fallback semantics.
		return nil
	default:
		return fmt.Errorf("auto download incoming media must be a boolean value")
	}
}
