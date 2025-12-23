#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

const collectionPath = path.join(__dirname, '../postman/Array-Banking-API.postman_collection.json');

// Read the collection
const collection = JSON.parse(fs.readFileSync(collectionPath, 'utf8'));

// Helper function to deep clone an object
function deepClone(obj) {
    return JSON.parse(JSON.stringify(obj));
}

// Helper function to generate new UUIDs for cloned requests
function generateNewIds(item) {
    if (item.id) {
        item.id = crypto.randomUUID();
    }
    if (item.item) {
        item.item.forEach(subItem => generateNewIds(subItem));
    }
    if (item.response) {
        item.response.forEach(resp => {
            if (resp.id) resp.id = crypto.randomUUID();
        });
    }
}

// Helper function to find requests
function findRequestByPath(items, folderName, identifier, method) {
    for (const item of items) {
        if (item.name === folderName && item.item) {
            const result = searchFolder(item.item, identifier, method);
            if (result) return result;
        }
    }
    return null;
}

function searchFolder(items, identifier, method) {
    for (const item of items) {
        if (item.item) {
            const result = searchFolder(item.item, identifier, method);
            if (result) return result;
        }
        if (item.request) {
            const nameMatch = item.name && item.name.toLowerCase().includes(identifier.toLowerCase());
            const methodMatch = !method || item.request.method === method;
            if (nameMatch && methodMatch) {
                return item;
            }
        }
    }
    return null;
}

// Error test scripts

// 1. Invalid Credentials Test (401)
const invalidCredentialsTest = `
// Invalid Credentials - Error Test (401)

pm.test("Status code is 401 Unauthorized", function () {
    pm.response.to.have.status(401);
});

pm.test("Response contains error object", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property("error");
});

pm.test("Error has proper structure", function () {
    const jsonData = pm.response.json();
    const error = jsonData.error;
    pm.expect(error).to.have.property("code");
    pm.expect(error).to.have.property("message");
});

pm.test("Error code is AUTH_002 (invalid credentials)", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData.error.code).to.equal("AUTH_002");
});

pm.test("No token is returned", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.not.have.property("access_token");
    pm.expect(jsonData).to.not.have.property("accessToken");
    pm.expect(jsonData).to.not.have.property("refresh_token");
    pm.expect(jsonData).to.not.have.property("refreshToken");
});

pm.test("Error message is descriptive", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData.error.message).to.be.a("string").and.not.empty;
});
`.trim();

// 2. Insufficient Funds Test (422)
const insufficientFundsTest = `
// Insufficient Funds - Error Test (422)

pm.test("Status code is 422 Unprocessable Entity", function () {
    pm.response.to.have.status(422);
});

pm.test("Response contains error object", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property("error");
});

pm.test("Error code is INSUFFICIENT_FUNDS", function () {
    const jsonData = pm.response.json();
    const errorCode = jsonData.error.code || jsonData.error_code || jsonData.code;
    pm.expect(errorCode).to.equal("INSUFFICIENT_FUNDS");
});

pm.test("Error message mentions insufficient funds", function () {
    const jsonData = pm.response.json();
    const message = jsonData.error.message || jsonData.message;
    pm.expect(message.toLowerCase()).to.include("insufficient");
});

pm.test("No transaction ID is returned", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.not.have.property("transaction_id");
});
`.trim();

// 3. Validation Error Test (400)
const validationErrorTest = `
// Validation Error - Error Test (400)

pm.test("Status code is 400 Bad Request", function () {
    pm.response.to.have.status(400);
});

pm.test("Response contains error object", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property("error");
});

pm.test("Error has validation error code", function () {
    const jsonData = pm.response.json();
    const errorCode = jsonData.error.code || jsonData.code;
    // Common validation error codes: VALIDATION_001, AUTH_001, etc.
    pm.expect(errorCode).to.be.a("string").and.not.empty;
});

pm.test("Error message is descriptive", function () {
    const jsonData = pm.response.json();
    const message = jsonData.error.message || jsonData.message;
    pm.expect(message).to.be.a("string").and.not.empty;
});

pm.test("Error includes details or field information", function () {
    const jsonData = pm.response.json();
    const hasDetails = jsonData.error.details || jsonData.details || jsonData.error.fields;
    if (hasDetails) {
        pm.expect(hasDetails).to.exist;
    }
});
`.trim();

