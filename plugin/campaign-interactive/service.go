package campaigninteractive

import (
	"fmt"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (p *Plugin) getPollVotes(db *gorm.DB, campaignID uuid.UUID) (*PollVotesResponse, error) {
	var campaign models.BulkMessageCampaign
	if err := db.Where("id = ?", campaignID).First(&campaign).Error; err != nil {
		return nil, fmt.Errorf("campaign not found: %w", err)
	}
	if campaign.PollQuestion == "" {
		return nil, nil
	}

	var recipientMsgIDs []string
	if err := db.Model(&models.BulkMessageRecipient{}).
		Where("campaign_id = ? AND whats_app_message_id != ''", campaignID).
		Pluck("whats_app_message_id", &recipientMsgIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to query recipients: %w", err)
	}

	if len(recipientMsgIDs) == 0 {
		options := campaign.PollOptions.Strings()
		return &PollVotesResponse{
			Question: campaign.PollQuestion,
			Options:  options,
			Total:    0,
			Results:  zeroResults(options),
		}, nil
	}

	options := campaign.PollOptions.Strings()

	type voteRow struct {
		Content   string `gorm:"column:content"`
		CreatedAt string `gorm:"column:created_at"`
	}
	var votes []voteRow
	if err := db.Table("messages").
		Where("organization_id = ? AND message_type = ? AND interactive_data->>'type' = ? AND reply_to_message_id IN (?)",
			campaign.OrganizationID, models.MessageTypePoll, "poll_vote",
			db.Table("messages").
				Select("id").
				Where("organization_id = ? AND message_type = ? AND whats_app_message_id IN ?",
					campaign.OrganizationID, models.MessageTypePoll, recipientMsgIDs),
		).
		Find(&votes).Error; err != nil {
		return nil, fmt.Errorf("failed to query votes: %w", err)
	}

	counts := make(map[string]int64, len(options))
	for _, v := range votes {
		counts[v.Content]++
	}

	var total int64
	results := make([]PollVote, 0, len(options))
	for _, opt := range options {
		c := counts[opt]
		total += c
		results = append(results, PollVote{
			Option: opt,
			Count:  c,
		})
	}
	for i := range results {
		if total > 0 {
			results[i].Percentage = fmt.Sprintf("%.1f%%", float64(results[i].Count)/float64(total)*100)
		} else {
			results[i].Percentage = "0.0%"
		}
	}

	return &PollVotesResponse{
		Question: campaign.PollQuestion,
		Options:  options,
		Total:    total,
		Results:  results,
	}, nil
}

func zeroResults(options []string) []PollVote {
	results := make([]PollVote, len(options))
	for i, opt := range options {
		results[i] = PollVote{Option: opt, Percentage: "0.0%"}
	}
	return results
}
