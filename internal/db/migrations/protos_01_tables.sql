CREATE TABLE instances (
    vm_id VARCHAR(255) NOT NULL
    ,name VARCHAR(255) NOT NULL
    ,ssh_key_seed VARCHAR(255) NOT NULL
    ,public_key VARCHAR(255) NOT NULL
    ,public_ip VARCHAR(255) NOT NULL
    ,cloud_type VARCHAR(255) NOT NULL
    ,cloud_name VARCHAR(255) NOT NULL
    ,location VARCHAR(255) NOT NULL
    ,protos_version VARCHAR(255) NOT NULL
    ,architecture VARCHAR(255) NOT NULL

    ,PRIMARY KEY (vm_id)
);

CREATE INDEX instances_name_idx ON instances (name);

CREATE TABLE cloud_providers (
    id VARCHAR(255) NOT NULL
    ,name VARCHAR(255) NOT NULL
    ,type VARCHAR(255) NOT NULL
    ,auth JSON NOT NULL

    ,PRIMARY KEY (id)
);

CREATE INDEX cloud_providers_name_idx ON cloud_providers (name);

CREATE TABLE ssh_keys (
    private VARCHAR(255) NOT NULL
    ,public VARCHAR(255) NOT NULL

    ,PRIMARY KEY (private)
);

CREATE TABLE apps (
    id VARCHAR(255) NOT NULL
    ,name VARCHAR(255) NOT NULL
    ,installer_ref VARCHAR(255) NOT NULL
    ,instance_name VARCHAR(255) NOT NULL
    ,desired_status VARCHAR(255) NOT NULL
    ,ip VARCHAR(255) NOT NULL
    ,persistence BOOLEAN NOT NULL

    ,PRIMARY KEY (id)
);

CREATE INDEX apps_name_idx ON apps (name);

CREATE TABLE users (
    username VARCHAR(255) NOT NULL
    ,name VARCHAR(255)
    ,is_disabled BOOLEAN NOT NULL

    ,PRIMARY KEY (username)
);

CREATE TABLE user_devices (
    id VARCHAR(255) NOT NULL
    ,name VARCHAR(255) NOT NULL
    ,public_key VARCHAR(255) NOT NULL
    ,network VARCHAR(255) NOT NULL
    ,user_id VARCHAR(255) NOT NULL

    ,PRIMARY KEY (id)
);

CREATE INDEX user_devices_name_idx ON user_devices (name);

CREATE INDEX user_devices_user_id_idx ON user_devices (user_id);
