"""Dev-box API server (Python/Flask)."""
import os
from flask import Flask, Response, jsonify

app = Flask(__name__)

@app.route("/")
def index():
    return jsonify({"message": "Hello from dev-box API (Python)"})

@app.route("/health")
def health():
    return jsonify({"status": "ok"})

@app.route("/metrics")
def metrics():
    return Response("# HELP devbox_up Whether the API process is running.\n# TYPE devbox_up gauge\ndevbox_up 1\n", mimetype="text/plain")

if __name__ == "__main__":
    port = int(os.environ.get("BACKEND_PORT", 8000))
    debug = os.environ.get("FLASK_DEBUG", "false").lower() == "true"
    app.run(host="0.0.0.0", port=port, debug=debug)
