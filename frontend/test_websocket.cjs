const WebSocket = require('ws');

// Test WebSocket connection to verify handlers work
async function testWebSocketHandlers() {
    console.log('🧪 Testing WebSocket message handlers...');
    
    return new Promise((resolve, reject) => {
        const ws = new WebSocket('ws://localhost:4891/ws');
        let timeout = setTimeout(() => {
            console.log('❌ Connection timeout');
            ws.close();
            reject(new Error('Connection timeout'));
        }, 5000);

        ws.on('open', () => {
            console.log('✅ WebSocket connected');
            clearTimeout(timeout);
            
            // Send a simple test message
            const testMessage = {
                type: 'request',
                method: 'getStatus',
                id: 'test_001',
                data: {},
                timestamp: new Date().toISOString()
            };
            
            console.log('📤 Sending test message:', JSON.stringify(testMessage, null, 2));
            ws.send(JSON.stringify(testMessage));
            
            // Set timeout for response
            timeout = setTimeout(() => {
                console.log('❌ Response timeout');
                ws.close();
                reject(new Error('Response timeout'));
            }, 10000);
        });

        ws.on('message', (data) => {
            console.log('📥 Received response:', data.toString());
            clearTimeout(timeout);
            ws.close();
            resolve('✅ WebSocket message handlers working correctly!');
        });

        ws.on('error', (error) => {
            console.log('❌ WebSocket error:', error.message);
            clearTimeout(timeout);
            reject(error);
        });

        ws.on('close', () => {
            console.log('🔌 WebSocket connection closed');
        });
    });
}

// Run test if this script is executed directly
if (require.main === module) {
    testWebSocketHandlers()
        .then(result => {
            console.log('\n🎉 Test Result:', result);
            process.exit(0);
        })
        .catch(error => {
            console.error('\n💥 Test Failed:', error.message);
            process.exit(1);
        });
}

module.exports = { testWebSocketHandlers };