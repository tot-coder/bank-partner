# Array Banking API - Postman Collection

Complete Postman collection for the Array Banking API with automated test scenarios, environment configuration, and authentication management.

## Quick Start

### 1. Import the Collection and Environment

1. Open Postman Desktop App (v10.0+) or Postman Web
2. Click **Import** in the top left
3. Select **Upload Files**
4. Import both files:
   - `Array-Banking-API.postman_collection.json`
   - `Array-Banking-API-Local.postman_environment.json` (for local development)
   - OR `Array-Banking-API-Template.postman_environment.json` (to create your own environment)

### 2. Select the Environment

1. In the top-right corner of Postman, click the environment dropdown
2. Select **Array Banking API - Local** (or your custom environment)

### 3. Configure Environment Variables

Update the following variables in your environment (click the eye icon 👁️ in top-right, then click **Edit**):

#### Required Variables
- **`base_url`**: API base URL (default: `http://localhost:8080/api/v1`)
- **`test_user_email`**: Your test user email for authentication
- **`test_user_password`**: Your test user password

#### Auto-Populated Variables (leave empty)
These will be automatically set by the collection scripts:
- `access_token` - JWT access token (populated after login)
- `refresh_token` - JWT refresh token (populated after login)
- `token_expiry` - Token expiration timestamp
- `test_account_id` - Account ID for testing
- `test_customer_id` - Customer ID for testing
- `test_transaction_id` - Transaction ID for testing

### 4. Authenticate