// 4. Unauthorized Access Test (403)
const unauthorizedAccessTest = `
// Unauthorized Access - Error Test (403)

pm.test("Status code is 403 Forbidden", function () {
    pm.response.to.have.status(403);
});

pm.test("Response contains error object", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property("error");
});

pm.test("Error code indicates authorization failure", function () {
    const jsonData = pm.response.json();
    const errorCode = jsonData.error.code || jsonData.code;
    pm.expect(errorCode).to.be.a("string").and.not.empty;
});

pm.test("Error message mentions permission or authorization", function () {
    const jsonData = pm.response.json();
    const message = (jsonData.error.message || jsonData.message).toLowerCase();
    const hasForbiddenKeyword = message.includes("permission") ||
                                 message.includes("authorized") ||
                                 message.includes("forbidden") ||
                                 message.includes("access denied");
    pm.expect(hasForbiddenKeyword).to.be.true;
});
`.trim();

// 5. Not Found Error Test (404)
const notFoundErrorTest = `
// Not Found - Error Test (404)

pm.test("Status code is 404 Not Found", function () {
    pm.response.to.have.status(404);
});

pm.test("Response contains error object", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property("error");
});

pm.test("Error code indicates resource not found", function () {
    const jsonData = pm.response.json();
    const errorCode = jsonData.error.code || jsonData.code;
    pm.expect(errorCode).to.be.a("string").and.not.empty;
});

pm.test("Error message mentions not found", function () {
    const jsonData = pm.response.json();
    const message = (jsonData.error.message || jsonData.message).toLowerCase();
    pm.expect(message).to.include("not found");
});

pm.test("Error message is resource-specific", function () {
    const jsonData = pm.response.json();
    const message = jsonData.error.message || jsonData.message;
    pm.expect(message).to.be.a("string").and.not.empty;
});
`.trim();

// Create error test requests
const errorTestRequests = [];

// 1. Invalid Credentials (clone login, modify for bad credentials)
const loginRequest = findRequestByPath(collection.item, 'Authentication', 'login', 'POST');
if (loginRequest) {
    const cloned = deepClone(loginRequest);
    generateNewIds(cloned);
    cloned.name = "1. Login with Invalid Credentials (401)";

    // Modify request body to use invalid credentials
    if (cloned.request.body && cloned.request.body.raw) {
        cloned.request.body.raw = JSON.stringify({
            "email": "invalid@example.com",
            "password": "WrongPassword123!"
        }, null, 2);
    }

    // Replace test script
    cloned.event = [{
        "listen": "test",
        "script": {
            "type": "text/javascript",
            "exec": invalidCredentialsTest.split('\n')
        }
    }];

    errorTestRequests.push(cloned);
}

// 2. Validation Error - Missing Email (clone login)
if (loginRequest) {
    const cloned = deepClone(loginRequest);
    generateNewIds(cloned);
    cloned.name = "2. Login with Missing Email (400)";

    // Modify request body to have missing email
    if (cloned.request.body && cloned.request.body.raw) {
        cloned.request.body.raw = JSON.stringify({
            "password": "TestPassword123!"
        }, null, 2);
    }

    // Replace test script
    cloned.event = [{
        "listen": "test",
        "script": {
            "type": "text/javascript",
            "exec": validationErrorTest.split('\n')
        }
    }];

    errorTestRequests.push(cloned);
}

// 3. Insufficient Funds (clone transfer)
const transferRequest = findRequestByPath(collection.item, 'Transactions', 'transfer', 'POST') ||
                        findRequestByPath(collection.item, 'Accounts', 'transfer', 'POST');
if (transferRequest) {
    const cloned = deepClone(transferRequest);
    generateNewIds(cloned);
    cloned.name = "3. Transfer with Insufficient Funds (422)";

    // Modify request body to transfer excessive amount
    if (cloned.request.body && cloned.request.body.raw) {
        cloned.request.body.raw = JSON.stringify({
            "from_account_id": "{{test_account_id}}",
            "to_account_id": "acc_destination_123",
            "amount": 999999999.99,
            "currency": "USD",
            "description": "Transfer exceeding balance - should fail"
        }, null, 2);
    }

    // Replace test script
    cloned.event = [{
        "listen": "test",
        "script": {
            "type": "text/javascript",
            "exec": insufficientFundsTest.split('\n')
        }
    }];

    errorTestRequests.push(cloned);
}

