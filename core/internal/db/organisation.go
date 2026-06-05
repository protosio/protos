package db

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/bokwoon95/sq"
)

const DefaultOrganisationName = "home"

type Organisation struct {
	ID        string
	Name      string
	CreatedAt string
}

func EnsureOrganisation(database *DB, name string) (Organisation, error) {
	organisations, err := ListOrganisations(database)
	if err != nil {
		return Organisation{}, err
	}
	if len(organisations) > 0 {
		return organisations[0], nil
	}
	return CreateOrganisation(database, name)
}

func CreateOrganisation(database *DB, name string) (Organisation, error) {
	id, err := GenerateOrganisationID()
	if err != nil {
		return Organisation{}, err
	}
	organisation := Organisation{
		ID:        id,
		Name:      normalizeOrganisationName(name),
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := Insert(database, createOrganisationInsertMapper(organisation)); err != nil {
		return Organisation{}, fmt.Errorf("create organisation: %w", err)
	}
	return organisation, nil
}

func ListOrganisations(database *DB) ([]Organisation, error) {
	organisations, err := SelectMultiple(database, createOrganisationQueryMapper(nil))
	if err != nil {
		return nil, fmt.Errorf("list organisations: %w", err)
	}
	return organisations, nil
}

func GenerateOrganisationID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate organisation id: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func normalizeOrganisationName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return DefaultOrganisationName
	}
	return name
}

func createOrganisationInsertMapper(organisation Organisation) InsertMapper {
	return func() sq.InsertQuery {
		u := sq.New[ORGANISATION]("")
		mapper := func(col *sq.Column) {
			col.SetString(u.ID, organisation.ID)
			col.SetString(u.NAME, organisation.Name)
			col.SetString(u.CREATED_AT, organisation.CreatedAt)
		}
		return sq.InsertInto(u).ColumnValues(mapper)
	}
}

func createOrganisationQueryMapper(predicates []sq.Predicate) QueryMapper[Organisation] {
	u := sq.New[ORGANISATION]("")
	query := sq.From(u)
	if len(predicates) > 0 {
		query = query.Where(predicates...)
	}

	return func() (sq.SelectQuery, func(row *sq.Row) Organisation) {
		mapper := func(row *sq.Row) Organisation {
			return Organisation{
				ID:        row.StringField(u.ID),
				Name:      row.StringField(u.NAME),
				CreatedAt: row.StringField(u.CREATED_AT),
			}
		}
		return query, mapper
	}
}
