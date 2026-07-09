package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	Book   *BookRepository
	Review *ReviewRepository
}

func New(db *sqlx.DB) *Repository {
	return &Repository{
		Book:   &BookRepository{db},
		Review: &ReviewRepository{db},
	}
}

func getEntityByField[T any](ctx context.Context, db *sqlx.DB, table, field string, val interface{}) (*T, error) {
	var entity T

	q := `SELECT * FROM ` + table + ` WHERE ` + field + ` = $1 LIMIT 1`
	err := db.GetContext(ctx, &entity, q, val)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &entity, nil
}
