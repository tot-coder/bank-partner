#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const collectionPath = path.join(__dirname, '../postman/Array-Banking-API.postman_collection.json');

// Read the collection
const collection = JSON.parse(fs.readFileSync(collectionPath, 'utf8'));

// Collection-level pre-request script for token management
const preRequestScript = `
// Collection-level pre-request script for automatic token refresh
const accessToken = pm.environment.get("access_token");
const tokenExpiry = pm.environment.get("token_expiry");
const refreshToken = pm.environment.get("refresh_token");

// Check if we need to refresh the token (if it's expired or about to expire in 1 minute)
const now = Date.now();
const expiryTime = parseInt(tokenExpiry) || 0;

if (accessToken && refreshToken && expiryTime > 0 && (now >= expiryTime - 60000)) {
    console.log("🔄 Token expired or expiring soon, attempting refresh...");

    // Note: In a real scenario, you would make a refresh token request here
    // For now, we'll just log the need to refresh
    // Users should manually re-login if tokens expire
    console.log("⚠️  Please re-authenticate using the Login endpoint");
}

// Set Authorization header if token exists
if (accessToken) {
    pm.request.headers.add({
        key: 'Authorization',
        value: \`Bearer \${accessToken}\`
    });
}
`.trim();

// Collection-level test script (optional)
const testScript = `
// Collection-level test script
// This runs after every request in the collection

// Log response time for performance monitoring
const responseTime = pm.response.responseTime;
console.log(\`⏱️  Response time: \${responseTime}ms\`);

// Check for common error patterns
if (pm.response.code === 401) {
    console.log("🔒 Authentication required or token expired");
    console.log("💡 Please run the Login endpoint to get a new token");
}
`.trim();

// Add event scripts to collection
collection.event = [
    {
        "listen": "prerequest",
        "script": {
            "type": "text/javascript",
            "exec": preRequestScript.split('\n')
        }
    },
    {
        "listen": "test",
        "script": {
            "type": "text/javascript",
            "exec": testScript.split('\n')
        }
    }
];

// Now add login-specific test script to capture tokens
// Find the login request in the Authentication folder
function findLoginRequest(items) {
    for (const item of items) {
        if (item.name === 'Authentication') {
            // Found the Authentication folder
            for (const authItem of item.item || []) {
                // Check if this is the login subfolder
                if (authItem.name && authItem.name.toLowerCase() === 'login') {
                    // Look inside the login subfolder for the actual request
                    for (const loginItem of authItem.item || []) {
                        if (loginItem.request?.method === 'POST') {
                            return loginItem;
                        }
                    }
                }
                // Also check if authItem is directly a POST login request
                if (authItem.name && authItem.name.toLowerCase().includes('login') &&
                    authItem.request?.method === 'POST') {
                    return authItem;
                }
            }
        }
    }
    return null;
}

const loginRequest = findLoginRequest(collection.item || []);

if (loginRequest) {
    // Add test script to login request
    const loginTestScript = `
// Login endpoint - Capture and store authentication tokens

pm.test("Status code is 200 OK", function () {
    pm.response.to.have.status(200);
});

pm.test("Response contains access token", function () {
    const jsonData = pm.response.json();
    // API returns either accessToken or access_token
    const hasToken = jsonData.accessToken || jsonData.access_token;
    pm.expect(hasToken).to.be.a("string").and.not.empty;
});

pm.test("Response contains refresh token", function () {
    const jsonData = pm.response.json();
    // API returns either refreshToken or refresh_token
    const hasRefreshToken = jsonData.refreshToken || jsonData.refresh_token;
    pm.expect(hasRefreshToken).to.be.a("string").and.not.empty;
});

pm.test("Token is valid JWT format", function () {
    const jsonData = pm.response.json();
    const accessToken = jsonData.accessToken || jsonData.access_token;
    const jwtRegex = /^[A-Za-z0-9-_]+\\.[A-Za-z0-9-_]+\\.[A-Za-z0-9-_]+$/;
    pm.expect(accessToken).to.match(jwtRegex, "Access token should be valid JWT");
});

// If login successful, store tokens in environment
if (pm.response.code === 200) {
    const jsonData = pm.response.json();

    // Store access token (handle both camelCase and snake_case)
    const accessToken = jsonData.accessToken || jsonData.access_token;
    pm.environment.set("access_token", accessToken);
    console.log("✅ Access token stored in environment");

    // Store refresh token
    const refreshToken = jsonData.refreshToken || jsonData.refresh_token;
    pm.environment.set("refresh_token", refreshToken);
    console.log("✅ Refresh token stored in environment");

    // Calculate and store token expiry time
    let expiresIn = 3600; // Default 1 hour

    if (jsonData.expiresAt) {
        // If expiresAt is provided, calculate seconds until expiry
        const expiryDate = new Date(jsonData.expiresAt);
        expiresIn = Math.floor((expiryDate.getTime() - Date.now()) / 1000);
    } else if (jsonData.expires_in) {
        expiresIn = jsonData.expires_in;
    }

    const expiryTime = Date.now() + (expiresIn * 1000);
    pm.environment.set("token_expiry", expiryTime.toString());
    console.log(\`✅ Token will expire in \${expiresIn} seconds\`);

    console.log("🎉 Authentication successful! You can now use other endpoints.");
}
    `.trim();

    loginRequest.event = loginRequest.event || [];
    loginRequest.event.push({
        "listen": "test",
        "script": {
            "type": "text/javascript",
            "exec": loginTestScript.split('\n')
        }
    });

    console.log("✅ Added test script to Login endpoint");
}

// Write back to file
fs.writeFileSync(collectionPath, JSON.stringify(collection, null, 2));

console.log("✅ Collection updated with authentication scripts!");
console.log("📝 Added collection-level pre-request script for token management");
console.log("📝 Added collection-level test script for response monitoring");
if (loginRequest) {
    console.log("📝 Added login-specific test script to capture tokens");
}
