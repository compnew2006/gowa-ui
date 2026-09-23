import http from 'k6/http';
import { check, sleep } from 'k6';

const baseURL = (__ENV.BASE_URL || '').replace(/\/+$/, '');
const targetEnvironment = (__ENV.K6_TARGET_ENV || '').toLowerCase();
const rawTokens = (__ENV.K6_TOKENS || __ENV.K6_TOKEN || '')
  .split(',')
  .map((token) => token.trim())
  .filter(Boolean);
const peakVUs = Number.parseInt(__ENV.K6_PEAK_VUS || '40', 10);
const warmVUs = Math.max(1, Math.min(5, peakVUs));
const metricTarget = (name, defaultValue) => {
  const value = Number.parseFloat(__ENV[name] || `${defaultValue}`);
  if (!Number.isFinite(value) || value <= 0) {
    throw new Error(`${name} must be a positive number.`);
  }
  return value;
};
const maxErrorRate = Number.parseFloat(__ENV.K6_MAX_ERROR_RATE || '0.01');
const minRequestsPerSecond = metricTarget('K6_MIN_RPS', 5);

if (!baseURL) {
  throw new Error('Set BASE_URL to the isolated staging API origin.');
}
if (targetEnvironment !== 'staging') {
  throw new Error('Set K6_TARGET_ENV=staging to confirm the target environment.');
}
if (rawTokens.length === 0) {
  throw new Error('Set K6_TOKEN or K6_TOKENS to staged read-only access token(s).');
}
if (!Number.isInteger(peakVUs) || peakVUs < 1 || peakVUs > 200) {
  throw new Error('K6_PEAK_VUS must be an integer between 1 and 200.');
}
if (!Number.isFinite(maxErrorRate) || maxErrorRate < 0 || maxErrorRate > 1) {
  throw new Error('K6_MAX_ERROR_RATE must be between 0 and 1.');
}

// Require a recognizable staging hostname as a second guard against an
// accidental production run. Give staging an explicit staging/stg/test/uat DNS
// name rather than disabling this check.
const baseURLMatch = baseURL.match(/^https:\/\/([a-z0-9.-]+)(?::[0-9]+)?$/i);
if (!baseURLMatch || !/(^|[.-])(staging|stage|stg|test|uat)([.-]|$)/i.test(baseURLMatch[1])) {
  throw new Error('BASE_URL must be HTTPS and use a staging/stage/stg/test/uat hostname.');
}

const dateTo = new Date();
const dateFrom = new Date(dateTo.getTime() - 7 * 24 * 60 * 60 * 1000);
const formatDate = (date) => date.toISOString().slice(0, 10);
const analyticsURL = `${baseURL}/api/analytics/agents?from=${formatDate(dateFrom)}&to=${formatDate(dateTo)}&group_by=day`;
const contactsURL = `${baseURL}/api/contacts?page=1&limit=50&has_messages=true`;

export const options = {
  discardResponseBodies: true,
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(50)', 'p(95)', 'p(99)'],
  scenarios: {
    authenticated_read_workload: {
      executor: 'ramping-vus',
      startVUs: warmVUs,
      stages: [
        { duration: '1m', target: warmVUs },
        { duration: '2m', target: warmVUs },
        { duration: '2m', target: peakVUs },
        { duration: '3m', target: peakVUs },
        { duration: '1m', target: 0 },
      ],
      gracefulRampDown: '30s',
    },
  },
  thresholds: {
    http_req_duration: [
      `p(50)<${metricTarget('K6_P50_MS', 300)}`,
      `p(95)<${metricTarget('K6_P95_MS', 1500)}`,
      `p(99)<${metricTarget('K6_P99_MS', 3000)}`,
    ],
    'http_req_duration{endpoint:contacts}': [
      `p(50)<${metricTarget('K6_CONTACTS_P50_MS', 300)}`,
      `p(95)<${metricTarget('K6_CONTACTS_P95_MS', 1200)}`,
      `p(99)<${metricTarget('K6_CONTACTS_P99_MS', 2500)}`,
    ],
    'http_req_duration{endpoint:agent_analytics}': [
      `p(50)<${metricTarget('K6_ANALYTICS_P50_MS', 500)}`,
      `p(95)<${metricTarget('K6_ANALYTICS_P95_MS', 2000)}`,
      `p(99)<${metricTarget('K6_ANALYTICS_P99_MS', 3500)}`,
    ],
    http_req_failed: [`rate<${maxErrorRate}`],
    checks: ['rate>0.99'],
    http_reqs: [`rate>${minRequestsPerSecond}`],
  },
};

function authHeaders() {
  // Distribute VUs across distinct test accounts so the app's per-user API
  // rate limit is measurable without concentrating the whole run on one user.
  const token = rawTokens[(__VU - 1) % rawTokens.length];
  return {
    Authorization: `Bearer ${token}`,
    Accept: 'application/json',
  };
}

function getReadOnly(url, endpoint) {
  const response = http.get(url, {
    headers: authHeaders(),
    tags: { endpoint },
    timeout: '10s',
  });

  check(response, {
    [`${endpoint}: HTTP 200`]: (res) => res.status === 200,
    [`${endpoint}: JSON response`]: (res) => (res.headers['Content-Type'] || '').includes('application/json'),
  });
}

export default function () {
  getReadOnly(contactsURL, 'contacts');

  // Keep analytics as a secondary workload so the test covers another
  // authenticated, database-backed GET without turning the run into an
  // analytics-only stress test.
  if (Math.random() < 0.25) {
    getReadOnly(analyticsURL, 'agent_analytics');
  }

  sleep(0.3);
}
