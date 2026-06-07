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
			col.SetBytes(u.ID, db.MustUUIDBytes(user.ID))
			col.SetString(u.USERNAME, user.Username)
			col.SetString(u.NAME, user.Name)
			col.SetBool(u.IS_DISABLED, user.IsDisabled)
		}
		return sq.InsertInto(u).ColumnValues(mapper)
	}
}

func createUserQueryMapper(predicates []sq.Predicate) db.QueryMapper[User] {
	u := sq.New[db.USER]("")
	var query sq.SelectQuery
	if len(predicates) != 0 {
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
				ID:         db.UUIDString(row.BytesField(u.ID)),
				Username:   row.StringField(u.USERNAME),
				Name:       row.StringField(u.NAME),
				IsDisabled: row.BoolField(u.IS_DISABLED),
			}
		}
		return query, mapper
	}
}

//
// UserDevice
//

func createUserDeviceInsertMapper(device UserDevice) db.InsertMapper {
	return func() sq.InsertQuery {
		d := sq.New[db.USER_DEVICE_METADATA]("")
		mapper := func(col *sq.Column) {
			col.SetBytes(d.ID, db.MustUUIDBytes(device.ID))
			col.SetString(d.PUBLIC_KEY, device.PublicKey)
			col.SetBytes(d.USER_ID, db.MustUUIDBytes(device.UserID))
			col.SetString(d.NAME, device.Name)
			col.SetInt(d.WITNESS_RANK, device.WitnessRank)
		}
		return sq.InsertInto(d).ColumnValues(mapper)
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
				ID:          db.UUIDString(row.BytesField(d.ID)),
				UserID:      db.UUIDString(row.BytesField(d.USER_ID)),
				PublicKey:   row.StringField(d.PUBLIC_KEY),
				Name:        row.StringField(d.NAME),
				WitnessRank: row.IntField(d.WITNESS_RANK),
			}
		}
		return query, mapper
	}
}

func createUserDeviceQueryAllMapper(excludePublicKey string) db.QueryMapper[UserDevice] {
	d := sq.New[db.USER_DEVICE_METADATA]("")
	query := sq.
		From(d)

	if excludePublicKey != "" {
		query = query.Where(d.PUBLIC_KEY.NeString(excludePublicKey))
	}

	return func() (sq.SelectQuery, func(row *sq.Row) UserDevice) {
		mapper := func(row *sq.Row) UserDevice {
			return UserDevice{
				ID:          db.UUIDString(row.BytesField(d.ID)),
				UserID:      db.UUIDString(row.BytesField(d.USER_ID)),
				PublicKey:   row.StringField(d.PUBLIC_KEY),
				Name:        row.StringField(d.NAME),
				WitnessRank: row.IntField(d.WITNESS_RANK),
			}
		}
		return query, mapper
	}
}
