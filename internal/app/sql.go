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
			col.SetString(a.ID, app.ID)
			col.SetString(a.INSTALLER_REF, app.InstallerRef)
			col.SetString(a.INSTANCE_ID, app.InstanceID)
			col.SetString(a.DESIRED_STATUS, app.DesiredStatus)
			col.SetBool(a.PERSISTENCE, app.Persistence)
			col.SetString(a.PUBLIC_KEY, app.ID)
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
		return sq.Update(a).SetFunc(mapper).Where(a.ID.EqString(app.ID))
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
				ID:            row.StringField(a.ID),
				InstallerRef:  row.StringField(a.INSTALLER_REF),
				InstanceID:    row.StringField(a.INSTANCE_ID),
				DesiredStatus: row.StringField(a.DESIRED_STATUS),
				Persistence:   row.BoolField(a.PERSISTENCE),
			}
		}
		return query, mapper
	}
}

func createAppDeleteByNameQuery(id string) db.DeleteMapper {
	return func() sq.DeleteQuery {
		a := sq.New[db.APP]("")
		return sq.DeleteFrom(a).Where(a.ID.EqString(id))
	}
}
