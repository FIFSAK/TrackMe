import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomString, randomIntBetween, randomItem } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// Configuration
const BASE_URL = 'http://localhost:80';
const LOGIN_EMAIL = 'admin123@gmail.com';
const LOGIN_PASSWORD = 'admin123';
const TOTAL_CLIENTS = 1000;

export const options = {
  // Stress test configuration
  stages: [
    { duration: '10s', target: 1000 },   // Ramp up
    { duration: '1m', target: 1000 },   // Ramp up
  ],
  
  thresholds: {
    http_req_duration: ['p(95)<3000'],
    http_req_failed: ['rate<0.05'],
    checks: ['rate>0.95'],
  },
  
  setupTimeout: '300s',
};

// All possible stages from your list
const ALL_STAGES = [
  'registration',
  'product_selection',
  'data_consent',
  'form_filling',
  'participants_specification',
  'terms_agreement',
  'client_questionnaire',
  'approval_waiting',
  'modifications',
  'document_signing',
  'payment_waiting',
  'completed'
];

// Global variables
let authToken = '';
let clients = []; // Array of {id, email, current_stage}

// ===== PHASE 1: SETUP =====
export function setup() {
  console.log('🚀 Starting test: Create clients and set random stages');
  
  // 1. Login
  console.log('🔐 Logging in...');
  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({
      email: LOGIN_EMAIL,
      password: LOGIN_PASSWORD
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  
  if (loginRes.status !== 200) {
    throw new Error(`Login failed: ${loginRes.status} - ${loginRes.body}`);
  }
  
  const loginData = JSON.parse(loginRes.body);
  authToken = loginData.data.token;
  console.log(`✅ Logged in. Token: ${authToken.substring(0, 30)}...`);
  
  // 2. Create clients with random initial stages
  console.log(`📝 Creating ${TOTAL_CLIENTS} clients...`);
  
  for (let i = 0; i < TOTAL_CLIENTS; i++) {
    // Random starting stage from the list
    const startStage = randomItem(ALL_STAGES);
    
    const clientData = {
      email: `client_${i}_${randomString(6)}@test.com`,
      name: `Stress Client ${i}`,
      stage: startStage,
      source: 'stress_test',
      channel: 'web',
      is_active: true,
      last_login: new Date().toISOString()
    };
    
    const createRes = http.post(
      `${BASE_URL}/api/v1/clients`,
      JSON.stringify(clientData),
      {
        headers: {
          'Authorization': `Bearer ${authToken}`,
          'Content-Type': 'application/json',
          'accept': 'application/json'
        }
      }
    );
    
    if (createRes.status === 201) {
      const clientJson = JSON.parse(createRes.body);
      clients.push({
        id: clientJson.data.id,
        email: clientData.email,
        current_stage: clientJson.data.current_stage
      });
      
      // Show progress
      if (clients.length % 100 === 0) {
        console.log(`   Created ${clients.length}/${TOTAL_CLIENTS} clients`);
      }
    } else {
      console.error(`Failed to create client ${i}: ${createRes.status} - ${createRes.body}`);
    }
    
    sleep(0.01);
  }
  
  console.log(`✅ Created ${clients.length} clients`);
  
  return {
    token: authToken,
    clients: clients
  };
}

// ===== PHASE 2: STRESS TEST - SET RANDOM STAGES =====
export default function (data) {
  const headers = {
    'accept': 'application/json',
    'Authorization': `Bearer ${data.token}`,
    'Content-Type': 'application/json'
  };
  
  // Pick a random client
  const randomIndex = randomIntBetween(0, data.clients.length - 1);
  const client = data.clients[randomIndex];
  
  // Update payload
  const updatePayload = {
    email: client.email,
    stage: 'registration'
  };
  
  // Make the update request
  const res = http.put(
    `${BASE_URL}/api/v1/clients/${client.id}/stage`,
    JSON.stringify(updatePayload),
    { headers: headers }
  );
  
  // Validations
  const checks = check(res, {
  'status is 200': (r) => r.status === 200,
  'response has data': (r) => {
    try {
      const json = JSON.parse(r.body);
      return !!json.data;
    } catch {
      return false;
    }
  }
});
  
  // Log failures
  if (!checks) {
    console.error(`❌ Update failed for client ${client.id}`);
    console.error(`From: ${client.current_stage}, To: ${targetStage}, Status: ${res.status}`);
  }

}
