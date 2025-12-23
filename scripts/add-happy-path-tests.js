#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const collectionPath = path.join(__dirname, '../postman/Array-Banking-API.postman_collection.json');

// Read the collection
const collection = JSON.parse(fs.readFileSync(collectionPath, 'utf8'));

// Helper function to find a request by folder name and request name/method
function findRequest(items, folderName, requestIdentifier, method = null) {
    for (const item of items) {
        if (item.name === folderName) {
            return searchInFolder(item.item || [], requestIdentifier, method);
        }
    }
    return null;
}

function searchInFolder(items, requestIdentifier, method) {
    for (const item of items) {
        // Check if this is a subfolder
        if (item.item && item.item.length > 0) {
            // Check if the folder name matches
            if (item.name && item.name.toLowerCase().includes(requestIdentifier.toLowerCase())) {
                // Search inside this subfolder
                const result = searchInFolder(item.item, requestIdentifier, method);
                if (result) return result;
            }
            // Also search in nested items
            const result = searchInFolder(item.item, requestIdentifier, method);
            if (result) return result;
        }

        // Check if this is the request we're looking for
        if (item.request) {
            const nameMatch = item.name && item.name.toLowerCase().includes(requestIdentifier.toLowerCase());
            const methodMatch = !method || item.request.method === method;

            if (nameMatch && methodMatch) {
                return item;
            }
        }
    }
    return null;
}

// Test scripts for each endpoint

// Account Creation Test
const accountCreationTest = `
// Account Creation - Happy Path Tests

pm.test("Status code is 201 Created", function () {
    pm.response.to.have.status(201);
});

pm.test("Response contains account ID", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property("id");
    pm.expect(jsonData.id).to.be.a("string").and.not.empty;
});

pm.test("Account type matches request", function () {
    const jsonData = pm.response.json();
    const requestBody = JSON.parse(pm.request.body.raw);
    pm.expect(jsonData.account_type || jsonData.accountType).to.equal(requestBody.account_type || requestBody.accountType);
});

pm.test("Account has initial balance", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property("balance");
    pm.expect(jsonData.balance).to.be.a("number");
});

pm.test("Account has account number", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property("account_number");
    pm.expect(jsonData.account_number || jsonData.accountNumber).to.be.a("string").and.not.empty;
});

// Store account ID for use in other tests
if (pm.response.code === 201) {
    const jsonData = pm.response.json();
    pm.environment.set("test_account_id", jsonData.id);
    console.log("✅ Account ID stored: " + jsonData.id);
}
`.trim();

// Account Listing Test
const accountListingTest = `
// Account Listing - Happy Path Tests

pm.test("Status code is 200 OK", function () {
    pm.response.to.have.status(200);
});

pm.test("Response is an array or has accounts array", function () {
    const jsonData = pm.response.json();
    if (Array.isArray(jsonData)) {
        pm.expect(jsonData).to.be.an("array");
    } else {
        pm.expect(jsonData).to.have.property("accounts");
        pm.expect(jsonData.accounts).to.be.an("array");
    }
});

pm.test("Each account has required fields", function () {
    const jsonData = pm.response.json();
    const accounts = Array.isArray(jsonData) ? jsonData : jsonData.accounts;

    if (accounts && accounts.length > 0) {
        accounts.forEach(account => {
            pm.expect(account).to.have.property("id");
            pm.expect(account).to.have.property("account_type");
            pm.expect(account).to.have.property("balance");
        });
    }
});

pm.test("Response has pagination metadata (if applicable)", function () {
    const jsonData = pm.response.json();
    if (jsonData.pagination) {
        pm.expect(jsonData.pagination).to.have.property("page");
        pm.expect(jsonData.pagination).to.have.property("limit");
    }
});

pm.test("Response time is acceptable", function () {
    pm.expect(pm.response.responseTime).to.be.below(500);
});
`.trim();

// Account Summary Test
const accountSummaryTest = `
// Account Summary - Happy Path Tests

pm.test("Status code is 200 OK", function () {
    pm.response.to.have.status(200);
});

pm.test("Response contains account information", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property("account");
    pm.expect(jsonData.account).to.have.property("id");
    pm.expect(jsonData.account).to.have.property("balance");
});

pm.test("Response contains summary data", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property("summary");
    pm.expect(jsonData.summary).to.be.an("object");
});

pm.test("Summary has transaction counts", function () {
    const jsonData = pm.response.json();
    if (jsonData.summary) {
        pm.expect(jsonData.summary).to.have.property("transaction_count");
    }
});

pm.test("Response includes recent transactions", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property("recent_transactions");
    pm.expect(jsonData.recent_transactions).to.be.an("array");
});

pm.test("Balance calculations are consistent", function () {
    const jsonData = pm.response.json();
    if (jsonData.summary && jsonData.summary.total_credits !== undefined && jsonData.summary.total_debits !== undefined) {
        const expectedChange = jsonData.summary.total_credits - jsonData.summary.total_debits;
        const actualChange = jsonData.summary.ending_balance - jsonData.summary.starting_balance;
        pm.expect(Math.abs(actualChange - expectedChange)).to.be.below(0.01); // Account for floating point
    }
});
`.trim();

