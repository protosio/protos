ALTER TABLE cloud_machines_metadata
    ADD COLUMN lifecycle_owner_peer_id VARCHAR(255) NOT NULL DEFAULT '';

UPDATE cloud_machines_metadata AS metadata
JOIN (
	SELECT subject_id, MIN(TRIM(author_peer_id)) AS owner_peer_id
	FROM task_operation_facts
	WHERE fact_kind = 'instance_peer_drain_authorized_v1'
	  AND subject_type = 'instance'
	  AND TRIM(author_peer_id) <> ''
	GROUP BY subject_id
	HAVING COUNT(DISTINCT TRIM(author_peer_id)) = 1
) AS authorized_owner
	ON authorized_owner.subject_id = BIN_TO_UUID(metadata.id)
SET metadata.lifecycle_owner_peer_id = authorized_owner.owner_peer_id
WHERE metadata.lifecycle_owner_peer_id = '';

UPDATE cloud_machines_metadata AS metadata
JOIN (
    SELECT subject_id, MIN(TRIM(owner_peer_id)) AS owner_peer_id
    FROM tasks
    WHERE task_stream = 'provisioners.instance.deploy'
      AND subject_type = 'instance'
      AND TRIM(owner_peer_id) <> ''
    GROUP BY subject_id
    HAVING COUNT(DISTINCT TRIM(owner_peer_id)) = 1
) AS historical_owner
    ON historical_owner.subject_id = BIN_TO_UUID(metadata.id)
SET metadata.lifecycle_owner_peer_id = historical_owner.owner_peer_id
WHERE metadata.lifecycle_owner_peer_id = ''
  AND NOT EXISTS (
	SELECT 1
	FROM task_operation_facts AS authorization
	WHERE authorization.fact_kind = 'instance_peer_drain_authorized_v1'
	  AND authorization.subject_type = 'instance'
	  AND authorization.subject_id = BIN_TO_UUID(metadata.id)
  );
