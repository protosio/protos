package provisioners

import (
	"github.com/bokwoon95/sq"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/tasks"
)

//
// Instance
//

func createInstanceInsertMapper(instance InstanceInfo) (db.InsertMapper, db.InsertMapper) {

	machineMapper := func() sq.InsertQuery {
		m := sq.New[db.MACHINE]("")
		mapper := func(col *sq.Column) {
			col.SetBytes(m.ID, db.MustUUIDBytes(instance.ID))
			col.SetString(m.NAME, instance.Name)
			col.SetString(m.KIND, instance.Kind)
			col.SetString(m.DESIRED_STATUS, instance.DesiredStatus)
			col.SetInt(m.REPLICATION_PRIORITY, instance.ReplicationPriority)
		}

		return sq.InsertInto(m).ColumnValues(mapper)
	}

	cloudMachineMetadataMapper := func() sq.InsertQuery {
		cmm := sq.New[db.CLOUD_MACHINE_METADATA]("")
		mapper := func(col *sq.Column) {
			col.SetBytes(cmm.ID, db.MustUUIDBytes(instance.ID))
			col.SetString(cmm.CLOUD_ID, instance.KindID)
			col.SetString(cmm.PROVIDER_RESOURCE_ID, instance.ProviderResourceID)
			col.SetString(cmm.PUBLIC_IP, instance.PublicIP)
			col.SetString(cmm.LOCATION, instance.Location)
			col.SetString(cmm.ARCHITECTURE, instance.Architecture)
			col.SetString(cmm.PUBLIC_KEY, instance.PublicKey)
			col.SetString(cmm.LIFECYCLE_OWNER_PEER_ID, instance.LifecycleOwnerPeerID)
		}

		return sq.InsertInto(cmm).ColumnValues(mapper)
	}

	return machineMapper, cloudMachineMetadataMapper
}

// createInstancePeerDrainAuthorizationUpdateMapper is P's exact CAS. It
// changes only desired_status, matches the complete machine/provider row
// captured in the original task, and requires P's preceding immutable fact.
func createInstancePeerDrainAuthorizationUpdateMapper(expected InstanceInfo, fact tasks.OperationFact) db.UpdateMapper {
	return func() sq.UpdateQuery {
		m := sq.New[db.MACHINE]("")
		cmm := sq.New[db.CLOUD_MACHINE_METADATA]("")
		facts := sq.New[db.TASK_OPERATION_FACT]("")
		return sq.Update(m).Set(m.DESIRED_STATUS.SetString(ServerStateDeleting)).Where(
			db.UUIDEq(m.ID, expected.ID),
			m.NAME.EqString(expected.Name),
			m.KIND.EqString(expected.Kind),
			m.DESIRED_STATUS.EqString(expected.DesiredStatus),
			m.REPLICATION_PRIORITY.EqInt(expected.ReplicationPriority),
			sq.Exists(sq.SelectOne().From(cmm).Where(
				cmm.ID.Eq(m.ID),
				cmm.CLOUD_ID.EqString(expected.KindID),
				cmm.PROVIDER_RESOURCE_ID.EqString(expected.ProviderResourceID),
				cmm.PUBLIC_IP.EqString(expected.PublicIP),
				cmm.LOCATION.EqString(expected.Location),
				cmm.ARCHITECTURE.EqString(expected.Architecture),
				cmm.PUBLIC_KEY.EqString(expected.PublicKey),
				cmm.LIFECYCLE_OWNER_PEER_ID.EqString(expected.LifecycleOwnerPeerID),
			)),
			sq.Exists(sq.SelectOne().From(facts).Where(
				facts.ID.EqString(fact.ID),
				facts.OPERATION_KEY.EqString(fact.OperationKey),
				facts.INTENT_DIGEST.EqString(fact.IntentDigest),
				facts.AUTHOR_PEER_ID.EqString(fact.AuthorPeerID),
			)),
		)
	}
}

