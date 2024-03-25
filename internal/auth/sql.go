package auth

import (
	"github.com/bokwoon95/sq"
	"github.com/protosio/protos/internal/db"
)

//
// User
//

func createUserInsertMapper(user User) func() (sq.Table, func(*sq.Column)) {
	return func() (sq.Table, func(*sq.Column)) {
		u := sq.New[db.USER]("")
		return u, func(col *sq.Column) {
			col.SetString(u.USERNAME, user.Name)
			col.SetString(u.NAME, user.Name)
			col.SetBool(u.IS_DISABLED, user.IsDisabled)
			col.SetArray(u.DEVICES, user.Devices)
		}
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
			col.SetArray(u.DEVICES, user.Devices)
		}, predicates
	}
}

func createUserQueryMapper(u db.USER, predicates []sq.Predicate) func() (sq.Table, func(row *sq.Row) User, []sq.Predicate) {
	return func() (sq.Table, func(row *sq.Row) User, []sq.Predicate) {
		mapper := func(row *sq.Row) User {

			return User{
				Username:   row.StringField(u.USERNAME),
				Name:       row.StringField(u.NAME),
				IsDisabled: row.BoolField(u.IS_DISABLED),
			}
		}
		return u, mapper, predicates
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

func createUserDeviceInsertMapper(device UserDevice) func() (sq.Table, func(*sq.Column)) {
	return func() (sq.Table, func(*sq.Column)) {
		d := sq.New[db.USER_DEVICE]("")
		return d, func(col *sq.Column) {
			col.SetString(d.ID, device.MachineID)
			col.SetString(d.PUBLIC_KEY, device.PublicKey)
			col.SetString(d.NETWORK, device.Network)
			col.SetString(d.NAME, device.Name)
		}
	}
}

func createUserDeviceUpdateMapper(device UserDevice) func() (sq.Table, func(*sq.Column), []sq.Predicate) {
	return func() (sq.Table, func(*sq.Column), []sq.Predicate) {
		d := sq.New[db.USER_DEVICE]("")
		predicates := []sq.Predicate{d.ID.EqString(device.MachineID)}
		return d, func(col *sq.Column) {
			col.SetString(d.ID, device.MachineID)
			col.SetString(d.PUBLIC_KEY, device.PublicKey)
			col.SetString(d.NETWORK, device.Network)
			col.SetString(d.NAME, device.Name)
		}, predicates
	}
}

func createUserDeviceQueryMapper(ud db.USER_DEVICE, predicates []sq.Predicate) func() (sq.Table, func(row *sq.Row) UserDevice, []sq.Predicate) {
	return func() (sq.Table, func(row *sq.Row) UserDevice, []sq.Predicate) {
		mapper := func(row *sq.Row) UserDevice {

			return UserDevice{
				MachineID: row.StringField(ud.ID),
				PublicKey: row.StringField(ud.PUBLIC_KEY),
				Network:   row.StringField(ud.NETWORK),
				Name:      row.StringField(ud.NAME),
			}
		}
		return ud, mapper, predicates
	}
}

func createUserDeviceDeleteByNameQuery(id string) func() (sq.Table, []sq.Predicate) {
	return func() (sq.Table, []sq.Predicate) {
		u := sq.New[db.USER_DEVICE]("")
		return u, []sq.Predicate{u.ID.EqString(id)}
	}
}