1. Expand the **Authentication** folder
2. Expand the **login** subfolder
3. Click on **Login user**
4. Update the request body with valid credentials:
   ```json
   {
     "email": "{{test_user_email}}",
     "password": "{{test_user_password}}"
   }
   ```
   OR use your environment variables (they'll automatically populate)
5. Click **Send**
6. ✅ If successful, the access token and refresh token will be automatically stored in your environment

### 5. Start Testing

You're now ready to use any endpoint in the collection! The authentication token will be automatically included in all requests.

---

## Collection Structure

The collection is organized into 5 main folders:

### 🔐 Authentication (5 endpoints)
- **Login** - Authenticate and receive JWT tokens
- **Register** - Create a new user account
- **Refresh** - Refresh expired access token
- **Logout** - Invalidate current session
- **Me** - Get current user profile

### 💰 Accounts (4 endpoints)
- **Get all user accounts** - List all accounts for authenticated user
- **Create account** - Create a new bank account
- **Get account by ID** - Retrieve specific account details
- **Get account summary** - Account summary with recent transactions

### 👥 Customers (4 endpoints)
- **Search customers** - Search for customers (staff/admin only)
- **Create customer** - Create new customer record (staff/admin only)
- **Get customer by ID** - Retrieve customer details
- **Update customer** - Update customer information

### 💸 Transactions (1 endpoint)
- **Create transfer** - Transfer funds between accounts

### ⚙️ Admin (4 endpoints)
- **Health check** - System health status
- **Admin endpoints** - User management and system operations

---

## Automated Testing Features

### Collection-Level Scripts

The collection includes automatic scripts that run on every request:

**Pre-Request Script:**
- Checks token expiration
- Automatically adds Authorization header with Bearer token
- Logs warnings if token is expired

**Post-Request Script:**
- Logs response times for performance monitoring
- Alerts on authentication errors (401)
- Provides helpful reminders to re-authenticate

### Login Endpoint Tests

The Login endpoint includes comprehensive automated tests:

✅ **Status Code Validation** - Verifies 200 OK response
✅ **Token Presence** - Confirms access and refresh tokens exist
✅ **JWT Format Validation** - Verifies token structure
✅ **Auto Token Storage** - Saves tokens to environment automatically
✅ **Expiry Calculation** - Computes and stores token expiration time

### Running Test Suites

#### Individual Request Testing
1. Click any request
2. Click **Send**
3. View test results in the **Test Results** tab

#### Collection Runner (Batch Testing)
1. Click the collection name **Array Banking API - Complete Test Suite**
2. Click **Run** (▶️ button)
3. Select which folder to run (or run entire collection)
4. Click **Run Array Banking API...**
5. View aggregated test results

---

## Environment Variables Reference

### Base Configuration
| Variable | Purpose | Example Value |
|----------|---------|---------------|
| `base_url` | API base URL | `http://localhost:8080/api/v1` |
| `api_version` | API version | `v1` |
| `baseUrl` | Alternative base URL format | `http://localhost:8080/api/v1` |

### Authentication Tokens (Auto-populated)
| Variable | Purpose | Set By |
|----------|---------|--------|
| `access_token` | JWT access token | Login endpoint |
| `refresh_token` | JWT refresh token | Login endpoint |
| `token_expiry` | Token expiration timestamp | Login endpoint |

### Test Data (Auto-populated or manual)
| Variable | Purpose | Set By |
|----------|---------|--------|
| `test_account_id` | Account ID for testing | Manual or test scripts |
| `test_customer_id` | Customer ID for testing | Manual or test scripts |
| `test_transaction_id` | Transaction ID for testing | Manual or test scripts |

### Test Credentials (Manual)
| Variable | Purpose | Example Value |
|----------|---------|---------------|
| `test_user_email` | Test user email | `test@example.com` |
| `test_user_password` | Test user password | `TestPassword123!` |

---

## Troubleshooting

### ❌ Getting 401 Unauthorized Errors

**Problem:** Your token has expired or is invalid.

**Solution:**
1. Run the **Login** endpoint again
2. Verify your credentials are correct
3. Check that your environment is selected in the dropdown

### ❌ Tests Failing with Connection Refused

**Problem:** API server is not running.

**Solution:**
1. Start your Array Banking API server: `go run cmd/api/main.go`
2. Verify the server is running on the correct port (default: 8080)
3. Check `base_url` environment variable matches your server URL

### ❌ Request Variables Not Populating

**Problem:** `{{variable}}` placeholders not replacing with actual values.

**Solution:**
1. Ensure an environment is selected (top-right dropdown)
2. Check environment variables are defined (click 👁️ icon)
3. Variable names are case-sensitive - verify exact spelling

### ❌ Cannot Find Login Endpoint

**Problem:** Collection folder structure is confusing.

**Solution:**
The login endpoint is nested: **Authentication > login > Login user**
(There's a subfolder called "login" inside the Authentication folder)

---

## Performance Testing

The collection includes comprehensive performance testing capabilities with automated assertions and test data files.

### Quick Performance Test

1. Click the collection **Array Banking API - Complete Test Suite**
2. Click **Run**
3. Select **Happy Path Suite** folder
4. Click **Select File** under Data and choose `test-data.csv`
5. Configure iterations:
   - **Iterations**: 10-50 (simulates concurrent users)
   - **Delay**: 100ms (delay between requests)
6. Click **Run Array Banking API...**
7. Review performance metrics in the run summary

### Performance Benchmarks

All endpoints include automated response time assertions:
- **Health check**: < 100ms
- **Read operations (GET)**: < 200ms
- **Write operations (POST/PUT)**: < 500ms
- **Delete operations**: < 300ms

### Test Data File

**Location**: `postman/test-data.csv`

Contains 25 test users across different scenarios:
- 10 regular users for happy path testing
- 10 load test users for stress testing
- 4 staff/admin users for privileged operations
- 1 invalid user for error testing

### Performance Test Scenarios

1. **Concurrent User Simulation**: 10-50 iterations with test data
2. **Transaction Throughput**: 100 iterations with 50ms delay
3. **Read-Heavy Workload**: Multiple GET requests in rapid succession
4. **Authentication Load**: High-volume login testing

### Advanced Performance Testing

For detailed performance testing instructions, scenarios, and best practices, see:

📄 **[PERFORMANCE-TESTING.md](PERFORMANCE-TESTING.md)** - Complete performance testing guide

The guide includes:
- Detailed test scenarios with configurations
- newman CLI examples for automation
- CI/CD integration examples
- Troubleshooting performance issues
- Advanced load testing with k6

---

## API Documentation

For complete API documentation with schemas and error codes, visit:
- **Scalar UI**: http://localhost:8080/docs
- **OpenAPI JSON**: http://localhost:8080/docs/swagger.json

---

## Tips & Best Practices

### 🔒 Security
- Never commit environment files with real credentials to version control
- Use Postman's secret variable type for passwords and tokens
- Regenerate test credentials regularly

### 📝 Test Data Management
- Create separate environments for different test users
- Use descriptive environment names (e.g., "Array API - Admin User", "Array API - Customer User")
- Keep test account IDs updated in environment variables

### 🚀 Productivity
- Use Postman's search (Ctrl/Cmd + K) to quickly find endpoints
- Create custom folders for "Happy Path Suite" and "Error Handling Suite"
- Save example requests/responses for reference

### 🧪 Testing Workflow
1. Start with Authentication (login)
2. Test read operations (GET requests) first
3. Test create operations (POST requests)
4. Test update operations (PUT/PATCH requests)
5. Test error scenarios last (invalid data, authorization errors)

---

## Developer Maintenance

This section is for developers who need to regenerate or maintain the Postman collection.

### How the Collection is Generated

The Postman collection is **automatically generated** from the OpenAPI 3.1 specification and then enhanced with multiple scripts. This ensures the collection stays in sync with the API code while retaining all the test features.

### Regenerating the Collection

When you update the API code (add/modify endpoints), regenerate the collection:

```bash
make postman
```

This single command performs 8 steps:

1. **Generate OpenAPI docs** - Extracts documentation from Go code annotations
2. **Generate base collection** - Converts OpenAPI spec to Postman format
3. **Fix auth & variables** - Removes individual auth, adds collection variables
4. **Organize folders** - Capitalizes folder names, organizes structure
5. **Add collection scripts** - Adds pre-request and test scripts at collection level
6. **Add happy path tests** - Adds automated tests for successful operations
7. **Add error tests** - Creates Error Handling Suite with failure scenarios
8. **Create test suites** - Organizes tests into Happy Path and Error suites
9. **Add performance assertions** - Adds response time validation (< 100ms, < 200ms, < 500ms)

### What Gets Enhanced

The regeneration process adds these features that would be lost from raw OpenAPI conversion:

#### 1. Authentication Management
- **Collection-level auth** with `Bearer {{access_token}}`
- Individual requests inherit auth automatically
- Public endpoints (login, register, health) marked as `noauth`

#### 2. Smart Variable Replacements
All placeholder values are replaced with Postman variables:
- `"email": "<string>"` → `"email": "{{test_user_email}}"`
- `"password": "<string>"` → `"password": "{{test_user_password}}"`
- `"refresh_token": "<string>"` → `"refresh_token": "{{refresh_token}}"`

**Total**: ~52 automatic replacements across all requests

#### 3. Collection Variables
Pre-configured variables for easy testing:
- `access_token` - Auto-populated by login scripts
- `refresh_token` - Auto-populated by login scripts
- `test_user_email` - Default: john.doe@example.com
- `test_user_password` - Default: Password123!

#### 4. Automated Test Scripts
- **Collection-level scripts**: Token expiration checks, auto-headers
- **Login tests**: 61 lines validating tokens, JWT format, auto-storage
- **Happy path tests**: Success scenarios for key endpoints
- **Error tests**: Failure scenarios (401, 400, 422, 403)
- **Performance assertions**: Response time validation for all endpoints

#### 5. Test Suites
- **Happy Path Suite**: Sequential flow testing (login → create → read → transfer)
- **Error Handling Suite**: Validates error responses and status codes

### Manual Workflow (if needed)

If you need to run steps individually:

```bash
# 1. Generate OpenAPI docs
make docs

# 2. Generate base collection
npx openapi-to-postmanv2 \
  -s docs/swagger.json \
  -o postman/Array-Banking-API.postman_collection.json \
  -p

# 3. Apply all enhancements
node scripts/fix-postman-collection.js
node scripts/organize-postman-collection.js
node scripts/add-auth-scripts.js
node scripts/add-happy-path-tests.js
node scripts/add-error-tests.js
node scripts/create-test-suites.js
node scripts/add-performance-assertions.js
```

### Adding New Automatic Fixes

To add new features to the regeneration workflow:

#### Add Variable Replacements

Edit `scripts/fix-postman-collection.js`:

```javascript
const VARIABLE_REPLACEMENTS = {
  // Existing replacements
  '\\"email\\": \\"<string>\\"': '\\"email\\": \\"{{test_user_email}}\\"',

  // Add new replacements
  'your-pattern': '{{your_variable}}',
};
```

#### Mark Public Endpoints

Edit `scripts/fix-postman-collection.js`:

```javascript
const PUBLIC_ENDPOINTS = [
  '/auth/register',
  '/auth/login',
  '/auth/refresh',
  '/health',
  '/docs',
  // Add new public endpoints here
];
```

#### Add Collection Variables

Edit `scripts/fix-postman-collection.js`:

```javascript
const variables = [
  {
    key: 'your_variable',
    value: 'default_value',
    type: 'string',
    description: 'Your variable description'
  },
  // ... other variables
];
```

### Troubleshooting Regeneration

#### Scripts in Collection Are Missing

**Problem**: After running `make postman`, the collection doesn't have test scripts.

**Solution**:
- All 7 enhancement scripts must run successfully
- Check for errors in the output: `make postman 2>&1 | grep -i error`
- Verify Node.js is installed: `node --version` (requires v16+)

#### Variables Not Replaced

**Problem**: Still seeing `"<string>"` instead of `{{variables}}`.

**Solution**:
- Check `scripts/fix-postman-collection.js` has the correct patterns
- Patterns must match the exact JSON format including escaping
- Run script manually to see debug output: `node scripts/fix-postman-collection.js`

#### Duplicate Test Suites Folders

**Problem**: Multiple "🧪 Test Suites" folders appear.

**Solution**:
- This is a known issue with scripts creating separate folders
- Safe to ignore - both Happy Path and Error suites are accessible
- Future improvement: merge into single folder in organize script

### Best Practices

1. **Never Edit the Collection Manually**: Always regenerate with `make postman`
2. **Edit Scripts, Not Collection**: Add features to the enhancement scripts
3. **Test After Regeneration**: Import updated collection into Postman and verify
4. **Version Control**: Commit both the collection and all enhancement scripts
5. **Document Changes**: Update this README when adding new features

### File Structure

```
.
├── docs/
│   ├── swagger.json           # OpenAPI 3.1 spec (source for Postman)
│   └── swagger.yaml           # OpenAPI 3.1 spec (YAML format)
├── postman/
│   ├── Array-Banking-API.postman_collection.json  # Enhanced collection
│   ├── Array-Banking-API-Local.postman_environment.json
│   ├── Array-Banking-API-Template.postman_environment.json
│   └── README.md              # This file
├── scripts/
│   ├── fix-postman-collection.js          # Step 3: Auth & variables
│   ├── organize-postman-collection.js     # Step 4: Folder organization
│   ├── add-auth-scripts.js                # Step 5: Collection scripts
│   ├── add-happy-path-tests.js            # Step 6: Success tests
│   ├── add-error-tests.js                 # Step 7: Error tests
│   ├── create-test-suites.js              # Step 8: Test suites
│   └── add-performance-assertions.js      # Step 9: Performance tests
└── Makefile                                # Contains 'make postman' target
```

---

## Support

For issues or questions:
- **API Documentation**: Check http://localhost:8080/docs
- **Postman Documentation**: https://learning.postman.com/docs/
- **Project Issues**: Open an issue in the project repository

---

**Last Updated**: 2025-10-24
**Collection Version**: 1.0
**Postman Compatibility**: v10.0+
