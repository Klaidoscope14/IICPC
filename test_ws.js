const WebSocket = require('ws');
const ws = new WebSocket('ws://localhost:8082/ws/benchmarks/91640f7d-cc4b-4238-a509-a4fc01110fe3/stream');

ws.on('open', function open() {
  console.log('connected');
});

ws.on('message', function incoming(data) {
  console.log('received: %s', data);
});

ws.on('error', function error(e) {
  console.log('error: ', e);
});