// createInstancePeerDrainAuthorizationFactCASMapper inserts P's immutable fact
// only when its exact pre-authorization machine/provider row still exists.
func createInstancePeerDrainAuthorizationFactCASMapper(fact tasks.OperationFact, expected InstanceInfo) db.InsertMapper {
	return func() sq.InsertQuery {
		table := sq.New[db.TASK_OPERATION_FACT]("")
		m := sq.New[db.MACHINE]("")
		cmm := sq.New[db.CLOUD_MACHINE_METADATA]("")
		return sq.InsertInto(table).
			Columns(
				table.ID,
				table.TASK_ID,
				table.FACT_KIND,
				table.OPERATION_KEY,
				table.INTENT_DIGEST,
				table.AUTHOR_PEER_ID,
				table.SUBJECT_TYPE,
				table.SUBJECT_ID,
				table.PAYLOAD,
			).
			Select(sq.Select(
				sq.Expr("{}", fact.ID),
				sq.Expr("{}", db.MustUUIDBytes(fact.TaskID)),
				sq.Expr("{}", fact.Kind),
				sq.Expr("{}", fact.OperationKey),
				sq.Expr("{}", fact.IntentDigest),
				sq.Expr("{}", fact.AuthorPeerID),
				sq.Expr("{}", fact.SubjectType),
				sq.Expr("{}", fact.SubjectID),
				sq.Expr("{}", sq.JSONValue(fact.Payload)),
			).From(m).Join(cmm, cmm.ID.Eq(m.ID)).Where(
				db.UUIDEq(m.ID, expected.ID),
				m.NAME.EqString(expected.Name),
				m.KIND.EqString(expected.Kind),
				m.DESIRED_STATUS.EqString(expected.DesiredStatus),
				m.REPLICATION_PRIORITY.EqInt(expected.ReplicationPriority),
				cmm.CLOUD_ID.EqString(expected.KindID),
				cmm.PROVIDER_RESOURCE_ID.EqString(expected.ProviderResourceID),
				cmm.PUBLIC_IP.EqString(expected.PublicIP),
				cmm.LOCATION.EqString(expected.Location),
				cmm.ARCHITECTURE.EqString(expected.Architecture),
				cmm.PUBLIC_KEY.EqString(expected.PublicKey),
				cmm.LIFECYCLE_OWNER_PEER_ID.EqString(expected.LifecycleOwnerPeerID),
			))
	}
}

func createInstanceUpdateMapper(instance InstanceInfo) (db.UpdateMapper, db.UpdateMapper) {

	machineMapper := func() sq.UpdateQuery {
		m := sq.New[db.MACHINE]("")
		mapper := func(col *sq.Column) {
			col.SetString(m.NAME, instance.Name)
			col.SetString(m.KIND, instance.Kind)
			col.SetString(m.DESIRED_STATUS, instance.DesiredStatus)
			col.SetInt(m.REPLICATION_PRIORITY, instance.ReplicationPriority)
		}

		return sq.Update(m).SetFunc(mapper).Where(
			db.UUIDEq(m.ID, instance.ID),
			instancePeerDrainAuthorizationAbsent(instance.ID),
			instanceLifecycleOwnerMatches(instance.ID, instance.LifecycleOwnerPeerID),
		)
	}

	cloudMachineMetadataMapper := func() sq.UpdateQuery {
		m := sq.New[db.CLOUD_MACHINE_METADATA]("")
		mapper := func(col *sq.Column) {
			col.SetString(m.CLOUD_ID, instance.KindID)
			col.SetString(m.PROVIDER_RESOURCE_ID, instance.ProviderResourceID)
			col.SetString(m.PUBLIC_IP, instance.PublicIP)
			col.SetString(m.LOCATION, instance.Location)
			col.SetString(m.ARCHITECTURE, instance.Architecture)
			col.SetString(m.PUBLIC_KEY, instance.PublicKey)
		}
		return sq.Update(m).SetFunc(mapper).Where(
			db.UUIDEq(m.ID, instance.ID),
			m.LIFECYCLE_OWNER_PEER_ID.EqString(instance.LifecycleOwnerPeerID),
			instancePeerDrainAuthorizationAbsent(instance.ID),
		)
	}

	return machineMapper, cloudMachineMetadataMapper
}

