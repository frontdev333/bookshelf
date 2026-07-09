package domain

import (
	"bookshelf/books-service/internal/client"
	"database/sql"
	"time"
)

type Book struct {
	ID            string          `db:"id"`
	Title         string          `db:"title"`
	Author        string          `db:"author"`
	UserID        string          `db:"created_by"`
	CreatedAt     time.Time       `db:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at"`
	Description   sql.NullString  `db:"description"`
	ISBN          sql.NullString  `db:"isbn"`
	ReviewsCount  sql.NullInt64   `db:"reviews_count"`
	PublishedYear sql.NullInt32   `db:"published_year"`
	AverageRating sql.NullFloat64 `db:"average_rating"`
}

type BookResponse struct {
	ID            string             `json:"id"`
	Title         string             `json:"title"`
	Author        string             `json:"author"`
	Creator       client.UserSummary `json:"creator"`
	UserID        string             `json:"created_by"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	Description   *string            `json:"description"`
	ISBN          *string            `json:"isbn"`
	ReviewsCount  *int               `json:"reviews_count"`
	PublishedYear *int               `json:"published_year"`
	AverageRating *float64           `json:"average_rating"`
}

type BookListResponse struct {
	Data       []BookResponse `json:"data"`
	Pagination Pagination     `json:"pagination"`
}

type CreateBookRequest struct {
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	Description   *string `json:"description"`
	ISBN          *string `json:"isbn"`
	PublishedYear *int    `json:"published_year"`
}

type UpdateBookRequest struct {
	Title         *string `json:"title"`
	Author        *string `json:"author"`
	Description   *string `json:"description"`
	ISBN          *string `json:"isbn"`
	PublishedYear *int    `json:"published_year"`
}

type BookFilter struct {
	Search *string `json:"search"`
	Sort   *string `json:"sort"`
	Order  *string `json:"order"`
	Page   *string `json:"page"`
	Limit  *string `json:"limit"`
}

type ListParams struct {
	Order  string
	Sort   string
	Search string
	Page   int
	Limit  int
}

func (b *Book) ToResponse(creator client.UserSummary, reviewsCount *int) *BookResponse {
	var desc *string
	desc = nil

	if b.Description.Valid {
		desc = &b.Description.String
	}

	var isbn *string
	isbn = nil

	if b.ISBN.Valid {
		isbn = &b.ISBN.String
	}

	var publishedYear *int
	publishedYear = nil

	if b.PublishedYear.Valid {
		tmpPubY := int(b.PublishedYear.Int32)
		publishedYear = &tmpPubY
	}

	var averageRating *float64
	averageRating = nil

	if b.AverageRating.Valid {
		averageRating = &b.AverageRating.Float64
	}

	return &BookResponse{
		ID:            b.ID,
		Title:         b.Title,
		Author:        b.Author,
		Creator:       creator,
		UserID:        creator.ID,
		CreatedAt:     b.CreatedAt,
		UpdatedAt:     b.UpdatedAt,
		Description:   desc,
		ISBN:          isbn,
		ReviewsCount:  reviewsCount,
		PublishedYear: publishedYear,
		AverageRating: averageRating,
	}
}

func (f *BookFilter) SeedDefaults() {
	if f.Sort == nil {
		sort := "id"
		f.Sort = &sort
	}

	if f.Order == nil {
		order := "desc"
		f.Order = &order
	}

	if f.Limit == nil {
		limit := "10"
		f.Order = &limit
	}

	if f.Page == nil {
		page := "1"
		f.Page = &page
	}
}
