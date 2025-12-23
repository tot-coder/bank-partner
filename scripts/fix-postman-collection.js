#!/usr/bin/env node

/**
 * Fix Postman Collection After Regeneration
 *
 * This script applies necessary fixes to the Postman collection that are lost
 * when regenerating from OpenAPI spec:
 *
 * 1. Removes individual request auth so they inherit from collection
 * 2. Replaces hardcoded values with Postman variables
 * 3. Sets up collection-level authentication
 */

const fs = require('fs');
const path = require('path');

const COLLECTION_PATH = path.join(__dirname, '../postman/Array-Banking-API.postman_collection.json');

// Variable replacements to apply
const VARIABLE_REPLACEMENTS = {
  // Auth request bodies (using regex to handle escaped newlines in raw strings)
  '\\"email\\": \\"<string>\\"': '\\"email\\": \\"{{test_user_email}}\\"',
  '\\"password\\": \\"<string>\\"': '\\"password\\": \\"{{test_user_password}}\\"',
  '\\"firstName\\": \\"<string>\\"': '\\"firstName\\": \\"{{test_user_first_name}}\\"',
  '\\"lastName\\": \\"<string>\\"': '\\"lastName\\": \\"{{test_user_last_name}}\\"',

  // Refresh token in request bodies
  '\\"refresh_token\\": \\"<string>\\"': '\\"refresh_token\\": \\"{{refresh_token}}\\"',
  '\\"refreshToken\\": \\"<string>\\"': '\\"refreshToken\\": \\"{{refresh_token}}\\"',

  // New user password in register endpoint
  '\\"new_password\\": \\"<string>\\"': '\\"new_password\\": \\"{{test_user_password}}\\"',

  // Current password in change password requests
  '\\"current_password\\": \\"<string>\\"': '\\"current_password\\": \\"{{test_user_password}}\\"',
};

// Endpoints that should NOT have authentication (they're public)
const PUBLIC_ENDPOINTS = [
  '/auth/register',
  '/auth/login',
  '/auth/refresh',
  '/health',
  '/docs',
];

function loadCollection() {
  console.log('📖 Loading Postman collection...');
  const content = fs.readFileSync(COLLECTION_PATH, 'utf8');
  return JSON.parse(content);
}

function saveCollection(collection) {
  console.log('💾 Saving Postman collection...');
  fs.writeFileSync(COLLECTION_PATH, JSON.stringify(collection, null, 2), 'utf8');
}

function isPublicEndpoint(url) {
  if (!url || !url.path) return false;
  const pathStr = Array.isArray(url.path) ? '/' + url.path.join('/') : url.path;
  return PUBLIC_ENDPOINTS.some(endpoint => pathStr.includes(endpoint));
}

function removeIndividualAuth(item) {
  let count = 0;

  function processItem(obj, path = '') {
    // Process request
    if (obj.request) {
      const url = obj.request.url;
      const isPublic = isPublicEndpoint(url);

      if (obj.request.auth) {
        if (isPublic) {
          // Public endpoints should have auth: { type: 'noauth' }
          obj.request.auth = { type: 'noauth' };
          console.log(`  ✓ Set noauth for public endpoint: ${obj.name}`);
        } else {
          // Protected endpoints should inherit from collection
          delete obj.request.auth;
          console.log(`  ✓ Removed auth from: ${obj.name}`);
        }
        count++;
      }
    }

    // Recursively process items in folders
    if (obj.item && Array.isArray(obj.item)) {
      obj.item.forEach(child => processItem(child, path + '/' + (obj.name || '')));
    }
  }

  processItem(item);
  return count;
}

function setCollectionAuth(collection) {
  console.log('\n🔐 Setting collection-level authentication...');

  collection.auth = {
    type: 'apikey',
    apikey: [
      {
        key: 'key',
        value: 'Authorization',
        type: 'string'
      },
      {
        key: 'value',
        value: 'Bearer {{access_token}}',
        type: 'string'
      },
      {
        key: 'in',
        value: 'header',
        type: 'string'
      }
    ]
  };

  console.log('  ✓ Collection-level auth configured with {{access_token}}');
}

function replaceHardcodedValues(collection) {
  console.log('\n📝 Replacing hardcoded values with variables...');

  let jsonStr = JSON.stringify(collection);
  let replacementCount = 0;

  for (const [find, replace] of Object.entries(VARIABLE_REPLACEMENTS)) {
    const regex = new RegExp(find.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g');
    const matches = (jsonStr.match(regex) || []).length;
    if (matches > 0) {
      jsonStr = jsonStr.replace(regex, replace);
      replacementCount += matches;
      console.log(`  ✓ Replaced "${find}" → "${replace}" (${matches} occurrences)`);
    }
  }

  console.log(`\n  Total replacements: ${replacementCount}`);
  return JSON.parse(jsonStr);
}

function addCollectionVariables(collection) {
  console.log('\n🔧 Setting up collection variables...');

  const variables = [
    {
      key: 'access_token',
      value: '',
      type: 'string',
      description: 'JWT access token (set by login test script)'
    },
    {
      key: 'refresh_token',
      value: '',
      type: 'string',
      description: 'JWT refresh token (set by login test script)'
    },
    {
      key: 'test_user_email',
      value: 'john.doe@example.com',
      type: 'string',
      description: 'Test user email for authentication'
    },
    {
      key: 'test_user_password',
      value: 'Password123!',
      type: 'string',
      description: 'Test user password for authentication'
    }
  ];

  // Merge with existing variables, keeping user-defined ones
  const existingVars = collection.variable || [];
  const existingKeys = new Set(existingVars.map(v => v.key));

  const newVars = variables.filter(v => !existingKeys.has(v.key));
  collection.variable = [...existingVars, ...newVars];

  console.log(`  ✓ Added ${newVars.length} collection variables`);
  newVars.forEach(v => console.log(`    - ${v.key}: ${v.description}`));
}

function main() {
  console.log('🔧 Fixing Postman Collection\n');
  console.log('=' .repeat(60));

  try {
    // Load collection
    const collection = loadCollection();

    // Apply fixes
    console.log('\n🗑️  Removing individual request authentication...');
    const authCount = removeIndividualAuth(collection);
    console.log(`\n  Total requests processed: ${authCount}`);

    setCollectionAuth(collection);
    addCollectionVariables(collection);

    const updatedCollection = replaceHardcodedValues(collection);

    // Save collection
    saveCollection(updatedCollection);

    console.log('\n' + '='.repeat(60));
    console.log('✅ Postman collection fixed successfully!\n');

  } catch (error) {
    console.error('\n❌ Error fixing collection:', error.message);
    process.exit(1);
  }
}

main();
