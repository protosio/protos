package db

import (
	"fmt"
	"strings"

	"github.com/bokwoon95/sq"
	"github.com/google/uuid"
)

const UUIDBinaryLength = 16

func NewUUIDv7() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate uuid v7: %w", err)
	}
	return id.String(), nil
}

func MustNewUUIDv7() string {
	id, err := NewUUIDv7()
	if err != nil {
		panic(err)
	}
	return id
}

func UUIDBytes(id string) ([]byte, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("parse uuid %q: %w", id, err)
	}
	out := make([]byte, UUIDBinaryLength)
	copy(out, parsed[:])
	return out, nil
}

func MustUUIDBytes(id string) []byte {
	bytes, err := UUIDBytes(id)
	if err != nil {
		panic(err)
	}
	return bytes
}

func UUIDString(bytes []byte) string {
	if len(bytes) != UUIDBinaryLength {
		return ""
	}
	var id uuid.UUID
	copy(id[:], bytes)
	return id.String()
}

func UUIDEq(field sq.BinaryField, id string) sq.Predicate {
	bytes, err := UUIDBytes(id)
	if err != nil {
		return sq.Expr("1 = 0")
	}
	return field.EqBytes(bytes)
}
