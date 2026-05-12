package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeUnclaimedChatAccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		allowView    bool
		allowSend    bool
		expectedView bool
		expectedSend bool
	}{
		{
			name:         "send implies view",
			allowView:    false,
			allowSend:    true,
			expectedView: true,
			expectedSend: true,
		},
		{
			name:         "view only stays view only",
			allowView:    true,
			allowSend:    false,
			expectedView: true,
			expectedSend: false,
		},
		{
			name:         "none stays none",
			allowView:    false,
			allowSend:    false,
			expectedView: false,
			expectedSend: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			view, send := normalizeUnclaimedChatAccess(tc.allowView, tc.allowSend)
			assert.Equal(t, tc.expectedView, view)
			assert.Equal(t, tc.expectedSend, send)
		})
	}
}

func TestIsContactAssignedToUser(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	otherUserID := uuid.New()

	tests := []struct {
		name    string
		contact *models.Contact
		userID  uuid.UUID
		want    bool
	}{
		{
			name:    "nil contact",
			contact: nil,
			userID:  userID,
			want:    false,
		},
		{
			name: "nil assigned user",
			contact: &models.Contact{
				AssignedUserID: nil,
			},
			userID: userID,
			want:   false,
		},
		{
			name: "zero user id",
			contact: &models.Contact{
				AssignedUserID: &userID,
			},
			userID: uuid.Nil,
			want:   false,
		},
		{
			name: "assigned to same user",
			contact: &models.Contact{
				AssignedUserID: &userID,
			},
			userID: userID,
			want:   true,
		},
		{
			name: "assigned to different user",
			contact: &models.Contact{
				AssignedUserID: &otherUserID,
			},
			userID: userID,
			want:   false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, isContactAssignedToUser(tc.contact, tc.userID))
		})
	}
}

func TestShouldAllowSelfAssignedRestrictedInstanceListBypass(t *testing.T) {
	t.Parallel()

	currentUserID := uuid.New()
	otherUserID := uuid.New()
	open := models.ChatStatusOpen
	closed := models.ChatStatusClosed

	tests := []struct {
		name                string
		statusFilter        *models.ChatStatus
		hasAssignedToFilter bool
		assignedToUserID    *uuid.UUID
		want                bool
	}{
		{
			name:                "requires open status",
			statusFilter:        &closed,
			hasAssignedToFilter: true,
			assignedToUserID:    &currentUserID,
			want:                false,
		},
		{
			name:                "requires assigned_to filter",
			statusFilter:        &open,
			hasAssignedToFilter: false,
			assignedToUserID:    &currentUserID,
			want:                false,
		},
		{
			name:                "requires assigned user id",
			statusFilter:        &open,
			hasAssignedToFilter: true,
			assignedToUserID:    nil,
			want:                false,
		},
		{
			name:                "requires same user",
			statusFilter:        &open,
			hasAssignedToFilter: true,
			assignedToUserID:    &otherUserID,
			want:                false,
		},
		{
			name:                "allows open self-assigned filter",
			statusFilter:        &open,
			hasAssignedToFilter: true,
			assignedToUserID:    &currentUserID,
			want:                true,
		},
		{
			name:                "nil status filter",
			statusFilter:        nil,
			hasAssignedToFilter: true,
			assignedToUserID:    &currentUserID,
			want:                false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := shouldAllowSelfAssignedRestrictedInstanceListBypass(
				tc.statusFilter,
				tc.hasAssignedToFilter,
				tc.assignedToUserID,
				currentUserID,
			)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestApplyRestrictedInstanceVisibilityFilter_NilQuery(t *testing.T) {
	t.Parallel()

	instanceIDs := []uuid.UUID{uuid.New()}
	result := applyRestrictedInstanceVisibilityFilter(nil, instanceIDs, uuid.Nil)
	assert.Nil(t, result)
}

func TestApplyRestrictedInstanceVisibilityFilter_EmptyRestrictedInstances(t *testing.T) {
	t.Parallel()

	result := applyRestrictedInstanceVisibilityFilter(nil, nil, uuid.Nil)
	assert.Nil(t, result)
}