func instanceLifecycleOwnerMatches(instanceID string, ownerPeerID string) sq.Predicate {
	metadata := sq.New[db.CLOUD_MACHINE_METADATA]("")
	return sq.Exists(sq.SelectOne().From(metadata).Where(
		db.UUIDEq(metadata.ID, instanceID),
		metadata.LIFECYCLE_OWNER_PEER_ID.EqString(ownerPeerID),
	))
}

func instancePeerDrainAuthorizationAbsent(instanceID string) sq.Predicate {
	facts := sq.New[db.TASK_OPERATION_FACT]("")
	return sq.NotExists(sq.SelectOne().From(facts).Where(
		facts.FACT_KIND.EqString(instancePeerDrainAuthorizationFact),
		facts.SUBJECT_TYPE.EqString(taskSubjectInstance),
		facts.SUBJECT_ID.EqString(instanceID),
	))
}

// createInstanceLifecycleUpdateMapper prevents a stale desired-state or
// reconcile writer on any peer from overwriting the monotonic authorization P.
func createInstanceLifecycleUpdateMapper(instance InstanceInfo) (db.UpdateMapper, db.UpdateMapper) {
	machineMapper := func() sq.UpdateQuery {
		m := sq.New[db.MACHINE]("")
		return sq.Update(m).SetFunc(func(col *sq.Column) {
			col.SetString(m.NAME, instance.Name)
			col.SetString(m.KIND, instance.Kind)
			col.SetString(m.DESIRED_STATUS, instance.DesiredStatus)
			col.SetInt(m.REPLICATION_PRIORITY, instance.ReplicationPriority)
		}).Where(
			db.UUIDEq(m.ID, instance.ID),
			instancePeerDrainAuthorizationAbsent(instance.ID),
			instanceLifecycleOwnerMatches(instance.ID, instance.LifecycleOwnerPeerID),
		)
	}
	metadataMapper := func() sq.UpdateQuery {
		cmm := sq.New[db.CLOUD_MACHINE_METADATA]("")
		return sq.Update(cmm).SetFunc(func(col *sq.Column) {
			col.SetString(cmm.CLOUD_ID, instance.KindID)
			col.SetString(cmm.PROVIDER_RESOURCE_ID, instance.ProviderResourceID)
			col.SetString(cmm.PUBLIC_IP, instance.PublicIP)
			col.SetString(cmm.LOCATION, instance.Location)
			col.SetString(cmm.ARCHITECTURE, instance.Architecture)
			col.SetString(cmm.PUBLIC_KEY, instance.PublicKey)
		}).Where(
			db.UUIDEq(cmm.ID, instance.ID),
			cmm.LIFECYCLE_OWNER_PEER_ID.EqString(instance.LifecycleOwnerPeerID),
			instancePeerDrainAuthorizationAbsent(instance.ID),
		)
	}
	return machineMapper, metadataMapper
}

func createInstanceFinalizeMapper(pendingID string, instance InstanceInfo) (db.UpdateMapper, db.UpdateMapper) {
	machineMapper := func() sq.UpdateQuery {
		m := sq.New[db.MACHINE]("")
		mapper := func(col *sq.Column) {
			col.SetString(m.NAME, instance.Name)
			col.SetString(m.KIND, instance.Kind)
			col.SetString(m.DESIRED_STATUS, instance.DesiredStatus)
			col.SetInt(m.REPLICATION_PRIORITY, instance.ReplicationPriority)
		}
		return sq.Update(m).SetFunc(mapper).Where(
			db.UUIDEq(m.ID, pendingID),
			instancePeerDrainAuthorizationAbsent(pendingID),
			instanceLifecycleOwnerMatches(pendingID, instance.LifecycleOwnerPeerID),
		)
	}

	cloudMachineMetadataMapper := func() sq.UpdateQuery {
		m := sq.New[db.CLOUD_MACHINE_METADATA]("")
		mapper := func(col *sq.Column) {
			col.SetString(m.CLOUD_ID, instance.KindID)
			col.SetString(m.PROVIDER_RESOURCE_ID, instance.ProviderResourceID)
			col.SetString(m.PUBLIC_IP, instance.PublicIP)
			col.SetString(m.LOCATION, instance.Location)
			col.SetString(m.ARCHITECTURE, instance.Architecture)
			col.SetString(m.PUBLIC_KEY, instance.PublicKey)
		}
		return sq.Update(m).SetFunc(mapper).Where(
			db.UUIDEq(m.ID, pendingID),
			m.LIFECYCLE_OWNER_PEER_ID.EqString(instance.LifecycleOwnerPeerID),
			instancePeerDrainAuthorizationAbsent(pendingID),
		)
	}

	return machineMapper, cloudMachineMetadataMapper
}

