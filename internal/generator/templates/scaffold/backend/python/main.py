"""Dev-box API server (Python/Flask)."""
import os
from flask import Flask, jsonify

app = Flask(__name__)

@app.route("/")
def index():
    return jsonify({"message": "Hello from dev-box API (Python)"})

@app.route("/health")
def health():
    return jsonify({"status": "ok"})

if __name__ == "__main__":
    port = int(os.environ.get("BACKEND_PORT", 8000))
    app.run(host="0.0.0.0", port=port, debug=True)
