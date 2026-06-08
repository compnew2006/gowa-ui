package campaigninteractive

import (
	"encoding/json"
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

	options := campaign.PollOptions.Strings()

	if len(recipientMsgIDs) == 0 {
		return &PollVotesResponse{
			Question: campaign.PollQuestion,
			Options:  options,
			Total:    0,
			Results:  zeroResults(options),
		}, nil
	}

	type voteRow struct {
		InteractiveData string `gorm:"column:interactive_data"`
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

	optionSet := make(map[string]bool, len(options))
	for _, opt := range options {
		optionSet[opt] = true
	}

	counts := make(map[string]int64, len(options))
	var total int64
	for _, v := range votes {
		var data struct {
			SelectedOptions []string `json:"selected_options"`
		}
		if err := json.Unmarshal([]byte(v.InteractiveData), &data); err != nil {
			continue
		}
		for _, sel := range data.SelectedOptions {
			if optionSet[sel] {
				counts[sel]++
				total++
			}
		}
	}

	results := make([]PollVote, 0, len(options))
	for _, opt := range options {
		results = append(results, PollVote{
			Option:     opt,
			Count:      counts[opt],
			Percentage: percentage(counts[opt], total),
		})
	}

	return &PollVotesResponse{
		Question: campaign.PollQuestion,
		Options:  options,
		Total:    total,
		Results:  results,
	}, nil
}


func percentage(count, total int64) string {
	if total > 0 {
		return fmt.Sprintf("%.1f%%", float64(count)/float64(total)*100)
	}
	return "0.0%"
}

func zeroResults(options []string) []PollVote {
	results := make([]PollVote, len(options))
	for i, opt := range options {
		results[i] = PollVote{Option: opt, Percentage: "0.0%"}
	}
	return results
}
