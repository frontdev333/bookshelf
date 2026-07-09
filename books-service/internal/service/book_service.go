package service

import (
	"bookshelf/books-service/internal/client"
	"bookshelf/books-service/internal/domain"
	"bookshelf/books-service/internal/repository"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strconv"
	"time"
)

var (
	ErrBookNotFound    = errors.New("book not found")
	ErrNotBookOwner    = errors.New("you are not the owner of this book")
	ErrBookTitleEmpty  = errors.New("book title is empty")
	ErrBookAuthorEmpty = errors.New("book author is empty")
)

type BookService struct {
	bookRepo   *repository.BookRepository
	reviewRepo *repository.ReviewRepository
	aClient    *client.AuthClient
}

func NewBookService(
	repo *repository.BookRepository,
	reviewRepo *repository.ReviewRepository,
	aClient *client.AuthClient,
) *BookService {
	return &BookService{
		repo,
		reviewRepo,
		aClient,
	}
}

func (s *BookService) Create(
	ctx context.Context,
	userId string,
	req domain.CreateBookRequest,
) (*domain.BookResponse, error) {
	if req.Title == "" {
		return nil, ErrBookTitleEmpty
	}

	if req.Author == "" {
		return nil, ErrBookAuthorEmpty
	}

	var desc sql.NullString
	var isbn sql.NullString
	var pubYear sql.NullInt32

	if req.Description == nil {
		desc.Valid = false
	} else {
		desc.String = *req.Description
		desc.Valid = true
	}

	if req.ISBN == nil {
		isbn.Valid = false
	} else {
		isbn.String = *req.ISBN
		isbn.Valid = true
	}

	if req.PublishedYear == nil {
		pubYear.Valid = false
	} else {
		pubYear.Int32 = int32(*req.PublishedYear)
		pubYear.Valid = true
	}

	b := &domain.Book{
		Title:         req.Title,
		Author:        req.Author,
		UserID:        userId,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Description:   desc,
		ISBN:          isbn,
		PublishedYear: pubYear,
		AverageRating: sql.NullFloat64{},
	}

	if err := s.bookRepo.Create(ctx, b); err != nil {
		return nil, err
	}

	reviewsCount, err := s.reviewRepo.GetReviewsCount(ctx, b.ID)
	if err != nil {
		return nil, err
	}

	u, err := s.aClient.GetUserByID(ctx, userId)
	if err != nil {
		return nil, err
	}

	return b.ToResponse(u.ToSummary(), &reviewsCount), nil
}

func (s *BookService) GetByID(ctx context.Context, id string) (*domain.BookResponse, error) {
	b, err := s.bookRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	u, err := s.aClient.GetUserByID(ctx, b.UserID)
	if err != nil {
		return nil, err
	}

	reviewsCount, err := s.reviewRepo.GetReviewsCount(ctx, id)
	if err != nil {
		return nil, err
	}

	return b.ToResponse(u.ToSummary(), &reviewsCount), nil
}

func (s *BookService) List(ctx context.Context, f domain.BookFilter) (*domain.BookListResponse, error) {
	f.SeedDefaults()

	var order string
	var sort string
	var search string
	var page int
	var limit int
	var err error

	if f.Sort != nil {
		switch *f.Sort {
		case "author":
			sort = "author"
		case "published_year":
			sort = "published_year"
		case "created_by":
			sort = "created_by"
		case "updated_at":
			sort = "updated_at"
		default:
			sort = "created_at"
		}
	}

	if f.Order == nil || *f.Order != "DESC" && *f.Order != "ASC" {
		order = "DESC"
	} else {
		order = "ASC"
	}

	if f.Search != nil {
		search = "%" + *f.Search + "%"
	}

	cPage, err := strconv.Atoi(*f.Page)
	if err != nil {
		return nil, err
	}

	cLimit, err := strconv.Atoi(*f.Limit)
	if err != nil {
		return nil, err
	}

	if f.Page != nil {
		page, err = strconv.Atoi(*f.Page)
		if err != nil {
			return nil, err
		}
	} else {
		page = 1
	}

	if f.Limit != nil {

		limit, err = strconv.Atoi(*f.Limit)
		if err != nil {
			return nil, err
		}
	} else {
		limit = 10
	}

	params := domain.ListParams{
		Order:  order,
		Sort:   sort,
		Search: search,
		Page:   page,
		Limit:  limit,
	}
	list, count, err := s.bookRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}

	booksResponse := make([]domain.BookResponse, 0, len(list))
	creatorsIDs := make([]string, 0, len(list))
	booksIDs := make([]string, 0, len(list))

	for _, v := range list {
		creatorsIDs = append(creatorsIDs, v.UserID)
		booksIDs = append(booksIDs, v.ID)
	}

	users, err := s.aClient.GetUsersByIDs(ctx, creatorsIDs)
	if err != nil {
		return nil, err
	}

	creators := make(map[string]client.UserPublic, len(users))

	for _, v := range users {
		creators[v.ID] = v
	}

	reviewsCounts, err := s.reviewRepo.GetReviewsCounts(ctx, booksIDs)
	if err != nil {
		return nil, err
	}

	for _, v := range list {
		u, ok := creators[v.UserID]
		if !ok {
			slog.Error("BookService List()", "error", "creator not found", "user_id", v.UserID)
			continue
		}

		reviewsCount := 0
		if c, ok := reviewsCounts[v.ID]; ok {
			reviewsCount = c
		}

		b := v.ToResponse(u.ToSummary(), &reviewsCount)

		booksResponse = append(booksResponse, *b)
	}

	return &domain.BookListResponse{
		Data: booksResponse,
		Pagination: domain.Pagination{
			Page:       cPage,
			Limit:      cLimit,
			Total:      count,
			TotalPages: (count + cLimit - 1) / cLimit,
		},
	}, nil
}

func (s *BookService) Update(ctx context.Context, userID, bookID string, req domain.UpdateBookRequest) (*domain.BookResponse, error) {
	b, err := s.bookRepo.GetByID(ctx, bookID)
	if err != nil {
		return nil, ErrBookNotFound
	}

	if b.UserID != userID {
		return nil, ErrNotBookOwner
	}

	if req.Author != nil {
		b.Author = *req.Author
	}

	if req.ISBN != nil {
		b.ISBN = sql.NullString{
			String: *req.ISBN,
			Valid:  true,
		}
	}

	if req.Description != nil {
		b.Description = sql.NullString{
			String: *req.Description,
			Valid:  true,
		}
	}

	if req.Title != nil {
		b.Title = *req.Title
	}

	if req.PublishedYear != nil {
		b.PublishedYear = sql.NullInt32{
			Int32: int32(*req.PublishedYear),
			Valid: true,
		}
	}

	if err = s.bookRepo.Update(ctx, b); err != nil {
		return nil, err
	}

	u, err := s.aClient.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	reviewsCount, err := s.reviewRepo.GetReviewsCount(ctx, b.ID)
	if err != nil {
		return nil, err
	}

	return b.ToResponse(u.ToSummary(), &reviewsCount), nil
}

func (s *BookService) Delete(ctx context.Context, userID, bookID string) error {
	b, err := s.bookRepo.GetByID(ctx, bookID)
	if err != nil {
		return ErrBookNotFound
	}

	if b.UserID != userID {
		return ErrNotBookOwner
	}

	return s.bookRepo.Delete(ctx, b.ID)
}
