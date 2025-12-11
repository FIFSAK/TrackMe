import http from 'k6/http';
import { check, sleep } from 'k6';

// Configuration
const BASE_URL = 'http://localhost:80';
const LOGIN_EMAIL = 'admin123@gmail.com';
const LOGIN_PASSWORD = 'admin123';

// Variables to store auth data
let authData = {
  token: '',
  userId: ''
};

// Setup runs once to get auth credentials
export function setup() {
  console.log('Setup: Logging in to get auth credentials...');
  
  const loginPayload = JSON.stringify({
    email: LOGIN_EMAIL,
    password: LOGIN_PASSWORD
  });
  
  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    loginPayload,
    {
      headers: {
        'Content-Type': 'application/json',
        'accept': 'application/json'
      }
    }
  );
  
  if (loginRes.status !== 200) {
    throw new Error(`Login failed: ${loginRes.status} - ${loginRes.body}`);
  }
  
  const response = JSON.parse(loginRes.body);
  authData.token = response.data.token;
  authData.userId = response.data.user.id;
  
  console.log(`✅ Got token: ${authData.token.substring(0, 30)}...`);
  console.log(`✅ User ID: ${authData.userId}`);
  
  return authData;
}

// Main test function
export default function (data) {
  const headers = {
    'accept': 'application/json',
    'Authorization': `Bearer ${data.token}`
  };
  
  // Stress test the GET user endpoint
  const res = http.get(
    `${BASE_URL}/api/v1/users/${data.userId}`,
    { headers: headers }
  );
  
  // Validations
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response has user data': (r) => {
      try {
        const json = JSON.parse(r.body);
        return json.data && json.data.id === data.userId;
      } catch {
        return false;
      }
    },
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
  
  // Small random sleep between requests
  sleep(0.1);
}

// Test options
export const options = {
  stages: [
    { duration: '10s', target: 1000 },  // Normal load
    { duration: '2m', target: 1000 },  // Normal load
  ],
  
  thresholds: {
    'http_req_duration': ['p(95)<1000'],  // 95% of requests under 1s
    'http_req_failed': ['rate<0.01'],     // Less than 1% errors
  },
};