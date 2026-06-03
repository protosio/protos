CREATE TABLE machines (
    id VARCHAR(255) NOT NULL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    kind VARCHAR(255) NOT NULL,
    desired_status VARCHAR(255),
    witness_rank INT NOT NULL
);

CREATE INDEX machines_name_idx ON machines (name);

CREATE TABLE cloud_machines_metadata (
    id VARCHAR(255) NOT NULL PRIMARY KEY,
    cloud_id VARCHAR(255) NOT NULL,
    provider_resource_id VARCHAR(255),
    public_ip VARCHAR(255) NOT NULL,
    location VARCHAR(255) NOT NULL,
    architecture VARCHAR(255) NOT NULL,
    public_key VARCHAR(255) NOT NULL
);

CREATE INDEX cloud_machines_metadata_public_key_idx ON cloud_machines_metadata (public_key);

CREATE TABLE cloud_providers (
    id VARCHAR(255) NOT NULL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(255) NOT NULL,
    auth JSON NOT NULL
);

CREATE INDEX cloud_providers_name_idx ON cloud_providers (name);

CREATE TABLE apps (
    id VARCHAR(255) NOT NULL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    installer_ref VARCHAR(255) NOT NULL,
    instance_id VARCHAR(255) NOT NULL,
    desired_status VARCHAR(255) NOT NULL,
    persistence TINYINT(1) NOT NULL,
    public_key VARCHAR(255) NOT NULL
);

CREATE INDEX apps_name_idx ON apps (name);

CREATE INDEX apps_public_key_idx ON apps (public_key);

CREATE TABLE users (
    username VARCHAR(255) NOT NULL PRIMARY KEY,
    name VARCHAR(255),
    is_disabled TINYINT(1) NOT NULL
);

CREATE TABLE user_devices_metadata (
    id VARCHAR(255) NOT NULL PRIMARY KEY,
    public_key VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    witness_rank INT NOT NULL
);

CREATE INDEX user_devices_metadata_public_key_idx ON user_devices_metadata (public_key);

CREATE INDEX user_devices_metadata_user_id_idx ON user_devices_metadata (user_id);

CREATE TABLE peers (
    id VARCHAR(255) NOT NULL PRIMARY KEY
);

CREATE TABLE tasks (
    id VARCHAR(255) NOT NULL PRIMARY KEY,
    task_stream VARCHAR(255) NOT NULL,
    subject_type VARCHAR(255) NOT NULL,
    subject_id VARCHAR(255) NOT NULL,
    status VARCHAR(64) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    progress INT NOT NULL,
    payload JSON NOT NULL,
    result JSON,
    error_message TEXT,
    attempts INT NOT NULL,
    max_attempts INT NOT NULL,
    created_at VARCHAR(64) NOT NULL,
    updated_at VARCHAR(64) NOT NULL,
    started_at VARCHAR(64),
    finished_at VARCHAR(64)
);

CREATE INDEX tasks_task_stream_idx ON tasks (task_stream);

CREATE INDEX tasks_subject_type_idx ON tasks (subject_type);

CREATE INDEX tasks_subject_id_idx ON tasks (subject_id);

CREATE INDEX tasks_status_idx ON tasks (status);

CREATE TABLE task_events (
    id VARCHAR(255) NOT NULL PRIMARY KEY,
    task_id VARCHAR(255) NOT NULL,
    status VARCHAR(64) NOT NULL,
    message TEXT NOT NULL,
    progress INT NOT NULL,
    details JSON,
    created_at VARCHAR(64) NOT NULL
);

CREATE INDEX task_events_task_id_idx ON task_events (task_id);

CREATE INDEX task_events_status_idx ON task_events (status);

CREATE TABLE exit_routes (
    id VARCHAR(255) NOT NULL PRIMARY KEY,
    device_id VARCHAR(255) NOT NULL,
    instance_id VARCHAR(255) NOT NULL,
    desired_status VARCHAR(255) NOT NULL,
    dns_server VARCHAR(255) NOT NULL,
    cidrs TEXT NOT NULL
);

CREATE INDEX exit_routes_device_id_idx ON exit_routes (device_id);

CREATE INDEX exit_routes_instance_id_idx ON exit_routes (instance_id);
