const express = require('express');
const app = express();
const PORT = process.env.BACKEND_PORT || 3000;

app.use(express.json());
app.get('/', (req, res) => res.json({ message: 'Hello from dev-box API (Node.js)' }));
app.get('/health', (req, res) => res.json({ status: 'ok' }));
app.get('/metrics', (req, res) => {
  res.type('text/plain').send('# HELP devbox_up Whether the API process is running.\n# TYPE devbox_up gauge\ndevbox_up 1\n');
});
app.listen(PORT, '0.0.0.0', () => console.log('Server running on port ' + PORT));
