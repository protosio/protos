package db

import "github.com/bokwoon95/sq"

func CreatePeerInsertMapper(peerID string) InsertMapper {
	return func() sq.InsertQuery {
		p := sq.New[PEER]("")
		mapper := func(col *sq.Column) {
			col.SetString(p.ID, peerID)
		}
		return sq.InsertInto(p).ColumnValues(mapper)
	}
}

func CreatePeerDeleteMapper(peerID string) DeleteMapper {
	return func() sq.DeleteQuery {
		p := sq.New[PEER]("")
		return sq.DeleteFrom(p).Where(p.ID.EqString(peerID))
	}
}

func GetPeerIDs(database *DB) (map[string]struct{}, error) {
	ids, err := SelectMultiple(database, createPeerQueryAllMapper())
	if err != nil {
		return nil, err
	}

	peers := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		peers[id] = struct{}{}
	}
	return peers, nil
}

func createPeerQueryAllMapper() QueryMapper[string] {
	p := sq.New[PEER]("")
	query := sq.From(p)

	return func() (sq.SelectQuery, func(row *sq.Row) string) {
		mapper := func(row *sq.Row) string {
			return row.StringField(p.ID)
		}
		return query, mapper
	}
}
