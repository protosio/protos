package provisioners

import (
	"github.com/bokwoon95/sq"
	"github.com/protosio/protos/internal/db"
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
			col.SetInt(m.WITNESS_RANK, instance.WitnessRank)
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
		}

		return sq.InsertInto(cmm).ColumnValues(mapper)
	}

	return machineMapper, cloudMachineMetadataMapper
}

func createInstanceUpdateMapper(instance InstanceInfo) (db.UpdateMapper, db.UpdateMapper) {

	machineMapper := func() sq.UpdateQuery {
		m := sq.New[db.MACHINE]("")
		mapper := func(col *sq.Column) {
			col.SetString(m.NAME, instance.Name)
			col.SetString(m.KIND, instance.Kind)
			col.SetString(m.DESIRED_STATUS, instance.DesiredStatus)
			col.SetInt(m.WITNESS_RANK, instance.WitnessRank)
		}

		return sq.Update(m).SetFunc(mapper).Where(db.UUIDEq(m.ID, instance.ID))
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
		return sq.Update(m).SetFunc(mapper).Where(db.UUIDEq(m.ID, instance.ID))
	}

	return machineMapper, cloudMachineMetadataMapper
}

func createInstanceFinalizeMapper(pendingID string, instance InstanceInfo) (db.UpdateMapper, db.UpdateMapper) {
	machineMapper := func() sq.UpdateQuery {
		m := sq.New[db.MACHINE]("")
		mapper := func(col *sq.Column) {
			col.SetString(m.NAME, instance.Name)
			col.SetString(m.KIND, instance.Kind)
			col.SetString(m.DESIRED_STATUS, instance.DesiredStatus)
			col.SetInt(m.WITNESS_RANK, instance.WitnessRank)
		}
		return sq.Update(m).SetFunc(mapper).Where(db.UUIDEq(m.ID, pendingID))
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
		return sq.Update(m).SetFunc(mapper).Where(db.UUIDEq(m.ID, pendingID))
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
				ID:                 db.UUIDString(row.BytesField(m.ID)),
				Name:               row.StringField(m.NAME),
				Kind:               row.StringField(m.KIND),
				DesiredStatus:      row.StringField(m.DESIRED_STATUS),
				WitnessRank:        row.IntField(m.WITNESS_RANK),
				KindID:             row.StringField(cmm.CLOUD_ID),
				ProviderResourceID: row.StringField(cmm.PROVIDER_RESOURCE_ID),
				PublicIP:           row.StringField(cmm.PUBLIC_IP),
				Location:           row.StringField(cmm.LOCATION),
				Architecture:       row.StringField(cmm.ARCHITECTURE),
				PublicKey:          row.StringField(cmm.PUBLIC_KEY),
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
				ID:                 db.UUIDString(row.BytesField(m.ID)),
				Name:               row.StringField(m.NAME),
				Kind:               row.StringField(m.KIND),
				DesiredStatus:      row.StringField(m.DESIRED_STATUS),
				WitnessRank:        row.IntField(m.WITNESS_RANK),
				KindID:             row.StringField(cmm.CLOUD_ID),
				ProviderResourceID: row.StringField(cmm.PROVIDER_RESOURCE_ID),
				PublicIP:           row.StringField(cmm.PUBLIC_IP),
				Location:           row.StringField(cmm.LOCATION),
				Architecture:       row.StringField(cmm.ARCHITECTURE),
				PublicKey:          row.StringField(cmm.PUBLIC_KEY),
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
				ID:                 db.UUIDString(row.BytesField(m.ID)),
				Name:               row.StringField(m.NAME),
				Kind:               row.StringField(m.KIND),
				DesiredStatus:      row.StringField(m.DESIRED_STATUS),
				WitnessRank:        row.IntField(m.WITNESS_RANK),
				KindID:             row.StringField(cmm.CLOUD_ID),
				ProviderResourceID: row.StringField(cmm.PROVIDER_RESOURCE_ID),
				PublicIP:           row.StringField(cmm.PUBLIC_IP),
				Location:           row.StringField(cmm.LOCATION),
				Architecture:       row.StringField(cmm.ARCHITECTURE),
				PublicKey:          row.StringField(cmm.PUBLIC_KEY),
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
// Cloud provider
//

func createCloudProviderInsertMapper(provider ProviderRecord) db.InsertMapper {
	return func() sq.InsertQuery {
		provider = provider.normalized()
		c := sq.New[db.CLOUD_PROVIDER]("")
		mapper := func(col *sq.Column) {
			col.SetBytes(c.ID, db.MustUUIDBytes(provider.ID))
			col.SetString(c.NAME, provider.Name)
			col.SetString(c.TYPE, provider.Type.String())
			col.SetJSON(c.AUTH, provider.Auth)
		}
		return sq.InsertInto(c).ColumnValues(mapper)
	}
}

func createCloudProviderUpdateMapper(provider ProviderRecord) db.UpdateMapper {
	return func() sq.UpdateQuery {
		provider = provider.normalized()
		c := sq.New[db.CLOUD_PROVIDER]("")
		predicates := []sq.Predicate{db.UUIDEq(c.ID, provider.ID)}
		mappper := func(col *sq.Column) {
			col.SetString(c.NAME, provider.Name)
			col.SetString(c.TYPE, provider.Type.String())
			col.SetJSON(c.AUTH, provider.Auth)
		}
		return sq.Update(c).SetFunc(mappper).Where(predicates...)
	}
}

func createCloudProviderQueryMapper(predicates []sq.Predicate) db.QueryMapper[ProviderRecord] {
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

	return func() (sq.SelectQuery, func(row *sq.Row) ProviderRecord) {
		mapper := func(row *sq.Row) ProviderRecord {
			record := ProviderRecord{
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

func createCloudProviderDeleteMapper(id string) db.DeleteMapper {
	return func() sq.DeleteQuery {
		c := sq.New[db.CLOUD_PROVIDER]("")
		return sq.DeleteFrom(c).Where(db.UUIDEq(c.ID, id))
	}
}
