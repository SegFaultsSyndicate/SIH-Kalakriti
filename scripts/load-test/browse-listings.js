// scripts/load-test/browse-listings.js
// Load test: Browse published listings (read-heavy scenario)

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '1m', target: 50 },   // Ramp up to 50 users
    { duration: '3m', target: 50 },   // Stay at 50 users
    { duration: '1m', target: 100 },  // Ramp up to 100 users
    { duration: '5m', target: 100 },  // Stay at 100 users
    { duration: '2m', target: 200 },  // Spike to 200 users
    { duration: '2m', target: 200 },  // Hold spike
    { duration: '1m', target: 0 },    // Ramp down
  ],
  thresholds: {
    'http_req_duration': ['p(95)<500', 'p(99)<1000'],
    'errors': ['rate<0.01'],
    'http_req_failed': ['rate<0.01'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';

export default function () {
  // Browse listings
  const listingsRes = http.get(`${BASE_URL}/api/v1/listings?state=PUBLISHED`);
  check(listingsRes, {
    'listings status 200': (r) => r.status === 200,
    'listings response time < 500ms': (r) => r.timings.duration < 500,
    'has items': (r) => {
      try {
        return JSON.parse(r.body).listings.length > 0;
      } catch {
        return false;
      }
    },
  }) || errorRate.add(1);

  sleep(1);

  // View a specific listing
  if (listingsRes.status === 200) {
    try {
      const items = JSON.parse(listingsRes.body).listings;
      if (items.length > 0) {
        const randomListing = items[Math.floor(Math.random() * items.length)];
        const detailRes = http.get(`${BASE_URL}/api/v1/listings/${randomListing.id}`);
        check(detailRes, {
          'listing detail status 200': (r) => r.status === 200,
          'listing detail response time < 300ms': (r) => r.timings.duration < 300,
        }) || errorRate.add(1);
      }
    } catch (e) {
      errorRate.add(1);
    }
  }

  sleep(2);
}