func createInstanceQueryMapper(id string) db.QueryMapper[InstanceInfo] {
	m := sq.New[db.MACHINE]("")
	cmm := sq.New[db.CLOUD_MACHINE_METADATA]("")

	query := sq.
		From(m).
		Join(cmm, cmm.ID.Eq(m.ID)).
		Where(db.UUIDEq(m.ID, id))

	return func() (sq.SelectQuery, func(row *sq.Row) InstanceInfo) {
		mapper := func(row *sq.Row) InstanceInfo {
			return InstanceInfo{
				ID:                   db.UUIDString(row.BytesField(m.ID)),
				Name:                 row.StringField(m.NAME),
				Kind:                 row.StringField(m.KIND),
				DesiredStatus:        row.StringField(m.DESIRED_STATUS),
				ReplicationPriority:  row.IntField(m.REPLICATION_PRIORITY),
				KindID:               row.StringField(cmm.CLOUD_ID),
				ProviderResourceID:   row.StringField(cmm.PROVIDER_RESOURCE_ID),
				PublicIP:             row.StringField(cmm.PUBLIC_IP),
				Location:             row.StringField(cmm.LOCATION),
				Architecture:         row.StringField(cmm.ARCHITECTURE),
				PublicKey:            row.StringField(cmm.PUBLIC_KEY),
				LifecycleOwnerPeerID: row.StringField(cmm.LIFECYCLE_OWNER_PEER_ID),
			}
		}
		return query, mapper
	}
}

func createInstanceQueryByNameMapper(name string) db.QueryMapper[InstanceInfo] {
	m := sq.New[db.MACHINE]("")
	cmm := sq.New[db.CLOUD_MACHINE_METADATA]("")

	query := sq.
		From(m).
		Join(cmm, cmm.ID.Eq(m.ID)).
		Where(m.NAME.EqString(name))

	return func() (sq.SelectQuery, func(row *sq.Row) InstanceInfo) {
		mapper := func(row *sq.Row) InstanceInfo {
			return InstanceInfo{
				ID:                   db.UUIDString(row.BytesField(m.ID)),
				Name:                 row.StringField(m.NAME),
				Kind:                 row.StringField(m.KIND),
				DesiredStatus:        row.StringField(m.DESIRED_STATUS),
				ReplicationPriority:  row.IntField(m.REPLICATION_PRIORITY),
				KindID:               row.StringField(cmm.CLOUD_ID),
				ProviderResourceID:   row.StringField(cmm.PROVIDER_RESOURCE_ID),
				PublicIP:             row.StringField(cmm.PUBLIC_IP),
				Location:             row.StringField(cmm.LOCATION),
				Architecture:         row.StringField(cmm.ARCHITECTURE),
				PublicKey:            row.StringField(cmm.PUBLIC_KEY),
				LifecycleOwnerPeerID: row.StringField(cmm.LIFECYCLE_OWNER_PEER_ID),
			}
		}
		return query, mapper
	}
}

