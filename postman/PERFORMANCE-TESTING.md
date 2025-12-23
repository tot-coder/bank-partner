# Performance Testing Guide - Array Banking API

Complete guide for performance testing the Array Banking API using Postman Collection Runner and test data files.

## Overview

The Array Banking API Postman collection includes comprehensive performance testing capabilities:
- **Response time assertions** on all endpoints
- **Test data CSV file** with 25 test users
- **Collection runner configurations** for load simulation
- **Performance benchmarks** for all operation types

---

## Performance Benchmarks

All endpoints include automated response time assertions:

| Operation Type | Threshold | Examples |
|----------------|-----------|----------|
| **Health Check** | < 100ms | `/health` |
| **Read Operations (GET)** | < 200ms | List accounts, get account, search customers |
| **Write Operations (POST/PUT)** | < 500ms | Create account, transfer funds, update customer |
| **Delete Operations** | < 300ms | Delete account, delete customer |

These assertions run automatically with every request to ensure performance standards are maintained.

---

## Test Data File

### Location
`postman/test-data.csv`

### Structure
The CSV file contains 25 test user accounts with varying scenarios:

| Column | Description | Example |
|--------|-------------|---------|
| `user_email` | Test user email | `user1@example.com` |
| `user_password` | Test user password | `TestPassword123!` |
| `test_scenario` | Scenario type | `happy_path`, `load_test`, `admin_test` |
| `account_type` | Account type to create | `checking`, `savings` |
| `transfer_amount` | Transfer amount | `50.00` |
| `expected_status` | Expected HTTP status | `200`, `201`, `401` |

### Test User Categories

**Regular Users (user1-user10)**: Happy path scenarios
- 10 users with valid credentials
- Mixed checking and savings accounts
- Transfer amounts from $25-$200

**Load Test Users (user11-user20)**: High-volume testing
- 10 users for concurrent load testing
- Varied account types and transfer amounts
- Designed for stress testing scenarios

**Staff/Admin Users (staff1-staff2, admin1-admin2)**: Privileged operations
- 4 users with elevated permissions
- For testing admin endpoints
- Customer search and management operations

**Invalid User (invalid@example.com)**: Error testing
- Incorrect password for 401 testing

---

## Performance Test Scenarios

### Scenario 1: Concurrent User Simulation (10-50 Users)

**Objective**: Test API behavior under concurrent load from multiple users

**Configuration**:
1. Open Postman Collection Runner
2. Select: **🧪 Test Suites → Happy Path Suite**
3. Click **Run** button
4. Configure settings:
   - **Iterations**: 10-50 (simulates 10-50 concurrent requests per endpoint)
   - **Delay**: 100ms (between requests)
   - **Data File**: Select `postman/test-data.csv`
   - **Keep variable values**: ✓ Checked

**What It Tests**:
- Login performance under load
- Account creation throughput
- Account listing with multiple concurrent requests
- Customer search performance
- Transfer processing under load

**Expected Results**:
- All requests complete successfully (200/201 status codes)
- Response times meet benchmark thresholds
- No timeout or connection errors
- Consistent performance across iterations

**Metrics to Monitor**:
- Average response time per endpoint
- Min/Max response times
- Request failure rate (should be 0%)
- Total test duration

---

### Scenario 2: Transaction Throughput Test

**Objective**: Measure maximum transaction processing capacity

**Configuration**:
1. Select: **Transactions → Transfer Funds** (or the transfer request in Happy Path Suite)
2. Click **Run** via Collection Runner
3. Configure settings:
   - **Iterations**: 100 (100 transactions)
   - **Delay**: 50ms (20 transactions per second)
   - **Data File**: Select `postman/test-data.csv`

**What It Tests**:
- Database transaction handling under high volume
- Lock contention and race conditions
- Balance calculation accuracy under load
- Transaction ID generation performance

**Expected Results**:
- Successful transfer processing for valid requests
- Response time < 500ms for 95th percentile
- No duplicate transaction IDs
- Accurate balance updates

**Metrics to Monitor**:
- Transactions per second (TPS)
- Average transaction response time
- P95 and P99 response times
- Error rate for insufficient funds vs. system errors

---

### Scenario 3: Read-Heavy Workload

**Objective**: Test API performance with primarily read operations

