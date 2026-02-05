-- name: get-all
SELECT COALESCE(JSON_OBJECT_AGG(key, value), '{}'::json) AS settings FROM (SELECT * FROM settings ORDER BY key) t;

-- name: update
INSERT INTO settings (key, value, updated_at)
SELECT key, value, now()
FROM jsonb_each($1)
ON CONFLICT (key) DO UPDATE 
SET value = EXCLUDED.value,
    updated_at = now();

-- name: get-by-prefix
SELECT COALESCE(JSON_OBJECT_AGG(key, value), '{}'::json) AS settings 
FROM settings 
WHERE key LIKE $1;

-- name: get
SELECT value FROM settings WHERE key = $1;
