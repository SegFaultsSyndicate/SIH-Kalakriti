// scripts/load-test/mixed-workload.js
// Load test: Realistic mixed workload (80% reads, 20% writes)

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '2m', target: 50 },
    { duration: '5m', target: 100 },
    { duration: '2m', target: 150 },
    { duration: '3m', target: 150 },
    { duration: '2m', target: 0 },
  ],
  thresholds: {
    'http_req_duration': ['p(95)<1000', 'p(99)<2000'],
    'errors': ['rate<0.02'],
    'http_req_failed': ['rate<0.02'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';

// Auth once per VU
let accessToken = '';

export function setup() {
  // Authenticate once
  const phone = `+919876543210`;
  const verifyRes = http.post(
    `${BASE_URL}/api/v1/auth/otp/verify`,
    JSON.stringify({ phone_e164: phone, code: '000000' }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  if (verifyRes.status === 200) {
    try {
      const body = JSON.parse(verifyRes.body);
      return { token: body.access_token };
    } catch {
      return { token: '' };
    }
  }
  return { token: '' };
}

export default function (data) {
  accessToken = data.token;
  const headers = accessToken ? {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${accessToken}`,
  } : { 'Content-Type': 'application/json' };

  const action = Math.random();

  if (action < 0.4) {
    // 40%: Browse listings
    const res = http.get(`${BASE_URL}/api/v1/listings?status=published&limit=20`);
    check(res, { 'browse status 200': (r) => r.status === 200 }) || errorRate.add(1);
  } else if (action < 0.6) {
    // 20%: Search
    const queries = ['madhubani', 'pottery', 'silk', 'brass'];
    const q = queries[randomIntBetween(0, queries.length - 1)];
    const res = http.get(`${BASE_URL}/api/v1/search?q=${q}`);
    check(res, { 'search status 200': (r) => r.status === 200 }) || errorRate.add(1);
  } else if (action < 0.8) {
    // 20%: View listing detail
    const id = `listing-${randomIntBetween(1, 100)}`;
    const res = http.get(`${BASE_URL}/api/v1/listings/${id}`);
    check(res, { 'detail status in [200,404]': (r) => r.status === 200 || r.status === 404 });
  } else if (action < 0.9 && accessToken) {
    // 10%: Get artisan profile (authenticated)
    const res = http.get(`${BASE_URL}/api/v1/artisans/me`, { headers });
    check(res, { 'profile status in [200,404]': (r) => r.status === 200 || r.status === 404 });
  } else if (accessToken) {
    // 10%: Generate upload URL (write operation)
    const res = http.post(
      `${BASE_URL}/api/v1/media/upload-url`,
      JSON.stringify({ mime_type: 'image/jpeg' }),
      { headers }
    );
    check(res, { 'upload-url status 200': (r) => r.status === 200 }) || errorRate.add(1);
  }

  sleep(randomIntBetween(1, 3));
}
