#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

const collectionPath = path.join(__dirname, '../postman/Array-Banking-API.postman_collection.json');

// Read the collection
const collection = JSON.parse(fs.readFileSync(collectionPath, 'utf8'));

// Helper to find and update test scripts
function addPerformanceAssertion(testScript, threshold, operation) {
    const lines = testScript.split('\n');

    // Check if performance test already exists
    const hasPerformanceTest = lines.some(line =>
        line.includes('Response time') || line.includes('responseTime')
    );

    if (hasPerformanceTest) {
        return testScript; // Already has performance assertion
    }

    // Add performance assertion at the end
    const performanceTest = `
pm.test("Response time is acceptable for ${operation}", function () {
    pm.expect(pm.response.responseTime).to.be.below(${threshold});
});`;

    return testScript + performanceTest;
}

// Recursively process all items
function processItems(items) {
    let updatedCount = 0;

    for (const item of items) {
        // Process nested folders
        if (item.item && item.item.length > 0) {
            updatedCount += processItems(item.item);
        }

        // Process request with tests
        if (item.request && item.event) {
            const testEvent = item.event.find(e => e.listen === 'test');

            if (testEvent && testEvent.script && testEvent.script.exec) {
                const method = item.request.method;
                const name = item.name.toLowerCase();

                // Determine threshold based on operation type
                let threshold = 500; // Default for writes
                let operation = 'write operations';

                if (method === 'GET') {
                    if (name.includes('health')) {
                        threshold = 100;
                        operation = 'health check';
                    } else {
                        threshold = 200;
                        operation = 'read operations';
                    }
                } else if (method === 'POST' || method === 'PUT' || method === 'PATCH') {
                    threshold = 500;
                    operation = 'write operations';
                } else if (method === 'DELETE') {
                    threshold = 300;
                    operation = 'delete operations';
                }

                // Get current test script
                const currentScript = testEvent.script.exec.join('\n');

                // Add performance assertion
                const updatedScript = addPerformanceAssertion(currentScript, threshold, operation);

                if (updatedScript !== currentScript) {
                    testEvent.script.exec = updatedScript.split('\n');
                    updatedCount++;
                }
            }
        }
    }

    return updatedCount;
}

// Process all items in collection
const updatedCount = processItems(collection.item);

// Write back to file
fs.writeFileSync(collectionPath, JSON.stringify(collection, null, 2));

console.log(`✅ Added performance assertions to ${updatedCount} endpoints`);
console.log("\n📊 Performance Benchmarks:");
console.log("   - Health check: < 100ms");
console.log("   - Read operations (GET): < 200ms");
console.log("   - Write operations (POST/PUT): < 500ms");
console.log("   - Delete operations: < 300ms");
console.log("\n💡 All endpoints now include response time validation!");
