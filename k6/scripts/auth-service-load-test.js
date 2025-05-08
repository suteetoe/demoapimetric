import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";

// Custom metrics
const registerErrors = new Counter('register_errors');
const loginErrors = new Counter('login_errors');
const profileErrors = new Counter('profile_errors');
const tenantErrors = new Counter('tenant_errors');

// Response time trends
const registerTrend = new Trend('register_time');
const loginTrend = new Trend('login_time');
const profileTrend = new Trend('profile_time');
const tenantTrend = new Trend('tenant_time');

// Success rates
const registerSuccess = new Rate('register_success');
const loginSuccess = new Rate('login_success');
const profileSuccess = new Rate('profile_success');
const tenantSuccess = new Rate('tenant_success');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 10 }, // Ramp up to 10 users
    { duration: '1m', target: 20 },  // Ramp up to 20 users
    { duration: '2m', target: 20 },  // Stay at 20 users for 2 minutes
    { duration: '30s', target: 0 },  // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'], // 95% of requests should be below 2s
    'http_req_duration{endpoint:register}': ['p(95)<3000'], // Registration can take longer
    'http_req_duration{endpoint:login}': ['p(95)<1000'],    // Login should be fast
    'register_success': ['rate>0.95'],  // 95% of registrations should be successful
    'login_success': ['rate>0.99'],     // 99% of logins should be successful
    'profile_success': ['rate>0.99'],   // 99% of profile requests should be successful
    'tenant_success': ['rate>0.98'],    // 98% of tenant operations should be successful
  },
};

// Shared data
const baseUrl = 'http://localhost:8082';
const userPool = new Map();

// Helper function for generating unique users
function generateUniqueUser() {
  const userId = randomString(8);
  return {
    email: `user_${userId}@example.com`,
    password: `password_${userId}`,
    first_name: `First${userId}`,
    last_name: `Last${userId}`,
  };
}

export default function () {
  const user = generateUniqueUser();
  let authToken = '';

  group('User Registration', function () {
    const registerUrl = `${baseUrl}/auth/register`;
    const payload = JSON.stringify(user);
    
    const params = {
      headers: {
        'Content-Type': 'application/json',
      },
      tags: { endpoint: 'register' },
    };

    const registerRes = http.post(registerUrl, payload, params);
    
    // Store response time
    registerTrend.add(registerRes.timings.duration);
    
    // Check if registration was successful
    const registerChecks = check(registerRes, {
      'registration status is 201': (r) => r.status === 201 || r.status === 200,
      'registration response has userId': (r) => r.json('user.id') !== undefined,
    });
    
    // Update metrics based on checks
    registerSuccess.add(registerChecks);
    
    if (!registerChecks) {
      registerErrors.add(1);
      console.log(`Registration failed: ${registerRes.status} ${registerRes.body}`);
      return;
    }
    
    // Store user in pool for potential future tests
    userPool.set(user.email, user);
    
    // Small sleep to allow backend processing
    sleep(1);
  });

  group('User Login', function () {
    const loginUrl = `${baseUrl}/auth/login`;
    const payload = JSON.stringify({
      email: user.email,
      password: user.password,
    });
    
    const params = {
      headers: {
        'Content-Type': 'application/json',
      },
      tags: { endpoint: 'login' },
    };
    
    const loginRes = http.post(loginUrl, payload, params);
    
    // Store response time
    loginTrend.add(loginRes.timings.duration);
    
    // Check if login was successful
    const loginChecks = check(loginRes, {
      'login status is 200': (r) => r.status === 200,
      'login response has token': (r) => r.json('token') !== undefined,
    });
    
    // Update metrics based on checks
    loginSuccess.add(loginChecks);
    
    if (!loginChecks) {
      loginErrors.add(1);
      console.log(`Login failed: ${loginRes.status} ${loginRes.body}`);
      return;
    }
    
    // Store token for subsequent requests
    authToken = loginRes.json('token');
    
    // Small sleep to simulate user behavior
    sleep(0.5);
  });

  // Only proceed if we have an auth token
  if (authToken) {
    group('User Profile', function () {
      const profileUrl = `${baseUrl}/api/users/profile`;
      
      const params = {
        headers: {
          'Authorization': `Bearer ${authToken}`,
        },
        tags: { endpoint: 'profile' },
      };
      
      const profileRes = http.get(profileUrl, params);
      
      // Store response time
      profileTrend.add(profileRes.timings.duration);
      
      // Check if profile request was successful
      const profileChecks = check(profileRes, {
        'profile status is 200': (r) => r.status === 200,
        'profile response has user data': (r) => r.json('email') !== undefined,
      });
      
      // Update metrics based on checks
      profileSuccess.add(profileChecks);
      
      if (!profileChecks) {
        profileErrors.add(1);
        console.log(`Profile request failed: ${profileRes.status} ${profileRes.body}`);
      }
      
      // Small sleep to simulate user behavior
      sleep(0.5);
    });

    // group('Tenant Selection', function () {
    //   const tenantUrl = `${baseUrl}/api/tenant-auth/select`;
    //   const payload = JSON.stringify({
    //     tenant_id: 1, // Using default tenant ID
    //   });
      
    //   const params = {
    //     headers: {
    //       'Authorization': `Bearer ${authToken}`,
    //       'Content-Type': 'application/json',
    //     },
    //     tags: { endpoint: 'tenant' },
    //   };
      
    //   const tenantRes = http.post(tenantUrl, payload, params);
      
    //   // Store response time
    //   tenantTrend.add(tenantRes.timings.duration);
      
    //   // Check if tenant selection was successful
    //   const tenantChecks = check(tenantRes, {
    //     'tenant selection status is 200': (r) => r.status === 200,
    //     'tenant response has token': (r) => r.json('token') !== undefined,
    //   });
      
    //   // Update metrics based on checks
    //   tenantSuccess.add(tenantChecks);
      
    //   if (!tenantChecks) {
    //     tenantErrors.add(1);
    //     console.log(`Tenant selection failed: ${tenantRes.status} ${tenantRes.body}`);
    //   }
      
    //   // Small sleep to simulate user behavior
    //   sleep(0.5);
    // });
  }

  // Final sleep to control request rate
  sleep(1);
}

// Generate HTML report after test completion
export function handleSummary(data) {
  return {
    "summary.html": htmlReport(data),
  };
}