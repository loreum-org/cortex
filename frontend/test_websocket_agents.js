import WebSocket from 'ws';

const ws = new WebSocket('ws://localhost:4891/ws');

ws.on('open', function open() {
    console.log('🔌 Connected to WebSocket');
    
    // Send get_agents request
    const request = {
        type: 'get_agents',
        id: 'test_agents_1234',
        data: {},
        timestamp: new Date().toISOString()
    };
    
    console.log('📤 Sending request:', JSON.stringify(request, null, 2));
    ws.send(JSON.stringify(request));
});

ws.on('message', function message(data) {
    const msg = JSON.parse(data.toString());
    console.log('📨 Received message:', JSON.stringify(msg, null, 2));
    
    if (msg.id === 'test_agents_1234') {
        console.log('✅ Got response for our agents request!');
        ws.close(1000, 'Normal closure');
    }
});

ws.on('error', function error(err) {
    console.error('❌ WebSocket error:', err);
});

ws.on('close', function close() {
    console.log('🔌 WebSocket closed');
    process.exit(0);
});

// Timeout after 10 seconds
setTimeout(() => {
    console.log('⏰ Test timed out');
    ws.close(1000, 'Timeout');
    process.exit(1);
}, 10000);