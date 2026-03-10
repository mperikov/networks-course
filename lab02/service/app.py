from dataclasses import asdict, dataclass
from threading import Lock
from typing import Dict, Optional

from flask import Flask, jsonify, request, send_file
import os


app = Flask(__name__)


@dataclass
class Product:
    id: int
    name: str
    description: str
    icon: Optional[str] = None


products: Dict[int, Product] = {}
products_lock = Lock()
next_product_id = 0


def error_response(message: str, status_code: int):
    return jsonify({"error": message}), status_code


def get_json_body() -> Optional[dict]:
    if not request.is_json:
        return None
    data = request.get_json(silent=True)
    if not isinstance(data, dict):
        return None
    return data


@app.post("/product")
def create_product():
    global next_product_id

    data = get_json_body()
    if data is None:
        return error_response("Request body must be a valid JSON object.", 400)

    name = data.get("name")
    description = data.get("description")

    if not isinstance(name, str) or not name.strip():
        return error_response("Field 'name' is required and must be a non-empty string.", 400)
    if not isinstance(description, str) or not description.strip():
        return error_response("Field 'description' is required and must be a non-empty string.", 400)

    with products_lock:
        product_id = next_product_id
        next_product_id += 1
        product = Product(id=product_id, name=name.strip(), description=description)
        products[product_id] = product

    return jsonify(asdict(product)), 201


@app.get("/product/<int:product_id>")
def get_product(product_id: int):
    product = products.get(product_id)
    if product is None:
        return error_response(f"Product with id={product_id} not found.", 404)
    return jsonify(asdict(product)), 200


@app.put("/product/<int:product_id>")
def update_product(product_id: int):
    with products_lock:
        product = products.get(product_id)
        if product is None:
            return error_response(f"Product with id={product_id} not found.", 404)

        data = get_json_body()
        if data is None:
            return error_response("Request body must be a valid JSON object.", 400)
        if not data:
            return error_response("At least one field must be provided for update.", 400)

        if "id" in data and data["id"] != product_id:
            return error_response("Field 'id' cannot be changed.", 400)

        if "name" in data:
            name = data["name"]
            if not isinstance(name, str) or not name.strip():
                return error_response("Field 'name' must be a non-empty string.", 400)
            product.name = name.strip()

        if "description" in data:
            description = data["description"]
            if not isinstance(description, str) or not description.strip():
                return error_response("Field 'description' must be a non-empty string.", 400)
            product.description = description

    return jsonify(asdict(product)), 200


@app.delete("/product/<int:product_id>")
def delete_product(product_id: int):
    with products_lock:
        product = products.pop(product_id, None)
    if product is None:
        return error_response(f"Product with id={product_id} not found.", 404)
    return jsonify(asdict(product)), 200


@app.get("/products")
def list_products():
    result = [asdict(product) for product in products.values()]
    return jsonify(result), 200


@app.post("/product/<int:product_id>/image")
def upload_product_image(product_id: int):
    with products_lock:
        product = products.get(product_id)
        if product is None:
            return error_response(f"Product with id={product_id} not found.", 404)

        file = request.files["icon"]

        if file is None:
            return error_response("Image payload is empty.", 400)

        filename = file.filename
        os.makedirs("data", exist_ok=True)
        file_path = os.path.join("data", filename)
        file.save(file_path)

        product.icon = filename

    return jsonify(asdict(product)), 200


@app.get("/product/<int:product_id>/image")
def get_product_image(product_id: int):
    with products_lock:
        product = products.get(product_id)
        if product is None:
            return error_response(f"Product with id={product_id} not found.", 404)

        filename = product.icon
        if filename is None:
            return error_response(f"Image for product id={product_id} not found.", 404)

        file_path = os.path.join("data", filename)

        if not os.path.isfile(file_path):
            return error_response(f"Image for product id={product_id} not found.", 404)

        return send_file(file_path, as_attachment=True, download_name=filename)


if __name__ == "__main__":
    app.run(host="127.0.0.1", port=8000, debug=True)