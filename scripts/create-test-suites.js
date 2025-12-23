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

// Find and clone happy path requests
const happyPathRequests = [];

// 1. Login
const loginRequest = findRequestByPath(collection.item, 'Authentication', 'login', 'POST');
if (loginRequest) {
    const cloned = deepClone(loginRequest);
    generateNewIds(cloned);
    cloned.name = "1. Login User";
    happyPathRequests.push(cloned);
}

// 2. Create Account
const createAccountRequest = findRequestByPath(collection.item, 'Accounts', 'create', 'POST');
if (createAccountRequest) {
    const cloned = deepClone(createAccountRequest);
    generateNewIds(cloned);
    cloned.name = "2. Create Account";
    happyPathRequests.push(cloned);
}

// 3. List Accounts
const listAccountsRequest = findRequestByPath(collection.item, 'Accounts', 'all', 'GET');
if (listAccountsRequest) {
    const cloned = deepClone(listAccountsRequest);
    generateNewIds(cloned);
    cloned.name = "3. List All Accounts";
    happyPathRequests.push(cloned);
}

// 4. Get Account (find "Get" request)
const getAccountRequest = findRequestByPath(collection.item, 'Accounts', 'account', 'GET');
if (getAccountRequest && !getAccountRequest.name.toLowerCase().includes('all')) {
    const cloned = deepClone(getAccountRequest);
    generateNewIds(cloned);
    cloned.name = "4. Get Account Details";
    happyPathRequests.push(cloned);
}

// 5. Account Summary/Transactions
const summaryRequest = findRequestByPath(collection.item, 'Accounts', 'transactions', 'GET');
if (summaryRequest) {
    const cloned = deepClone(summaryRequest);
    generateNewIds(cloned);
    cloned.name = "5. Get Account Transactions";
    happyPathRequests.push(cloned);
}

// 6. Customer Search
const searchRequest = findRequestByPath(collection.item, 'Customers', 'search', 'GET');
if (searchRequest) {
    const cloned = deepClone(searchRequest);
    generateNewIds(cloned);
    cloned.name = "6. Search Customers";
    happyPathRequests.push(cloned);
}

// 7. Transfer
const transferRequest = findRequestByPath(collection.item, 'Transactions', 'transfer', 'POST') ||
                        findRequestByPath(collection.item, 'Accounts', 'transfer', 'POST');
if (transferRequest) {
    const cloned = deepClone(transferRequest);
    generateNewIds(cloned);
    cloned.name = "7. Transfer Funds";
    happyPathRequests.push(cloned);
}

// Create Happy Path Suite folder
const happyPathSuite = {
    "name": "🧪 Test Suites",
    "description": "Organized test suites for automated testing via Collection Runner",
    "item": [
        {
            "name": "Happy Path Suite",
            "description": "Complete happy path workflow: Login → Create Account → List Accounts → Transfer → Verify. Run this suite to validate core functionality works correctly.",
            "item": happyPathRequests
        }
    ]
};

// Add the test suites folder to the collection
collection.item.push(happyPathSuite);

// Write back to file
fs.writeFileSync(collectionPath, JSON.stringify(collection, null, 2));

console.log("✅ Test Suites folder created successfully!");
console.log(`📁 Happy Path Suite contains ${happyPathRequests.length} requests:`);
happyPathRequests.forEach(req => {
    console.log(`   - ${req.name}`);
});
console.log("\n💡 To run the Happy Path Suite:");
console.log("   1. Open Postman");
console.log("   2. Navigate to: 🧪 Test Suites → Happy Path Suite");
console.log("   3. Click 'Run' button");
console.log("   4. Click 'Run Happy Path Suite'");
