package whatsmeow

import (
	"context"
	"fmt"
	"strings"

	"github.com/compnew2006/whatomate/pkg/provider"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// Ensure WhatsmeowAdapter implements GroupProvider, JoinGroupProvider, GroupParticipantProvider, and GroupInfoProvider.
var (
	_ provider.GroupProvider            = (*WhatsmeowAdapter)(nil)
	_ provider.JoinGroupProvider        = (*WhatsmeowAdapter)(nil)
	_ provider.GroupParticipantProvider = (*WhatsmeowAdapter)(nil)
	_ provider.GroupInfoProvider        = (*WhatsmeowAdapter)(nil)
)

// GetGroups returns all groups the whatsmeow instance has joined.
func (a *WhatsmeowAdapter) GetGroups(ctx context.Context, instanceID string) ([]provider.GroupInfo, error) {
	client, err := a.getClient(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("get groups: %w", err)
	}

	groups, err := client.GetJoinedGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("get joined groups: %w", err)
	}

	result := make([]provider.GroupInfo, len(groups))
	needFullInfo := make([]int, 0)

	for i, g := range groups {
		count := g.ParticipantCount
		if count == 0 && len(g.Participants) > 0 {
			count = len(g.Participants)
		}
		result[i] = provider.GroupInfo{
			JID:              g.JID.String(),
			Name:             g.GroupName.Name,
			ParticipantCount: count,
		}
		if count == 0 {
			needFullInfo = append(needFullInfo, i)
		}
	}

	if len(needFullInfo) > 0 {
		type idxRes struct {
			i     int
			count int
		}
		ch := make(chan idxRes, len(needFullInfo))
		for _, i := range needFullInfo {
			go func(idx int, jid string) {
				parsed, _ := types.ParseJID(jid)
				info, err := client.GetGroupInfo(ctx, parsed)
				if err == nil {
					count := info.ParticipantCount
					if count == 0 {
						count = len(info.Participants)
					}
					ch <- idxRes{i: idx, count: count}
				} else {
					ch <- idxRes{i: idx, count: 0}
				}
			}(i, result[i].JID)
		}
		for range needFullInfo {
			r := <-ch
			result[r.i].ParticipantCount = r.count
		}
	}

	return result, nil
}

// VerifyGroupMembership checks that a group exists and returns its info.
func (a *WhatsmeowAdapter) VerifyGroupMembership(ctx context.Context, instanceID string, groupJID string) (*provider.GroupInfo, error) {
	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return nil, fmt.Errorf("parse group JID %q: %w", groupJID, err)
	}

	client, err := a.getClient(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("verify group: %w", err)
	}

	info, err := client.GetGroupInfo(ctx, jid)
	if err != nil {
		return nil, fmt.Errorf("get group info for %q: %w", groupJID, err)
	}

	return &provider.GroupInfo{
		JID:              info.JID.String(),
		Name:             info.GroupName.Name,
		ParticipantCount: info.ParticipantCount,
	}, nil
}

// GetGroupInfoFromLink fetches group metadata from an invite link without joining.
func (a *WhatsmeowAdapter) GetGroupInfoFromLink(ctx context.Context, instanceID string, inviteLink string) (*provider.GroupInfo, error) {
	client, err := a.getClient(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("get client for link preview: %w", err)
	}

	inviteCode := extractInviteCode(inviteLink)
	if inviteCode == "" {
		return nil, fmt.Errorf("invalid invite link: no code found")
	}

	info, err := client.GetGroupInfoFromLink(ctx, inviteCode)
	if err != nil {
		return nil, fmt.Errorf("get group info from link: %w", err)
	}

	return &provider.GroupInfo{
		JID:              info.JID.String(),
		Name:             info.GroupName.Name,
		ParticipantCount: info.ParticipantCount,
	}, nil
}

