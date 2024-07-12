package user

import (
	"github.com/bokwoon95/sq"
	"github.com/protosio/protos/internal/db"
)

//
// User
//

func createUserInsertMapper(user User) db.InsertMapper {
	return func() sq.InsertQuery {
		u := sq.New[db.USER]("")
		mapper := func(col *sq.Column) {
			col.SetString(u.USERNAME, user.Name)
			col.SetString(u.NAME, user.Name)
			col.SetBool(u.IS_DISABLED, user.IsDisabled)
		}
		return sq.InsertInto(u).ColumnValues(mapper)
	}
}

func createUserUpdateMapper(user User) func() (sq.Table, func(*sq.Column), []sq.Predicate) {
	return func() (sq.Table, func(*sq.Column), []sq.Predicate) {
		u := sq.New[db.USER]("")
		predicates := []sq.Predicate{u.USERNAME.EqString(user.Username)}
		return u, func(col *sq.Column) {
			col.SetString(u.USERNAME, user.Name)
			col.SetString(u.NAME, user.Name)
			col.SetBool(u.IS_DISABLED, user.IsDisabled)
		}, predicates
	}
}

func createUserQueryMapper(predicates []sq.Predicate) db.QueryMapper[User] {
	u := sq.New[db.USER]("")
	var query sq.SelectQuery
	if len(predicates) == 0 {
		query = sq.
			From(u).
			Where(predicates...)
	} else {
		query = sq.
			From(u)
	}
	return func() (sq.SelectQuery, func(row *sq.Row) User) {
		mapper := func(row *sq.Row) User {

			return User{
				Username:   row.StringField(u.USERNAME),
				Name:       row.StringField(u.NAME),
				IsDisabled: row.BoolField(u.IS_DISABLED),
			}
		}
		return query, mapper
	}
}

func createUserDeleteByNameQuery(username string) func() (sq.Table, []sq.Predicate) {
	return func() (sq.Table, []sq.Predicate) {
		u := sq.New[db.USER]("")
		return u, []sq.Predicate{u.USERNAME.EqString(username)}
	}
}

//
// UserDevice
//

func createUserDeviceInsertMapper(device UserDevice) db.InsertMapper {
	return func() sq.InsertQuery {
		d := sq.New[db.USER_DEVICE_METADATA]("")
		mapper := func(col *sq.Column) {
			col.SetString(d.ID, device.ID)
			col.SetString(d.PUBLIC_KEY, device.PublicKey)
			col.SetString(d.USER_ID, device.UserID)
			col.SetString(d.NAME, device.Name)
		}
		return sq.InsertInto(d).ColumnValues(mapper)
	}
}

func createUserDeviceUpdateMapper(device UserDevice) db.UpdateMapper {
	return func() sq.UpdateQuery {
		d := sq.New[db.USER_DEVICE_METADATA]("")
		mapper := func(col *sq.Column) {
			col.SetString(d.ID, device.ID)
			col.SetString(d.PUBLIC_KEY, device.PublicKey)
			col.SetString(d.USER_ID, device.UserID)
			col.SetString(d.NAME, device.Name)
		}
		return sq.Update(d).SetFunc(mapper).Where(d.ID.EqString(device.ID))
	}
}

func createUserDeviceQueryMapper(publicKey string) db.QueryMapper[UserDevice] {
	d := sq.New[db.USER_DEVICE_METADATA]("")
	query := sq.
		From(d).
		Where(d.PUBLIC_KEY.EqString(publicKey))

	return func() (sq.SelectQuery, func(row *sq.Row) UserDevice) {
		mapper := func(row *sq.Row) UserDevice {
			return UserDevice{
				ID:        row.StringField(d.ID),
				UserID:    row.StringField(d.USER_ID),
				PublicKey: row.StringField(d.PUBLIC_KEY),
				Name:      row.StringField(d.NAME),
			}
		}
		return query, mapper
	}
}

func createUserDeviceQueryAllMapper() db.QueryMapper[UserDevice] {
	d := sq.New[db.USER_DEVICE_METADATA]("")
	query := sq.
		From(d)

	return func() (sq.SelectQuery, func(row *sq.Row) UserDevice) {
		mapper := func(row *sq.Row) UserDevice {
			return UserDevice{
				ID:        row.StringField(d.ID),
				UserID:    row.StringField(d.USER_ID),
				PublicKey: row.StringField(d.PUBLIC_KEY),
				Name:      row.StringField(d.NAME),
			}
		}
		return query, mapper
	}
}

func createUserDeviceDeleteByNameQuery(id string) db.DeleteMapper {
	return func() sq.DeleteQuery {
		u := sq.New[db.USER_DEVICE_METADATA]("")
		return sq.DeleteFrom(u).Where(u.ID.EqString(id))
	}
}
