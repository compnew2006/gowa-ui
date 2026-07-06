package handlers

import "strings"

const (
	ReasonCodePolicyNoInbound   = "POLICY_NO_INBOUND"
	ReasonCodeInstanceBlocked   = "INSTANCE_BLOCKED"
	ReasonCodeInstanceNotConn   = "INSTANCE_NOT_CONNECTED"
	ReasonCodePolicyDraftOnly   = "POLICY_DRAFT_ONLY"
	ReasonCodePolicyNoInstance  = "POLICY_NO_INSTANCE"
	ReasonCodePolicyRolloutWait = "POLICY_ROLLOUT_AUDIT"
)

func reasonCodeDetails(reasonCode string) map[string]any {
	code := strings.TrimSpace(reasonCode)
	if code == "" {
		return nil
	}
	return map[string]any{
		"reason_code": code,
	}
}
