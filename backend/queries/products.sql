-- name: GetProduct :one
SELECT * FROM products WHERE id = $1;

-- name: ListProducts :many
SELECT * FROM products ORDER BY name;

-- name: CreateProduct :one
INSERT INTO products (name, name_el, category, manufacturer, requires_prescription)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;
