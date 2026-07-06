package tenant_test

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveOrganizationID_PrefersHostLockedOrganization(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestDB(t)

	defaultOrg := &models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Default Organization",
		Slug:      "default-org-" + uuid.NewString()[:8],
	}
	hostOrg := &models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Host Locked Organization",
		Slug:      "tenant-one-" + uuid.NewString()[:8],
	}
	overrideOrg := &models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Header Override Organization",
		Slug:      "tenant-two-" + uuid.NewString()[:8],
	}
	require.NoError(t, db.Create(defaultOrg).Error)
	require.NoError(t, db.Create(hostOrg).Error)
	require.NoError(t, db.Create(overrideOrg).Error)

	req := testutil.NewGETRequest(t)
	req.RequestCtx.Request.SetRequestURI("http://" + hostOrg.Slug + ".localhost/")
	testutil.SetAuthContext(req, defaultOrg.ID, uuid.New())
	testutil.SetHeader(req, "X-Organization-ID", overrideOrg.ID.String())

	resolvedOrgID, err := tenant.ResolveOrganizationID(req, db)
	require.NoError(t, err)
	assert.Equal(t, hostOrg.ID, resolvedOrgID)
}

func TestResolveHostOrganization_IgnoresRootHostsAndIPs(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestDB(t)

	org := &models.Organization{
		BaseModel: models.BaseModel{ID: uuid.New()},
		Name:      "Tenant One",
		Slug:      "tenant-one-" + uuid.NewString()[:8],
	}
	require.NoError(t, db.Create(org).Error)

	for _, host := range []string{
		"http://ofuqalmadenah.com/",
		"http://localhost/",
		"http://127.0.0.1/",
	} {
		req := testutil.NewGETRequest(t)
		req.RequestCtx.Request.SetRequestURI(host)

		resolved, err := tenant.ResolveHostOrganization(req, db)
		require.NoError(t, err, "host=%s", host)
		assert.Nil(t, resolved, "host=%s", host)
	}
}
