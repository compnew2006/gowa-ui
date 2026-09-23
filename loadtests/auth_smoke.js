import http from 'k6/http';
import { check } from 'k6';

const baseURL = (__ENV.BASE_URL || '').replace(/\/+$/, '');
const email = __ENV.K6_AUTH_EMAIL || '';
const password = __ENV.K6_AUTH_PASSWORD || '';

if (!baseURL) {
  throw new Error('Set BASE_URL to the isolated staging API origin.');
}
if ((__ENV.K6_TARGET_ENV || '').toLowerCase() !== 'staging') {
  throw new Error('Set K6_TARGET_ENV=staging to confirm the target environment.');
}
if (!email || !password) {
  throw new Error('Set K6_AUTH_EMAIL and K6_AUTH_PASSWORD for a dedicated staging user.');
}
const baseURLMatch = baseURL.match(/^https:\/\/([a-z0-9.-]+)(?::[0-9]+)?$/i);
if (!baseURLMatch || !/(^|[.-])(staging|stage|stg|test|uat)([.-]|$)/i.test(baseURLMatch[1])) {
  throw new Error('BASE_URL must be HTTPS and use a staging/stage/stg/test/uat hostname.');
}

export const options = {
  discardResponseBodies: true,
  summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(50)', 'p(95)', 'p(99)'],
  scenarios: {
    single_auth_cycle: {
      executor: 'shared-iterations',
      vus: 1,
      iterations: 1,
      maxDuration: '30s',
      exec: 'authCycle',
    },
  },
  thresholds: {
    http_req_duration: ['p(50)<500', 'p(95)<2000', 'p(99)<3500'],
    http_req_failed: ['rate<0.01'],
    checks: ['rate>0.99'],
  },
};

export function authCycle() {
  const loginResponse = http.post(
    `${baseURL}/api/auth/login`,
    JSON.stringify({ email, password }),
    {
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      tags: { endpoint: 'auth_login' },
      timeout: '10s',
    },
  );
  const loginOK = check(loginResponse, {
    'login: HTTP 200': (res) => res.status === 200,
    'login: JSON response': (res) => (res.headers['Content-Type'] || '').includes('application/json'),
  });
  if (!loginOK) {
    return;
  }

  // The same VU's cookie jar carries the refresh cookie set by login. Refresh
  // consumes that one-use token and replaces it, so this scenario deliberately
  // performs only one cycle and never shares a token across VUs.
  const refreshResponse = http.post(`${baseURL}/api/auth/refresh`, null, {
    headers: { Accept: 'application/json' },
    tags: { endpoint: 'auth_refresh' },
    timeout: '10s',
  });
  check(refreshResponse, {
    'refresh: HTTP 200': (res) => res.status === 200,
    'refresh: JSON response': (res) => (res.headers['Content-Type'] || '').includes('application/json'),
  });
}

export default function () {
  authCycle();
}
