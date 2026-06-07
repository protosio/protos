package app

import (
	"github.com/bokwoon95/sq"
	"github.com/protosio/protos/internal/db"
)

func createAppInsertMapper(app App) db.InsertMapper {
	return func() sq.InsertQuery {
		a := sq.New[db.APP]("")
		mapper := func(col *sq.Column) {
			col.SetString(a.NAME, app.Name)
			col.SetBytes(a.ID, db.MustUUIDBytes(app.ID))
			col.SetString(a.INSTALLER_REF, app.InstallerRef)
			col.SetString(a.INSTANCE_ID, app.InstanceID)
			col.SetString(a.DESIRED_STATUS, app.DesiredStatus)
			col.SetBool(a.PERSISTENCE, app.Persistence)
			col.SetString(a.PUBLIC_KEY, app.PublicKey)
		}
		return sq.InsertInto(a).ColumnValues(mapper)
	}
}

func createAppUpdateMapper(app App) db.UpdateMapper {
	return func() sq.UpdateQuery {
		a := sq.New[db.APP]("")
		mapper := func(col *sq.Column) {
			col.SetString(a.NAME, app.Name)
			col.SetString(a.INSTALLER_REF, app.InstallerRef)
			col.SetString(a.INSTANCE_ID, app.InstanceID)
			col.SetString(a.DESIRED_STATUS, app.DesiredStatus)
			col.SetBool(a.PERSISTENCE, app.Persistence)
		}
		return sq.Update(a).SetFunc(mapper).Where(db.UUIDEq(a.ID, app.ID))
	}
}

func createAppQueryMapper(predicates []sq.Predicate) db.QueryMapper[App] {
	a := sq.New[db.APP]("")
	var query sq.SelectQuery
	if len(predicates) != 0 {
		query = sq.
			From(a).
			Where(predicates...)
	} else {
		query = sq.
			From(a)
	}

	return func() (sq.SelectQuery, func(row *sq.Row) App) {
		mapper := func(row *sq.Row) App {
			return App{
				Name:          row.StringField(a.NAME),
				ID:            db.UUIDString(row.BytesField(a.ID)),
				InstallerRef:  row.StringField(a.INSTALLER_REF),
				InstanceID:    row.StringField(a.INSTANCE_ID),
				DesiredStatus: row.StringField(a.DESIRED_STATUS),
				Persistence:   row.BoolField(a.PERSISTENCE),
				PublicKey:     row.StringField(a.PUBLIC_KEY),
				IP:            appIPFromPublicKey(row.StringField(a.PUBLIC_KEY)),
			}
		}
		return query, mapper
	}
}

func createAppDeleteByNameQuery(id string) db.DeleteMapper {
	return func() sq.DeleteQuery {
		a := sq.New[db.APP]("")
		return sq.DeleteFrom(a).Where(db.UUIDEq(a.ID, id))
	}
}

func createAppInstancePublicKeyQueryMapper(instanceID string) db.QueryMapper[string] {
	cmm := sq.New[db.CLOUD_MACHINE_METADATA]("")
	query := sq.From(cmm).Where(db.UUIDEq(cmm.ID, instanceID))

	return func() (sq.SelectQuery, func(row *sq.Row) string) {
		mapper := func(row *sq.Row) string {
			return row.StringField(cmm.PUBLIC_KEY)
		}
		return query, mapper
	}
}