// JoinGroupWithLink joins a WhatsApp group using an invite link.
func (a *WhatsmeowAdapter) JoinGroupWithLink(ctx context.Context, instanceID string, inviteLink string) (string, error) {
	client, err := a.getClient(ctx, instanceID)
	if err != nil {
		return "", fmt.Errorf("join group: %w", err)
	}

	inviteLink = strings.TrimSpace(inviteLink)
	if inviteLink == "" {
		return "", fmt.Errorf("invite link is empty")
	}

	inviteCode := extractInviteCode(inviteLink)
	if inviteCode == "" {
		return "", fmt.Errorf("invalid invite link: no code found")
	}

	resp, err := client.JoinGroupWithLink(ctx, inviteCode)
	if err != nil {
		return "", fmt.Errorf("join group with link %q: %w", inviteCode, err)
	}

	return resp.String(), nil
}

// parseParticipantGroup converts whatsmeow GroupParticipant to provider.GroupParticipant.
// It resolves phone numbers from LID when the primary JID is a LID.
func parseParticipantGroup(p types.GroupParticipant, phoneResolver func(string) string) provider.GroupParticipant {
	jid := p.JID.String()
	phoneNumber := ""

	// If JID is a LID, try to resolve the phone number
	if p.JID.Server == types.HiddenUserServer {
		if pn := phoneResolver(jid); pn != "" {
			phoneNumber = pn
		}
	}

	// If we still don't have a phone number and the WhatsApp types has one, use it
	if phoneNumber == "" && p.PhoneNumber.User != "" {
		phoneNumber = p.PhoneNumber.User
	}

	return provider.GroupParticipant{
		JID:          jid,
		PhoneNumber:  phoneNumber,
		IsAdmin:      p.IsAdmin,
		IsSuperAdmin: p.IsSuperAdmin,
	}
}

// AddGroupParticipants adds one or more participants to a group.
func (a *WhatsmeowAdapter) AddGroupParticipants(ctx context.Context, instanceID string, groupJID string, participantJIDs []string) ([]provider.GroupParticipant, error) {
	groupJIDObj, err := types.ParseJID(groupJID)
	if err != nil {
		return nil, fmt.Errorf("parse group JID %q: %w", groupJID, err)
	}

	client, err := a.getClient(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("add group participants: %w", err)
	}

	jids := make([]types.JID, 0, len(participantJIDs))
	for _, raw := range participantJIDs {
		jid, err := types.ParseJID(raw)
		if err != nil {
			return nil, fmt.Errorf("parse participant JID %q: %w", raw, err)
		}
		jids = append(jids, jid)
	}

	participants, err := client.UpdateGroupParticipants(ctx, groupJIDObj, jids, whatsmeow.ParticipantChangeAdd)
	if err != nil {
		return nil, fmt.Errorf("add participants to group %q: %w", groupJID, err)
	}

	result := make([]provider.GroupParticipant, 0, len(participants))
	for _, p := range participants {
		result = append(result, parseParticipantGroup(p, a.phoneResolver(ctx)))
	}
	return result, nil
}

// RemoveGroupParticipants removes one or more participants from a group.
func (a *WhatsmeowAdapter) RemoveGroupParticipants(ctx context.Context, instanceID string, groupJID string, participantJIDs []string) ([]provider.GroupParticipant, error) {
	groupJIDObj, err := types.ParseJID(groupJID)
	if err != nil {
		return nil, fmt.Errorf("parse group JID %q: %w", groupJID, err)
	}

	client, err := a.getClient(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("remove group participants: %w", err)
	}

	jids := make([]types.JID, 0, len(participantJIDs))
	for _, raw := range participantJIDs {
		jid, err := types.ParseJID(raw)
		if err != nil {
			return nil, fmt.Errorf("parse participant JID %q: %w", raw, err)
		}
		jids = append(jids, jid)
	}

	participants, err := client.UpdateGroupParticipants(ctx, groupJIDObj, jids, whatsmeow.ParticipantChangeRemove)
	if err != nil {
		return nil, fmt.Errorf("remove participants from group %q: %w", groupJID, err)
	}

	result := make([]provider.GroupParticipant, 0, len(participants))
	for _, p := range participants {
		result = append(result, parseParticipantGroup(p, a.phoneResolver(ctx)))
	}
	return result, nil
}

