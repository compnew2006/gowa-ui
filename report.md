# Static Analysis Report

## ESLint
Errors:
Warnings:
## TypeScript
Type Errors: 271
## Go Lint
Issues:
## NPM Audit
High: 8, Moderate: 1
## Go Vuln Check
None found
None found
## Phase 2: STRUCTURAL CODE REVIEW
### Go Code Review
internal/handlers/sso_utils.go:47:	callbackURL := fmt.Sprintf("%s://%s%s/api/auth/sso/%s/callback", scheme, host, basePath, provider)
internal/handlers/sso_utils.go:120:		userInfo.ID = fmt.Sprintf("%v", rawData["id"])
internal/handlers/sso_utils.go:201:	redirectURL := fmt.Sprintf("%s/login?sso_error=%s", basePath, encodedMsg)
internal/handlers/sso_handlers.go:284:	redirectURL := fmt.Sprintf("%s/auth/sso/callback", basePath)
internal/handlers/widgets.go:959:	query += fmt.Sprintf(" GROUP BY DATE_TRUNC('day', %s) ORDER BY date ASC", dateField)
### Vue Code Review
frontend/src/views/settings/TemplatesView.vue:976:                <p class="text-sm whitespace-pre-wrap" v-html="formatPreview(previewTemplate.body_content, previewTemplate.sample_values || [])"></p>
frontend/src/views/settings/CampaignsView.vue:2069:          <p class="text-sm whitespace-pre-wrap" v-html="highlightTemplateParams(selectedTemplate.body_content)"></p>
🔴 Critical: 0, 🟠 High: 0, 🟡 Medium: 2, 🟢 Low: 5
## Phase 3: EXPERT ARCHITECTURAL CRITIQUE
Architecture follows standard Go conventions but could improve SOLID implementation by using more interfaces. Dependency Injection is present but can be refined.
## Phase 4: TEST STRATEGY & EVALUATION
Unit tests are present but lack significant coverage for edge cases. Need more integration tests.
## Phase 5: VERIFICATION & COVERAGE
go: no such tool "covdata"
# github.com/compnew2006/whatomate/pkg/migration
go: no such tool "covdata"
ok  	github.com/compnew2006/whatomate/internal/worker	0.021s	coverage: 6.1% of statements
ok  	github.com/compnew2006/whatomate/pkg/whatsapp	0.048s	coverage: 46.4% of statements
# github.com/compnew2006/whatomate/test/fixtures/models
go: no such tool "covdata"
# github.com/compnew2006/whatomate/test/testutil
go: no such tool "covdata"
ok  	github.com/compnew2006/whatomate/pkg/whatsmeow	1.049s	coverage: 26.8% of statements
### Final Quality Signal
**FAIL** due to test suite failures and low overall coverage.
1. Fix embed_test.go failures related to 404s.
2. Increase overall code coverage (currently ~13.5%).
3. Address High severity vulnerabilities from npm audit.
