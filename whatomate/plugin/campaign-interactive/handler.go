package campaigninteractive

import (
	"net/http"

	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/google/uuid"
	"github.com/zerodha/fastglue"
)

// PollVote represents a single poll vote aggregated result.
type PollVote struct {
	Option     string `json:"option"`
	Count      int64  `json:"count"`
	Percentage string `json:"percentage"`
}

// PollVoteDetail represents an individual vote for detailed listing.
type PollVoteDetail struct {
	PhoneNumber string `json:"phone_number"`
	Selected    string `json:"selected_options"`
	VotedAt     string `json:"voted_at"`
}

// PollVotesResponse is the API response for poll vote analytics.
type PollVotesResponse struct {
	Question string           `json:"question"`
	Options  []string         `json:"options"`
	Total    int64            `json:"total_votes"`
	Results  []PollVote       `json:"results"`
	Votes    []PollVoteDetail `json:"votes,omitempty"`
}

func (p *Plugin) handleGetPollVotes(rc *fastglue.Request) error {
	orgID, ok := middleware.GetOrganizationID(rc)
	if !ok {
		return rc.SendErrorEnvelope(http.StatusUnauthorized, "missing organization", nil, "UNAUTHORIZED")
	}

	campaignIDStr := rc.RequestCtx.UserValue("id").(string)
	if campaignIDStr == "" {
		return rc.SendErrorEnvelope(http.StatusBadRequest, "campaign id required", nil, "VALIDATION_ERROR")
	}
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		return rc.SendErrorEnvelope(http.StatusBadRequest, "invalid campaign id", nil, "VALIDATION_ERROR")
	}

	scopedDB := tenant.ScopedDB(p.DB, orgID)

	resp, err := p.getPollVotes(scopedDB, campaignID)
	if err != nil {
		p.Log.Error("handleGetPollVotes: failed", "err", err, "campaign_id", campaignID)
		return rc.SendErrorEnvelope(http.StatusInternalServerError, "failed to get poll votes", nil, "INTERNAL_ERROR")
	}
	if resp == nil {
		return rc.SendErrorEnvelope(http.StatusNotFound, "campaign has no poll", nil, "NOT_FOUND")
	}

	return rc.SendEnvelope(resp)
}
