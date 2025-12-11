import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = 'http://localhost:80';
const LOGIN_EMAIL = 'admin123@gmail.com';
const LOGIN_PASSWORD = 'admin123';

let authToken = '';

export function setup() {
  // Login
  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({
      email: LOGIN_EMAIL,
      password: LOGIN_PASSWORD
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  
  const loginData = JSON.parse(loginRes.body);
  return { token: loginData.data.token };
}

export default function (data) {
  const headers = {
    'accept': 'application/json',
    'Authorization': `Bearer ${data.token}`
  };
  
  // Test different variations of rollback-count
  const variations = [
    '?type=rollback-count',
    '?type=rollback-count&interval=day',
    '?type=rollback-count&interval=week',
    '?type=rollback-count&interval=month',
    '?type=rollback-count&limit=10&offset=0',
    '?type=rollback-count&interval=day&limit=50&offset=100'
  ];
  
  const url = `${BASE_URL}/api/v1/metrics${variations[__ITER % variations.length]}`;
  
  const res = http.get(url, { headers: headers });
  
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
  
  sleep(0.1);
}

export const options = {
  stages: [
    { duration: '10s', target: 1000 },
    { duration: '2m', target: 1000 },
  ],
  
  thresholds: {
    http_req_duration: ['p(95)<1000'],
    http_req_failed: ['rate<0.02'],
  },
};