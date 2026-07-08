import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 20 }, // ramp up to 20 users
    { duration: '20s', target: 20 }, // stay at 20 users
    { duration: '10s', target: 0 },  // scale down
  ],
  thresholds: {
    http_req_duration: ['p(95)<300'], // p95 must be less than 300ms
  },
};

const BASE_URL = __ENV.API_BASE_URL || 'http://localhost:8082';

export default function () {
  const responses = http.batch([
    ['GET', `${BASE_URL}/api/rest/v1/explorer/stats/summary`],
    ['GET', `${BASE_URL}/api/rest/v1/explorer/gas-tracker`],
    ['GET', `${BASE_URL}/api/rest/v1/explorer/charts/tx`],
  ]);

  check(responses[0], {
    'summary status is 200': (r) => r.status === 200,
  });
  check(responses[1], {
    'gas-tracker status is 200': (r) => r.status === 200,
  });

  sleep(1);
}
