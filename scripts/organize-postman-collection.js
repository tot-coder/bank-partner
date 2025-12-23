#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const collectionPath = path.join(__dirname, '../postman/Array-Banking-API.postman_collection.json');

// Read the collection
const collection = JSON.parse(fs.readFileSync(collectionPath, 'utf8'));

// Helper function to capitalize folder names
function capitalizeFolder(name) {
    const nameMap = {
        'accounts': 'Accounts',
        'admin': 'Admin',
        'auth': 'Authentication',
        'customers': 'Customers',
        'transfers': 'Transactions',
        'health': 'Health',
        'docs': 'Documentation'
    };
    return nameMap[name] || name;
}

// Reorganize folders
const folders = collection.item || [];
const reorganized = [];
const adminItems = [];

// First pass: capitalize names and collect items
folders.forEach(folder => {
    if (folder.name === 'health' || folder.name === 'docs') {
        // Move these under Admin
        adminItems.push(...(folder.item || []));
    } else if (folder.name === 'admin') {
        // Collect existing admin items
        adminItems.push(...(folder.item || []));
    } else {
        // Capitalize and keep
        folder.name = capitalizeFolder(folder.name);
        reorganized.push(folder);
    }
});

// Add consolidated Admin folder
if (adminItems.length > 0) {
    reorganized.push({
        name: 'Admin',
        description: 'Administrative and system endpoints including health checks and documentation',
        item: adminItems
    });
}

// Sort folders in desired order
const folderOrder = ['Authentication', 'Accounts', 'Customers', 'Transactions', 'Admin'];
reorganized.sort((a, b) => {
    const aIndex = folderOrder.indexOf(a.name);
    const bIndex = folderOrder.indexOf(b.name);
    if (aIndex === -1 && bIndex === -1) return 0;
    if (aIndex === -1) return 1;
    if (bIndex === -1) return -1;
    return aIndex - bIndex;
});

// Update collection
collection.item = reorganized;

// Add collection-level info
collection.info = {
    name: 'Array Banking API - Complete Test Suite',
    description: 'Production-quality banking REST API for developer assessment and interviewing. This collection includes all endpoints with automated test scenarios covering authentication, account management, customer operations, transactions, and admin functions.',
    schema: 'https://schema.getpostman.com/json/collection/v2.1.0/collection.json',
    _postman_id: collection.info?._postman_id || require('crypto').randomUUID()
};

// Write back to file
fs.writeFileSync(collectionPath, JSON.stringify(collection, null, 2));

console.log('✅ Collection reorganized successfully!');
console.log('📁 Folder structure:');
reorganized.forEach(folder => {
    console.log(`   - ${folder.name} (${folder.item?.length || 0} items)`);
});