func createInstanceQueryAllMapper(excludePublicKey string) db.QueryMapper[InstanceInfo] {
	m := sq.New[db.MACHINE]("")
	cmm := sq.New[db.CLOUD_MACHINE_METADATA]("")

	query := sq.
		From(m).
		Join(cmm, cmm.ID.Eq(m.ID))

	if excludePublicKey != "" {
		query = query.Where(cmm.PUBLIC_KEY.NeString(excludePublicKey))
	}

	return func() (sq.SelectQuery, func(row *sq.Row) InstanceInfo) {
		mapper := func(row *sq.Row) InstanceInfo {
			return InstanceInfo{
				ID:                   db.UUIDString(row.BytesField(m.ID)),
				Name:                 row.StringField(m.NAME),
				Kind:                 row.StringField(m.KIND),
				DesiredStatus:        row.StringField(m.DESIRED_STATUS),
				ReplicationPriority:  row.IntField(m.REPLICATION_PRIORITY),
				KindID:               row.StringField(cmm.CLOUD_ID),
				ProviderResourceID:   row.StringField(cmm.PROVIDER_RESOURCE_ID),
				PublicIP:             row.StringField(cmm.PUBLIC_IP),
				Location:             row.StringField(cmm.LOCATION),
				Architecture:         row.StringField(cmm.ARCHITECTURE),
				PublicKey:            row.StringField(cmm.PUBLIC_KEY),
				LifecycleOwnerPeerID: row.StringField(cmm.LIFECYCLE_OWNER_PEER_ID),
			}
		}
		return query, mapper
	}
}

func createInstanceDeleteMapper(id string) (db.DeleteMapper, db.DeleteMapper) {
	mDelete := func() sq.DeleteQuery {
		m := sq.New[db.MACHINE]("")
		return sq.DeleteFrom(m).Where(db.UUIDEq(m.ID, id))
	}

	cmmDelete := func() sq.DeleteQuery {
		cmm := sq.New[db.CLOUD_MACHINE_METADATA]("")
		return sq.DeleteFrom(cmm).Where(db.UUIDEq(cmm.ID, id))
	}

	return mDelete, cmmDelete
}

func createAppDeleteByInstanceMapper(instanceID string) db.DeleteMapper {
	return func() sq.DeleteQuery {
		a := sq.New[db.APP]("")
		return sq.DeleteFrom(a).Where(a.INSTANCE_ID.EqString(instanceID))
	}
}

//
// Provisioner persistence
//

func createProvisionerInsertMapper(provisioner ProvisionerRecord) db.InsertMapper {
	return func() sq.InsertQuery {
		provisioner = provisioner.normalized()
		c := sq.New[db.CLOUD_PROVIDER]("")
		mapper := func(col *sq.Column) {
			col.SetBytes(c.ID, db.MustUUIDBytes(provisioner.ID))
			col.SetString(c.NAME, provisioner.Name)
			col.SetString(c.TYPE, provisioner.Type.String())
			col.SetJSON(c.AUTH, provisioner.Auth)
		}
		return sq.InsertInto(c).ColumnValues(mapper)
	}
}

func createProvisionerUpdateMapper(provisioner ProvisionerRecord) db.UpdateMapper {
	return func() sq.UpdateQuery {
		provisioner = provisioner.normalized()
		c := sq.New[db.CLOUD_PROVIDER]("")
		predicates := []sq.Predicate{db.UUIDEq(c.ID, provisioner.ID)}
		mappper := func(col *sq.Column) {
			col.SetString(c.NAME, provisioner.Name)
			col.SetString(c.TYPE, provisioner.Type.String())
			col.SetJSON(c.AUTH, provisioner.Auth)
		}
		return sq.Update(c).SetFunc(mappper).Where(predicates...)
	}
}

func createProvisionerQueryMapper(predicates []sq.Predicate) db.QueryMapper[ProvisionerRecord] {
	cp := sq.New[db.CLOUD_PROVIDER]("")
	var query sq.SelectQuery
	if len(predicates) != 0 {
		query = sq.
			From(cp).
			Where(predicates...)
	} else {
		query = sq.
			From(cp)
	}

	return func() (sq.SelectQuery, func(row *sq.Row) ProvisionerRecord) {
		mapper := func(row *sq.Row) ProvisionerRecord {
			record := ProvisionerRecord{
				ID:   db.UUIDString(row.BytesField(cp.ID)),
				Name: row.StringField(cp.NAME),
				Type: Type(row.StringField(cp.TYPE)),
			}
			row.JSONField(&record.Auth, cp.AUTH)
			return record.normalized()
		}
		return query, mapper
	}
}

func createProvisionerDeleteMapper(id string) db.DeleteMapper {
	return func() sq.DeleteQuery {
		c := sq.New[db.CLOUD_PROVIDER]("")
		return sq.DeleteFrom(c).Where(db.UUIDEq(c.ID, id))
	}
}
