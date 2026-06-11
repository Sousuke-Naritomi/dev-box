const express = require('express');
const app = express();
const PORT = process.env.BACKEND_PORT || 3000;

app.use(express.json());
app.get('/', (req, res) => res.json({ message: 'Hello from dev-box API (Node.js)' }));
app.get('/health', (req, res) => res.json({ status: 'ok' }));
app.listen(PORT, '0.0.0.0', () => console.log('Server running on port ' + PORT));