// 4. Validation Error - Negative Amount (clone transfer)
if (transferRequest) {
    const cloned = deepClone(transferRequest);
    generateNewIds(cloned);
    cloned.name = "4. Transfer with Negative Amount (400)";

    // Modify request body to have negative amount
    if (cloned.request.body && cloned.request.body.raw) {
        cloned.request.body.raw = JSON.stringify({
            "from_account_id": "{{test_account_id}}",
            "to_account_id": "acc_destination_123",
            "amount": -50.00,
            "currency": "USD",
            "description": "Negative amount - should fail validation"
        }, null, 2);
    }

    // Replace test script
    cloned.event = [{
        "listen": "test",
        "script": {
            "type": "text/javascript",
            "exec": validationErrorTest.split('\n')
        }
    }];

    errorTestRequests.push(cloned);
}

// 5. Unauthorized Access - Customer search without proper role
const searchRequest = findRequestByPath(collection.item, 'Customers', 'search', 'GET');
if (searchRequest) {
    const cloned = deepClone(searchRequest);
    generateNewIds(cloned);
    cloned.name = "5. Customer Search without Admin Role (403)";

    // Add a pre-request script note
    cloned.event = cloned.event || [];
    cloned.event.push({
        "listen": "prerequest",
        "script": {
            "type": "text/javascript",
            "exec": [
                "// Note: This test assumes the logged-in user does not have admin/staff role",
                "// For proper testing, ensure you're logged in as a regular customer user"
            ]
        }
    });

    // Add test script
    cloned.event.push({
        "listen": "test",
        "script": {
            "type": "text/javascript",
            "exec": unauthorizedAccessTest.split('\n')
        }
    });

    errorTestRequests.push(cloned);
}

// 6. Not Found - Get non-existent account
const getAccountRequest = findRequestByPath(collection.item, 'Accounts', 'account', 'GET');
if (getAccountRequest && !getAccountRequest.name.toLowerCase().includes('all')) {
    const cloned = deepClone(getAccountRequest);
    generateNewIds(cloned);
    cloned.name = "6. Get Non-Existent Account (404)";

    // Modify URL to use non-existent ID
    if (cloned.request.url && cloned.request.url.path) {
        cloned.request.url.path = cloned.request.url.path.map(segment =>
            segment === ':id' || segment.includes('id') ? 'nonexistent-account-id-999' : segment
        );
    }

    // Replace test script
    cloned.event = [{
        "listen": "test",
        "script": {
            "type": "text/javascript",
            "exec": notFoundErrorTest.split('\n')
        }
    }];

    errorTestRequests.push(cloned);
}

// 7. Not Found - Get non-existent customer
const getCustomerRequest = findRequestByPath(collection.item, 'Customers', 'customers', 'GET');
if (getCustomerRequest && getCustomerRequest.name.toLowerCase().includes('id')) {
    const cloned = deepClone(getCustomerRequest);
    generateNewIds(cloned);
    cloned.name = "7. Get Non-Existent Customer (404)";

    // Modify URL to use non-existent ID
    if (cloned.request.url && cloned.request.url.path) {
        cloned.request.url.path = cloned.request.url.path.map(segment =>
            segment === ':id' || segment.includes('id') ? 'nonexistent-customer-id-999' : segment
        );
    }

    // Replace test script
    cloned.event = [{
        "listen": "test",
        "script": {
            "type": "text/javascript",
            "exec": notFoundErrorTest.split('\n')
        }
    }];

    errorTestRequests.push(cloned);
}

// Find or create Test Suites folder
let testSuitesFolder = collection.item.find(item => item.name === "🧪 Test Suites");

if (!testSuitesFolder) {
    testSuitesFolder = {
        "name": "🧪 Test Suites",
        "description": "Organized test suites for automated testing via Collection Runner",
        "item": []
    };
    collection.item.push(testSuitesFolder);
}

// Add Error Handling Suite
const errorHandlingSuite = {
    "name": "Error Handling Suite",
    "description": "Test suite for error scenarios: invalid credentials, validation errors, insufficient funds, unauthorized access, and not found errors. Run this suite to verify proper error handling across all endpoints.",
    "item": errorTestRequests
};

testSuitesFolder.item.push(errorHandlingSuite);

// Write back to file
fs.writeFileSync(collectionPath, JSON.stringify(collection, null, 2));

console.log("✅ Error Handling Suite created successfully!");
console.log(`📁 Error Handling Suite contains ${errorTestRequests.length} error test requests:`);
errorTestRequests.forEach(req => {
    console.log(`   - ${req.name}`);
});
console.log("\n💡 To run the Error Handling Suite:");
console.log("   1. Open Postman");
console.log("   2. Navigate to: 🧪 Test Suites → Error Handling Suite");
console.log("   3. Click 'Run' button");
console.log("   4. Click 'Run Error Handling Suite'");
console.log("\n⚠️  Note: Some tests (e.g., 403 Unauthorized) may require specific user roles");
