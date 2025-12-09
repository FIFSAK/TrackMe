import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomString, randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

export const options = {
  stages: [
    { duration: '30s', target: 10 },  // Ramp up
    // { duration: '1m', target: 50 },   // Normal load
    { duration: '30s', target: 100 }, // Stress load
    // { duration: '30s', target: 0 },   // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
  },
};

export default function () {
  // Generate random data
  const randomId = crypto.randomUUID();
  const timestamp = Date.now();
  
  const payload = {
    email: `testuser${randomId}${timestamp}@gmail.com`,
    name: `User${randomId}`,
    password: generatePassword(),
  };
  
  const headers = {
    'accept': 'application/json',
    'Content-Type': 'application/json',
  };
  
  const res = http.post(
    'http://localhost:80/api/v1/auth/register',
    JSON.stringify(payload),
    { headers: headers }
  );
  
  check(res, {
    'status is 201 or 409': (r) => r.status === 201 || r.status === 409,
    'response has content-type': (r) => r.headers['Content-Type'].includes('application/json'),
    'response time < 1s': (r) => r.timings.duration < 1000,
  });
  
  if (res.status === 201) {
    console.log(`Registered: ${payload.email}`);
    // Optionally save token for later tests
    // const token = res.json('token');
  } else if (res.status === 409) {
    console.log(`User already exists: ${payload.email}`);
  } else {
    console.log(`Failed: ${res.status} - ${res.body}`);
  }
  
  sleep(randomIntBetween(1, 3));
}

function generatePassword() {
  const length = randomIntBetween(8, 16);
  const charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*';
  let password = '';
  
  for (let i = 0; i < length; i++) {
    password += charset.charAt(Math.floor(Math.random() * charset.length));
  }
  
  return password;
}