**Configuration**:
1. Create custom test suite with GET requests:
   - List Accounts (5x)
   - Get Account Details (3x)
   - Account Transactions (2x)
2. Run via Collection Runner
3. Configure settings:
   - **Iterations**: 50
   - **Delay**: 20ms (fast paced)

**What It Tests**:
- Database read query performance
- Caching effectiveness (if implemented)
- Connection pool efficiency
- Memory usage under read load

**Expected Results**:
- All GET requests < 200ms
- Consistent response times across iterations
- No database connection exhaustion
- Linear scalability

---

### Scenario 4: Authentication Load Test

**Objective**: Test authentication system under high login volume

**Configuration**:
1. Select: **Authentication → Login** endpoint
2. Run via Collection Runner
3. Configure settings:
   - **Iterations**: 100
   - **Delay**: 100ms (10 logins per second)
   - **Data File**: `test-data.csv`

**What It Tests**:
- JWT token generation performance
- Password hashing performance (bcrypt)
- Session management under load
- Rate limiting effectiveness

**Expected Results**:
- Successful authentication for valid credentials
- 401 responses for invalid credentials
- Response time < 500ms for 90th percentile
- Rate limiting triggers after threshold (if configured)

---

## Running Performance Tests

### Step 1: Prepare Environment

1. **Start API Server**:
   ```bash
   go run cmd/api/main.go
   ```

2. **Verify Server Health**:
   ```bash
   curl http://localhost:8080/health
   ```

3. **Create Test Users** (if needed):
   - Use the registration endpoint to create users from `test-data.csv`
   - Or use a database seeding script

### Step 2: Import Collection and Data

1. Open Postman Desktop App
2. Import `Array-Banking-API.postman_collection.json`
3. Import `Array-Banking-API-Local.postman_environment.json`
4. Select the environment in top-right dropdown

### Step 3: Run Performance Test Suite

**Option A: Via Collection Runner (GUI)**

1. Click collection name: **Array Banking API - Complete Test Suite**
2. Click **Run** button (▶️)
3. Select test suite folder (e.g., "Happy Path Suite")
4. Click **Select File** under "Data"
5. Choose `postman/test-data.csv`
6. Configure iterations and delay
7. Click **Run Array Banking API...**
8. Monitor results in real-time

**Option B: Via Postman CLI (newman)**

```bash
# Install newman if not already installed
npm install -g newman

# Run Happy Path Suite with test data
newman run postman/Array-Banking-API.postman_collection.json \
  --environment postman/Array-Banking-API-Local.postman_environment.json \
  --folder "Happy Path Suite" \
  --iteration-data postman/test-data.csv \
  --iteration-count 10 \
  --delay-request 100 \
  --reporters cli,json \
  --reporter-json-export results.json

# Run with higher load
newman run postman/Array-Banking-API.postman_collection.json \
  --environment postman/Array-Banking-API-Local.postman_environment.json \
  --folder "Happy Path Suite" \
  --iteration-data postman/test-data.csv \
  --iteration-count 50 \
  --delay-request 50 \
  --reporters cli,htmlextra \
  --reporter-htmlextra-export performance-report.html
```

### Step 4: Analyze Results

**In Postman Collection Runner**:
- View summary statistics (total, passed, failed)
- Check individual request results
- Review response times (avg, min, max)
- Export results for further analysis

**Key Metrics to Analyze**:
1. **Pass Rate**: Should be 100% for valid test data
2. **Average Response Time**: Should meet benchmark thresholds
3. **P95 Response Time**: 95th percentile should be < 2x average
4. **Failure Rate**: Should be 0% for happy path tests
5. **Throughput**: Requests per second (calculated from total time)

---

## Performance Testing Best Practices

### 1. Baseline Testing
- Run tests with 1 iteration first to establish baseline
- Verify all tests pass before increasing load
- Document baseline response times

### 2. Gradual Load Increase
- Start with low iterations (10)
- Gradually increase to medium (50)
- Then test high load (100+)
- Monitor for performance degradation

### 3. Realistic Test Data
- Use production-like data volumes
- Include edge cases (large transfers, many accounts)
- Vary request patterns (not all identical)

### 4. Environment Consistency
- Use dedicated test environment
- Ensure consistent hardware resources
- Run tests at similar times (avoid peak production hours)

