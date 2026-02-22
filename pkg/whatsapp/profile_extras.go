package whatsapp

import (
	"context"
)

// UploadProfilePicture uploads a profile picture and returns the handle
// It uses the Resumable Upload API as required for profile pictures
// UploadProfilePicture uploads a profile picture and returns the handle
// It uses the Resumable Upload API as required for profile pictures
func (c *Client) UploadProfilePicture(ctx context.Context, account *Account, fileData []byte, mimeType string) (string, error) {
	// Profile pictures use the resumable upload API
	// Note: ResumableUpload signature: (ctx, account, data, mimeType, filename)
	return c.ResumableUpload(ctx, account, fileData, mimeType, "profile_picture")
}
