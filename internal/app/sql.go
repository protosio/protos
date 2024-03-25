package app

import (
	"net"

	"github.com/bokwoon95/sq"
	"github.com/protosio/protos/internal/db"
)

func createAppInsertMapper(app App) func() (sq.Table, func(*sq.Column)) {
	return func() (sq.Table, func(*sq.Column)) {
		a := sq.New[db.APP]("")
		return a, func(col *sq.Column) {
			col.SetString(a.NAME, app.Name)
			col.SetString(a.ID, app.ID)
			col.SetString(a.INSTALLER_REF, app.InstallerRef)
			col.SetString(a.INSTANCE_NAME, app.InstanceName)
			col.SetString(a.DESIRED_STATUS, app.DesiredStatus)
			col.SetString(a.IP, app.IP.String())
			col.SetBool(a.PERSISTENCE, app.Persistence)
		}
	}
}

func createAppUpdateMapper(app App) func() (sq.Table, func(*sq.Column), []sq.Predicate) {
	return func() (sq.Table, func(*sq.Column), []sq.Predicate) {
		a := sq.New[db.APP]("")
		predicates := []sq.Predicate{a.ID.EqString(app.ID)}
		return a, func(col *sq.Column) {
			col.SetString(a.NAME, app.Name)
			col.SetString(a.INSTALLER_REF, app.InstallerRef)
			col.SetString(a.INSTANCE_NAME, app.InstanceName)
			col.SetString(a.DESIRED_STATUS, app.DesiredStatus)
			col.SetString(a.IP, app.IP.String())
			col.SetBool(a.PERSISTENCE, app.Persistence)
		}, predicates
	}
}

func createInstanceQueryMapper(a db.APP, predicates []sq.Predicate) func() (sq.Table, func(row *sq.Row) App, []sq.Predicate) {
	return func() (sq.Table, func(row *sq.Row) App, []sq.Predicate) {
		mapper := func(row *sq.Row) App {

			return App{
				Name:          row.StringField(a.NAME),
				ID:            row.StringField(a.ID),
				InstallerRef:  row.StringField(a.INSTALLER_REF),
				InstanceName:  row.StringField(a.INSTANCE_NAME),
				DesiredStatus: row.StringField(a.DESIRED_STATUS),
				IP:            net.ParseIP(row.StringField(a.IP)),
				Persistence:   row.BoolField(a.PERSISTENCE),
			}
		}
		return a, mapper, predicates
	}
}

func createAppDeleteByNameQuery(name string) func() (sq.Table, []sq.Predicate) {
	return func() (sq.Table, []sq.Predicate) {
		a := sq.New[db.APP]("")
		return a, []sq.Predicate{a.NAME.EqString(name)}
	}
}
