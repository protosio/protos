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