// Customer Search Test
const customerSearchTest = `
// Customer Search - Happy Path Tests

pm.test("Status code is 200 OK", function () {
    pm.response.to.have.status(200);
});

pm.test("Response contains customers array", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property("customers");
    pm.expect(jsonData.customers).to.be.an("array");
});

pm.test("Each customer has required fields", function () {
    const jsonData = pm.response.json();
    if (jsonData.customers && jsonData.customers.length > 0) {
        jsonData.customers.forEach(customer => {
            pm.expect(customer).to.have.property("id");
            pm.expect(customer).to.have.property("first_name");
            pm.expect(customer).to.have.property("last_name");
            pm.expect(customer).to.have.property("email");
        });
    }
});

pm.test("Response has pagination metadata", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property("pagination");
    pm.expect(jsonData.pagination).to.have.property("page");
    pm.expect(jsonData.pagination).to.have.property("limit");
    pm.expect(jsonData.pagination).to.have.property("total");
});

pm.test("Response time is acceptable for search", function () {
    pm.expect(pm.response.responseTime).to.be.below(1000);
});

// Store first customer ID if available
if (pm.response.code === 200) {
    const jsonData = pm.response.json();
    if (jsonData.customers && jsonData.customers.length > 0) {
        pm.environment.set("test_customer_id", jsonData.customers[0].id);
        console.log("✅ Customer ID stored: " + jsonData.customers[0].id);
    }
}
`.trim();

// Fund Transfer Test
const fundTransferTest = `
// Fund Transfer - Happy Path Tests

pm.test("Status code is 200 OK or 201 Created", function () {
    pm.expect([200, 201]).to.include(pm.response.code);
});

pm.test("Response contains transaction ID", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property("transaction_id");
    pm.expect(jsonData.transaction_id).to.be.a("string").and.not.empty;
});

pm.test("Response contains debit and credit transaction IDs", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData).to.have.property("debit_transaction_id");
    pm.expect(jsonData).to.have.property("credit_transaction_id");
});

pm.test("Transfer amount matches request", function () {
    const jsonData = pm.response.json();
    const requestBody = JSON.parse(pm.request.body.raw);
    pm.expect(jsonData.amount).to.equal(requestBody.amount);
});

pm.test("Transfer status is completed", function () {
    const jsonData = pm.response.json();
    pm.expect(jsonData.status).to.equal("completed");
});

pm.test("From and to account IDs match request", function () {
    const jsonData = pm.response.json();
    const requestBody = JSON.parse(pm.request.body.raw);
    pm.expect(jsonData.from_account_id).to.equal(requestBody.from_account_id);
    pm.expect(jsonData.to_account_id).to.equal(requestBody.to_account_id);
});

// Store transaction ID for later tests
if (pm.response.code === 200 || pm.response.code === 201) {
    const jsonData = pm.response.json();
    pm.environment.set("test_transaction_id", jsonData.transaction_id);
    console.log("✅ Transaction ID stored: " + jsonData.transaction_id);
}
`.trim();

// Add test scripts to appropriate endpoints
let addedTests = 0;

// 1. Account Creation (POST /accounts)
const createAccountRequest = findRequest(collection.item, 'Accounts', 'create', 'POST');
if (createAccountRequest) {
    createAccountRequest.event = createAccountRequest.event || [];
    createAccountRequest.event.push({
        "listen": "test",
        "script": {
            "type": "text/javascript",
            "exec": accountCreationTest.split('\n')
        }
    });
    console.log("✅ Added test script to Create Account endpoint");
    addedTests++;
}

// 2. Account Listing (GET /accounts)
const listAccountsRequest = findRequest(collection.item, 'Accounts', 'all', 'GET');
if (listAccountsRequest) {
    listAccountsRequest.event = listAccountsRequest.event || [];
    listAccountsRequest.event.push({
        "listen": "test",
        "script": {
            "type": "text/javascript",
            "exec": accountListingTest.split('\n')
        }
    });
    console.log("✅ Added test script to List Accounts endpoint");
    addedTests++;
}

// 3. Account Summary (GET /accounts/{id}/summary or similar)
const accountSummaryRequest = findRequest(collection.item, 'Accounts', 'summary', 'GET') ||
                               findRequest(collection.item, 'Accounts', 'transactions', 'GET');
if (accountSummaryRequest) {
    accountSummaryRequest.event = accountSummaryRequest.event || [];
    accountSummaryRequest.event.push({
        "listen": "test",
        "script": {
            "type": "text/javascript",
            "exec": accountSummaryTest.split('\n')
        }
    });
    console.log("✅ Added test script to Account Summary/Transactions endpoint");
    addedTests++;
}

// 4. Customer Search (GET /customers/search)
const customerSearchRequest = findRequest(collection.item, 'Customers', 'search', 'GET');
if (customerSearchRequest) {
    customerSearchRequest.event = customerSearchRequest.event || [];
    customerSearchRequest.event.push({
        "listen": "test",
        "script": {
            "type": "text/javascript",
            "exec": customerSearchTest.split('\n')
        }
    });
    console.log("✅ Added test script to Customer Search endpoint");
    addedTests++;
}

// 5. Fund Transfer (POST /transfers or /accounts/{id}/transfer)
const transferRequest = findRequest(collection.item, 'Transactions', 'transfer', 'POST') ||
                        findRequest(collection.item, 'Accounts', 'transfer', 'POST');
if (transferRequest) {
    transferRequest.event = transferRequest.event || [];
    transferRequest.event.push({
        "listen": "test",
        "script": {
            "type": "text/javascript",
            "exec": fundTransferTest.split('\n')
        }
    });
    console.log("✅ Added test script to Fund Transfer endpoint");
    addedTests++;
}

// Write back to file
fs.writeFileSync(collectionPath, JSON.stringify(collection, null, 2));

console.log(`\n✅ Collection updated with ${addedTests} happy path test scripts!`);
console.log("📝 Test scripts added for:");
console.log("   - Account Creation (POST)");
console.log("   - Account Listing (GET)");
console.log("   - Account Summary/Transactions (GET)");
console.log("   - Customer Search (GET)");
console.log("   - Fund Transfer (POST)");