### 5. Result Documentation
- Save collection runner results
- Export newman JSON/HTML reports
- Track performance trends over time
- Set up alerts for regression

---

## Troubleshooting Performance Issues

### Issue: Response Times Exceed Thresholds

**Possible Causes**:
- Database not optimized (missing indexes)
- N+1 query problems
- Inefficient algorithms
- Resource contention (CPU, memory)

**Investigation Steps**:
1. Check API server logs for slow queries
2. Review database query execution plans
3. Monitor server resource utilization
4. Profile application code

### Issue: Requests Timing Out

**Possible Causes**:
- Connection pool exhaustion
- Database locks or deadlocks
- Network latency
- Server overload

**Investigation Steps**:
1. Check database connection pool settings
2. Review transaction isolation levels
3. Monitor network latency between services
4. Check server load and scale if needed

### Issue: High Failure Rate

**Possible Causes**:
- Test data doesn't match database state
- Race conditions in concurrent tests
- Authentication tokens expiring
- Rate limiting active

**Investigation Steps**:
1. Verify test users exist in database
2. Check for proper test data cleanup
3. Review token expiration settings
4. Confirm rate limiting thresholds

### Issue: Inconsistent Performance

**Possible Causes**:
- Cold start effects
- Garbage collection pauses
- Cache warming
- Background processes

**Investigation Steps**:
1. Run warmup iterations before measuring
2. Monitor GC metrics during tests
3. Check for scheduled background jobs
4. Ensure consistent test environment

---

## Advanced Performance Testing

### Using k6 for Advanced Load Testing

For more sophisticated performance testing beyond Postman:

```javascript
// k6 script example (load-test.js)
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '30s', target: 10 },  // Ramp up
    { duration: '1m', target: 50 },   // Stay at 50 users
    { duration: '30s', target: 0 },   // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% under 500ms
  },
};

export default function () {
  // Login
  let loginRes = http.post('http://localhost:8080/api/v1/auth/login', JSON.stringify({
    email: 'user1@example.com',
    password: 'TestPassword123!',
  }), { headers: { 'Content-Type': 'application/json' } });

  check(loginRes, {
    'login status 200': (r) => r.status === 200,
  });

  let token = loginRes.json('accessToken');

  // List accounts
  let accountsRes = http.get('http://localhost:8080/api/v1/accounts', {
    headers: { 'Authorization': `Bearer ${token}` },
  });

  check(accountsRes, {
    'accounts status 200': (r) => r.status === 200,
    'response time < 200ms': (r) => r.timings.duration < 200,
  });

  sleep(1);
}
```

Run with:
```bash
k6 run load-test.js
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Performance Tests

on:
  push:
    branches: [main]
  schedule:
    - cron: '0 0 * * *'  # Daily at midnight

jobs:
  performance-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Start API Server
        run: |
          go run cmd/api/main.go &
          sleep 5

      - name: Install Newman
        run: npm install -g newman newman-reporter-htmlextra

      - name: Run Performance Tests
        run: |
          newman run postman/Array-Banking-API.postman_collection.json \
            --environment postman/Array-Banking-API-Local.postman_environment.json \
            --folder "Happy Path Suite" \
            --iteration-data postman/test-data.csv \
            --iteration-count 25 \
            --reporters cli,htmlextra \
            --reporter-htmlextra-export performance-report.html

      - name: Upload Results
        uses: actions/upload-artifact@v2
        with:
          name: performance-report
          path: performance-report.html
```

---

## Summary

**Performance Testing Capabilities**:
- ✅ Response time assertions on all 12 endpoints
- ✅ Test data CSV with 25 users and multiple scenarios
- ✅ Collection runner configurations for load simulation
- ✅ 4 comprehensive performance test scenarios
- ✅ Best practices and troubleshooting guide
- ✅ CI/CD integration examples

**To Get Started**:
1. Import collection and environment into Postman
2. Start your API server
3. Run Happy Path Suite with test-data.csv
4. Review performance metrics
5. Gradually increase load to find limits

**For Advanced Testing**:
- Use newman CLI for automation
- Integrate with CI/CD pipelines
- Use k6 or Artillery for sophisticated load testing
- Monitor and trend performance over time
