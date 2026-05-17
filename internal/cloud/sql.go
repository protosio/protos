package cloud

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
			col.SetString(m.ID, instance.ID)
			col.SetString(m.NAME, instance.Name)
			col.SetString(m.KIND, instance.Kind)
		}

		return sq.InsertInto(m).ColumnValues(mapper)
	}

	cloudMachineMetadataMapper := func() sq.InsertQuery {
		cmm := sq.New[db.CLOUD_MACHINE_METADATA]("")
		mapper := func(col *sq.Column) {
			col.SetString(cmm.ID, instance.ID)
			col.SetString(cmm.CLOUD_ID, instance.KindID)
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
		}

		return sq.Update(m).SetFunc(mapper).Where(m.ID.EqString(instance.ID))
	}

	cloudMachineMetadataMapper := func() sq.UpdateQuery {
		m := sq.New[db.CLOUD_MACHINE_METADATA]("")
		mapper := func(col *sq.Column) {
			col.SetString(m.CLOUD_ID, instance.KindID)
			col.SetString(m.PUBLIC_IP, instance.PublicIP)
			col.SetString(m.LOCATION, instance.Location)
			col.SetString(m.ARCHITECTURE, instance.Architecture)
			col.SetString(m.PUBLIC_KEY, instance.PublicKey)
		}
		return sq.Update(m).SetFunc(mapper).Where(m.ID.EqString(instance.ID))
	}

	return machineMapper, cloudMachineMetadataMapper
}

func createInstanceQueryMapper(id string) db.QueryMapper[InstanceInfo] {
	m := sq.New[db.MACHINE]("")
	cmm := sq.New[db.CLOUD_MACHINE_METADATA]("")

	query := sq.
		From(m).
		Join(cmm, cmm.ID.Eq(m.ID)).
		Where(m.ID.EqString(id))

	return func() (sq.SelectQuery, func(row *sq.Row) InstanceInfo) {
		mapper := func(row *sq.Row) InstanceInfo {
			return InstanceInfo{
				ID:           row.StringField(m.ID),
				Name:         row.StringField(m.NAME),
				Kind:         row.StringField(m.KIND),
				KindID:       row.StringField(cmm.CLOUD_ID),
				PublicIP:     row.StringField(cmm.PUBLIC_IP),
				Location:     row.StringField(cmm.LOCATION),
				Architecture: row.StringField(cmm.ARCHITECTURE),
				PublicKey:    row.StringField(cmm.PUBLIC_KEY),
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
				ID:           row.StringField(m.ID),
				Name:         row.StringField(m.NAME),
				Kind:         row.StringField(m.KIND),
				KindID:       row.StringField(cmm.CLOUD_ID),
				PublicIP:     row.StringField(cmm.PUBLIC_IP),
				Location:     row.StringField(cmm.LOCATION),
				Architecture: row.StringField(cmm.ARCHITECTURE),
				PublicKey:    row.StringField(cmm.PUBLIC_KEY),
			}
		}
		return query, mapper
	}
}

func createInstanceDeleteMapper(id string) (db.DeleteMapper, db.DeleteMapper) {
	mDelete := func() sq.DeleteQuery {
		m := sq.New[db.MACHINE]("")
		return sq.DeleteFrom(m).Where(m.ID.EqString(id))
	}

	cmmDelete := func() sq.DeleteQuery {
		cmm := sq.New[db.CLOUD_MACHINE_METADATA]("")
		return sq.DeleteFrom(cmm).Where(cmm.ID.EqString(id))
	}

	return mDelete, cmmDelete
}

//
// Cloud provider
//

func createCloudProviderInsertMapper(provider ProviderRecord) db.InsertMapper {
	return func() sq.InsertQuery {
		provider = provider.normalized()
		c := sq.New[db.CLOUD_PROVIDER]("")
		mapper := func(col *sq.Column) {
			col.SetString(c.ID, provider.ID)
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
		predicates := []sq.Predicate{c.ID.EqString(provider.ID)}
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
				ID:   row.StringField(cp.ID),
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
		return sq.DeleteFrom(c).Where(c.ID.EqString(id))
	}
}
