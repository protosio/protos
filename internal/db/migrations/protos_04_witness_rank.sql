ALTER TABLE machines
ADD COLUMN witness_rank INT NOT NULL DEFAULT 0;

UPDATE machines
SET witness_rank = CASE
    WHEN LOWER(kind) = 'local_vm' THEN 30
    WHEN LOWER(kind) = 'cloud_vm' THEN 100
    ELSE 0
END
WHERE witness_rank = 0;

ALTER TABLE user_devices_metadata
ADD COLUMN witness_rank INT NOT NULL DEFAULT 0;

UPDATE user_devices_metadata
SET witness_rank = CASE
    WHEN LOWER(name) LIKE '%phone%' THEN 10
    WHEN LOWER(name) LIKE '%iphone%' THEN 10
    WHEN LOWER(name) LIKE '%android%' THEN 10
    WHEN LOWER(name) LIKE '%mobile%' THEN 10
    ELSE 50
END
WHERE witness_rank = 0;
