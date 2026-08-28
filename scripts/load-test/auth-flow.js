// scripts/load-test/auth-flow.js
// Load test: Authentication flow (OTP request → verify → refresh)

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '30s', target: 10 },
    { duration: '2m', target: 10 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    'http_req_duration': ['p(95)<1000'],
    'errors': ['rate<0.05'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';

export default function () {
  const phone = `+91${Math.floor(7000000000 + Math.random() * 3000000000)}`;

  // Request OTP
  const otpRes = http.post(
    `${BASE_URL}/api/v1/auth/otp/request`,
    JSON.stringify({ phone_e164: phone }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  check(otpRes, {
    'otp request status 200': (r) => r.status === 200,
    'otp request response time < 500ms': (r) => r.timings.duration < 500,
  }) || errorRate.add(1);

  sleep(1);

  // In dev mode with AUTH_DEV_OTP_ENABLED=true, code is always "000000"
  const verifyRes = http.post(
    `${BASE_URL}/api/v1/auth/otp/verify`,
    JSON.stringify({ phone_e164: phone, code: '000000' }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  const verifyOk = check(verifyRes, {
    'verify status 200': (r) => r.status === 200,
    'verify response time < 500ms': (r) => r.timings.duration < 500,
    'has tokens': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.access_token && body.refresh_token;
      } catch {
        return false;
      }
    },
  });

  if (!verifyOk) {
    errorRate.add(1);
    return;
  }

  // Extract tokens
  let accessToken, refreshToken;
  try {
    const body = JSON.parse(verifyRes.body);
    accessToken = body.access_token;
    refreshToken = body.refresh_token;
  } catch {
    errorRate.add(1);
    return;
  }

  sleep(2);

  // Refresh token
  const refreshRes = http.post(
    `${BASE_URL}/api/v1/auth/refresh`,
    JSON.stringify({ refresh_token: refreshToken }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  check(refreshRes, {
    'refresh status 200': (r) => r.status === 200,
    'refresh response time < 500ms': (r) => r.timings.duration < 500,
  }) || errorRate.add(1);

  sleep(1);
}
