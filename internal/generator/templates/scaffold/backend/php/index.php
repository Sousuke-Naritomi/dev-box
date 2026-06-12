<?php
header('Content-Type: application/json');
$path = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
switch ($path) {
    case '/': echo json_encode(['message' => 'Hello from dev-box API (PHP)']); break;
    case '/health': echo json_encode(['status' => 'ok']); break;
    case '/metrics':
        header('Content-Type: text/plain; version=0.0.4');
        echo "# HELP devbox_up Whether the API process is running.\n# TYPE devbox_up gauge\ndevbox_up 1\n";
        break;
    default: http_response_code(404); echo json_encode(['error' => 'Not Found']); break;
}
