CREATE TABLE organisations (
    id VARCHAR(64) NOT NULL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at VARCHAR(64) NOT NULL
);

CREATE INDEX organisations_name_idx ON organisations (name);