// PromoteGroupParticipants promotes participants to group admin.
func (a *WhatsmeowAdapter) PromoteGroupParticipants(ctx context.Context, instanceID string, groupJID string, participantJIDs []string) ([]provider.GroupParticipant, error) {
	groupJIDObj, err := types.ParseJID(groupJID)
	if err != nil {
		return nil, fmt.Errorf("parse group JID %q: %w", groupJID, err)
	}

	client, err := a.getClient(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("promote group participants: %w", err)
	}

	jids := make([]types.JID, 0, len(participantJIDs))
	for _, raw := range participantJIDs {
		jid, err := types.ParseJID(raw)
		if err != nil {
			return nil, fmt.Errorf("parse participant JID %q: %w", raw, err)
		}
		jids = append(jids, jid)
	}

	participants, err := client.UpdateGroupParticipants(ctx, groupJIDObj, jids, whatsmeow.ParticipantChangePromote)
	if err != nil {
		return nil, fmt.Errorf("promote participants in group %q: %w", groupJID, err)
	}

	result := make([]provider.GroupParticipant, 0, len(participants))
	for _, p := range participants {
		result = append(result, parseParticipantGroup(p, a.phoneResolver(ctx)))
	}
	return result, nil
}

// DemoteGroupParticipants demotes participants from group admin.
func (a *WhatsmeowAdapter) DemoteGroupParticipants(ctx context.Context, instanceID string, groupJID string, participantJIDs []string) ([]provider.GroupParticipant, error) {
	groupJIDObj, err := types.ParseJID(groupJID)
	if err != nil {
		return nil, fmt.Errorf("parse group JID %q: %w", groupJID, err)
	}

	client, err := a.getClient(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("demote group participants: %w", err)
	}

	jids := make([]types.JID, 0, len(participantJIDs))
	for _, raw := range participantJIDs {
		jid, err := types.ParseJID(raw)
		if err != nil {
			return nil, fmt.Errorf("parse participant JID %q: %w", raw, err)
		}
		jids = append(jids, jid)
	}

	participants, err := client.UpdateGroupParticipants(ctx, groupJIDObj, jids, whatsmeow.ParticipantChangeDemote)
	if err != nil {
		return nil, fmt.Errorf("demote participants in group %q: %w", groupJID, err)
	}

	result := make([]provider.GroupParticipant, 0, len(participants))
	for _, p := range participants {
		result = append(result, parseParticipantGroup(p, a.phoneResolver(ctx)))
	}
	return result, nil
}

// GetGroupParticipants returns all participants of a group.
func (a *WhatsmeowAdapter) GetGroupParticipants(ctx context.Context, instanceID string, groupJID string) ([]provider.GroupParticipant, error) {
	groupJIDObj, err := types.ParseJID(groupJID)
	if err != nil {
		return nil, fmt.Errorf("parse group JID %q: %w", groupJID, err)
	}

	client, err := a.getClient(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("get group participants: %w", err)
	}

	info, err := client.GetGroupInfo(ctx, groupJIDObj)
	if err != nil {
		return nil, fmt.Errorf("get group info for %q: %w", groupJID, err)
	}

	result := make([]provider.GroupParticipant, 0, len(info.Participants))
	for _, p := range info.Participants {
		result = append(result, parseParticipantGroup(p, func(lid string) string {
			if a.manager != nil {
				return a.manager.lookupPNForLID(ctx, lid)
			}
			return ""
		}))
	}
	return result, nil
}

// extractInviteCode extracts the invite code from a WhatsApp invite link.
func extractInviteCode(link string) string {
	link = strings.TrimSpace(link)
	if idx := strings.LastIndex(link, "/"); idx >= 0 {
		return link[idx+1:]
	}
	return link
}

// phoneResolver returns a function that resolves LID to phone number.
func (a *WhatsmeowAdapter) phoneResolver(ctx context.Context) func(string) string {
	return func(lid string) string {
		if a.manager != nil {
			return a.manager.lookupPNForLID(ctx, lid)
		}
		return ""
	}
}
