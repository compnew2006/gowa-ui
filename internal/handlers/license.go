package handlers

import (
	"errors"
	"strings"

	"github.com/compnew2006/whatomate/internal/license"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type activateLicenseRequest struct {
	Token string `json:"token"`
}

// GetLicenseBootstrap returns the public license bootstrap payload for the activation UI.
func (a *App) GetLicenseBootstrap(r *fastglue.Request) error {
	if a.License == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "License service is unavailable", nil, "")
	}

	resp, err := a.License.Bootstrap(r.RequestCtx)
	if err != nil {
		a.Log.Error("Failed to load license bootstrap", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to load license status", nil, "")
	}
	return r.SendEnvelope(resp)
}

// ActivateLicense installs a signed offline license token on the current host.
func (a *App) ActivateLicense(r *fastglue.Request) error {
	if a.License == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "License service is unavailable", nil, "")
	}

	var req activateLicenseRequest
	if err := a.decodeRequest(r, &req); err != nil {
		return nil
	}

	resp, err := a.License.Activate(r.RequestCtx, req.Token)
	if err != nil {
		var activationErr *license.ActivationError
		if errors.As(err, &activationErr) {
			return r.SendErrorEnvelope(activationErr.StatusCode, activationErr.Message, map[string]any{
				"code": activationErr.Code,
			}, "")
		}
		a.Log.Error("Failed to activate license", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to activate license", nil, "")
	}
	return r.SendEnvelope(resp)
}

func (a *App) licenseAllowsPublicRequest(path string) bool {
	if path == "/health" || path == "/ready" || path == "/api/license/bootstrap" || path == "/api/license/activate" || path == "/api/webhook" {
		return true
	}
	if path == "" || path == "/" {
		return true
	}
	if !strings.HasPrefix(path, "/api") && path != "/ws" {
		return true
	}
	return false
}

func (a *App) licenseAllowsCleanupRequest(method, path string) bool {
	if a.licenseAllowsPublicRequest(path) {
		return true
	}

	switch {
	case path == "/api/auth/login",
		path == "/api/auth/logout",
		path == "/api/auth/refresh",
		path == "/api/auth/switch-org",
		strings.HasPrefix(path, "/api/auth/sso/"):
		return true
	}

	switch method {
	case fasthttp.MethodGet:
		switch path {
		case "/api/config", "/api/me", "/api/me/organizations", "/api/organizations", "/api/users", "/api/accounts", "/api/instances":
			return true
		}
	case fasthttp.MethodDelete:
		switch {
		case strings.HasPrefix(path, "/api/organizations/"),
			strings.HasPrefix(path, "/api/users/"),
			strings.HasPrefix(path, "/api/accounts/"),
			strings.HasPrefix(path, "/api/instances/"):
			return true
		}
	}

	return false
}

func (a *App) LicenseBlocksRequest(method, path string) bool {
	if a.License == nil {
		return false
	}
	if a.License.IsLocked() {
		return !a.licenseAllowsPublicRequest(path)
	}
	if a.License.RequiresQuotaCleanup() {
		return !a.licenseAllowsCleanupRequest(method, path)
	}
	return false
}

func (a *App) SendLicenseBlocked(r *fastglue.Request) *fastglue.Request {
	if a.License != nil && a.License.RequiresQuotaCleanup() {
		return a.SendLicenseQuotaCleanupRequired(r)
	}
	return a.SendLicenseLocked(r)
}

func (a *App) SendLicenseQuotaCleanupRequired(r *fastglue.Request) *fastglue.Request {
	_ = r.SendErrorEnvelope(fasthttp.StatusLocked, "License quota overage requires cleanup before Whatomate can resume", map[string]any{
		"code":         "license_quota_overage",
		"cleanup_url":  "/license-cleanup",
		"activate_url": "/activate",
	}, "")
	return nil
}

func (a *App) SendLicenseLocked(r *fastglue.Request) *fastglue.Request {
	_ = r.SendErrorEnvelope(fasthttp.StatusLocked, "A valid license is required to use Whatomate", map[string]any{
		"code":         "license_locked",
		"activate_url": "/activate",
	}, "")
	return nil
}

func (a *App) checkQuotaOrRespond(r *fastglue.Request, resource string, orgID uuid.UUID) bool {
	if a.License == nil {
		return true
	}
	if a.License.RequiresQuotaCleanup() {
		a.SendLicenseQuotaCleanupRequired(r)
		return false
	}
	check, err := a.License.CheckQuota(r.RequestCtx, resource, orgID)
	if err != nil {
		a.Log.Error("Failed to evaluate license quota", "resource", resource, "error", err, "organization_id", orgID)
		_ = r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to evaluate license quota", nil, "")
		return false
	}
	if check.Allowed {
		return true
	}

	_ = r.SendErrorEnvelope(fasthttp.StatusPaymentRequired, "Licensed quota exceeded", map[string]any{
		"resource":   check.Resource,
		"current":    check.Current,
		"limit":      check.Limit,
		"over_quota": check.OverQuota,
	}, "")
	return false
}

func (a *App) licenseBlocksValueDelivery() bool {
	return a.License != nil && a.License.BlockValueDelivery()
}
