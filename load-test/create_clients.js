import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomString, randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// Configuration
const BASE_URL = 'http://localhost:80';

// Auth token and user ID from login
let authData = {};

// Stress test configuration
export const options = {
  // Since this is a WRITE operation, use conservative targets
  stages: [
    { duration: '10s', target: 1000},
    { duration: '1m', target: 2000},
  ],
  
  thresholds: {
    // Write operations can be slower
    http_req_duration: [
      'p(95)<2000',  // 95% under 2 seconds
      'p(99)<3000',  // 99% under 3 seconds
    ],
    
    // Keep error rate low for writes
    http_req_failed: ['rate<0.02'],  // Less than 2% errors
    
    // Success rate should be high
    checks: ['rate>0.98'],
  },
  
  // Tags for better reporting
  tags: {
    endpoint: 'create_client',
    test_type: 'stress',
  },
};

// Setup: Login once to get token
export function setup() {
  console.log('🔐 Logging in to get auth token...');
  
  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({
      email: 'admin123@gmail.com',
      password: 'admin123'
    }),
    {
      headers: { 'Content-Type': 'application/json' }
    }
  );
  
  if (loginRes.status !== 200) {
    throw new Error(`Login failed: ${loginRes.status}`);
  }
  
  const response = JSON.parse(loginRes.body);
  authData.token = response.data.token;
  authData.userId = response.data.user.id;
  
  console.log(`✅ Logged in as: ${response.data.user.email}`);
  console.log(`✅ User ID: ${authData.userId}`);
  
  return authData;
}

// Generate random client data
function generateClientData() {
  const randomId = crypto.randomUUID();
  const randomEmail = `client_${randomId}@test.com`;
  const randomName = `Client`;
  
  return {
    app: randomString(6),
    channel: ['web', 'mobile', 'api', 'partner'][randomIntBetween(0, 3)],
    contracts: [
      {
        amount: 10,
        autopayment: 'enabled',
        conclusion_date: '2025-01-02T15:04:05Z',
        expiration_date: '2026-01-02T15:04:05Z',
        id: randomString(8),
        name: `Contract`,
        number: `CT`,
        payment_frequency: 'monthly',
        status: 'active'
      }
    ],
    email: randomEmail,
    is_active: Math.random() > 0.5,
    last_login: '2026-01-02T15:04:05Z',
    name: randomName,
    source: 'google',
    stage: 'registration'
  };
}

// Main test
export default function (data) {
  const headers = {
    'accept': 'application/json',
    'Authorization': `Bearer ${data.token}`,
    'Content-Type': 'application/json'
  };
  
  // Generate unique client data for each request
  const clientData = generateClientData();
  
  // Make the request
  const res = http.post(
    `${BASE_URL}/api/v1/clients`,
    JSON.stringify(clientData),
    { headers: headers }
  );
  
  // Validations
  const checks = check(res, {
    'status is 201': (r) => r.status === 201,
  });
  
  // Log failures
  if (!checks) {
    console.error(`❌ Create client failed: ${res.status}`);
    console.error(`Response: ${res.body.substring(0, 200)}`);
  }
}

