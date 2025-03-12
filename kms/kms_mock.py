from flask import Flask, request, jsonify
import secrets
import base64

app = Flask(__name__)
keys = {}

@app.route('/', methods=['POST'])
def key_management():
  data = request.get_json()
  print(data)

  # sanitize
  if not data or 'path' not in data or 'keylen' not in data:
    return jsonify({"error": "Invalid input. Expected JSON with 'path' and 'keylen'."}), 400
  if not isinstance(data['keylen'], int) or data['keylen'] <= 0:
    return jsonify({"error": "'keylen' must be a positive integer."}), 400

  path = data['path']
  keylen = data['keylen']

  key_id = (path, keylen)

  if key_id in keys:
    key = keys[key_id]
  else:
    key = secrets.token_bytes(keylen)
    keys[key_id] = key

  # base64 encode for resp
  b64_key = base64.b64encode(key).decode('utf-8')
  print("DEBUG:", path, keylen, b64_key)
  return jsonify({"key": b64_key})

if __name__ == '__main__':
    app.run(debug=True)

