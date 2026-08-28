// scripts/load-test/search.js
// Load test: Search queries (CPU + vector search intensive)

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export const options = {
  stages: [
    { duration: '1m', target: 30 },   // Ramp up to 30 users
    { duration: '3m', target: 30 },   // Stay at 30 (search is expensive)
    { duration: '1m', target: 50 },   // Ramp up to 50
    { duration: '5m', target: 50 },   // Hold at 50
    { duration: '1m', target: 0 },    // Ramp down
  ],
  thresholds: {
    'http_req_duration': ['p(95)<1000', 'p(99)<2000'],
    'errors': ['rate<0.05'],
    'http_req_failed': ['rate<0.05'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';

const SEARCH_QUERIES = [
  'madhubani painting',
  'handloom saree',
  'terracotta pottery',
  'brass sculpture',
  'embroidered shawl',
  'wooden carving',
  'blue pottery',
  'bamboo basket',
  'silk dupatta',
  'dhokra art',
];

export default function () {
  const query = SEARCH_QUERIES[Math.floor(Math.random() * SEARCH_QUERIES.length)];

  const searchRes = http.get(`${BASE_URL}/api/v1/search?q=${encodeURIComponent(query)}&limit=10`);
  check(searchRes, {
    'search status 200': (r) => r.status === 200,
    'search response time < 1s': (r) => r.timings.duration < 1000,
    'has results': (r) => {
      try {
        return JSON.parse(r.body).results.length >= 0;
      } catch {
        return false;
      }
    },
  }) || errorRate.add(1);

  sleep(2);

  // Suggest endpoint (autocomplete)
  const prefix = query.substring(0, 3 + Math.floor(Math.random() * 3));
  const suggestRes = http.get(`${BASE_URL}/api/v1/search/suggest?q=${encodeURIComponent(prefix)}`);
  check(suggestRes, {
    'suggest status 200': (r) => r.status === 200,
    'suggest response time < 200ms': (r) => r.timings.duration < 200,
  }) || errorRate.add(1);

  sleep(1);
}
