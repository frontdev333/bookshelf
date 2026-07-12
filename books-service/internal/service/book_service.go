package service

import (
	"bookshelf/books-service/internal/client"
	"bookshelf/books-service/internal/domain"
	"bookshelf/books-service/internal/repository"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrBookNotFound       = errors.New("book not found")
	ErrNotBookOwner       = errors.New("you are not the owner of this book")
	ErrBookTitleEmpty     = errors.New("book title is empty")
	ErrBookAuthorEmpty    = errors.New("book author is empty")
	ErrInvalidPagination  = errors.New("invalid pagination parameters")
	ErrCreatorLookupError = errors.New("book creator lookup failed")
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
	if strings.TrimSpace(req.Title) == "" {
		return nil, ErrBookTitleEmpty
	}

	if strings.TrimSpace(req.Author) == "" {
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

	u, err := s.aClient.GetUserByID(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCreatorLookupError, err)
	}

	if err := s.bookRepo.Create(ctx, b); err != nil {
		return nil, err
	}

	reviewsCount := 0
	return b.ToResponse(u.ToSummary(), &reviewsCount), nil
}

func (s *BookService) GetByID(ctx context.Context, id string) (*domain.BookResponse, error) {
	b, err := s.bookRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrBookNotFound) {
			return nil, ErrBookNotFound
		}
		return nil, err
	}

	u, err := s.aClient.GetUserByID(ctx, b.UserID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCreatorLookupError, err)
	}

	reviewsCount, err := s.reviewRepo.GetReviewsCount(ctx, id)
	if err != nil {
		return nil, err
	}

	return b.ToResponse(u.ToSummary(), &reviewsCount), nil
}

func (s *BookService) List(ctx context.Context, f domain.BookFilter) (*domain.BookListResponse, error) {
	f.SeedDefaults()

	sort := "created_at"
	if f.Sort != nil {
		switch strings.ToLower(strings.TrimSpace(*f.Sort)) {
		case "author":
			sort = "author"
		case "published_year":
			sort = "published_year"
		case "created_by":
			sort = "created_by"
		case "updated_at":
			sort = "updated_at"
		}
	}

	order := "DESC"
	if f.Order != nil {
		switch strings.ToUpper(strings.TrimSpace(*f.Order)) {
		case "ASC":
			order = "ASC"
		case "DESC":
			order = "DESC"
		}
	}

	search := ""
	if f.Search != nil {
		search = "%" + strings.TrimSpace(*f.Search) + "%"
	}

	page, err := strconv.Atoi(strings.TrimSpace(*f.Page))
	if err != nil || page <= 0 {
		return nil, ErrInvalidPagination
	}

	limit, err := strconv.Atoi(strings.TrimSpace(*f.Limit))
	if err != nil || limit <= 0 {
		return nil, ErrInvalidPagination
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
	creatorIDs := make([]string, 0, len(list))
	booksIDs := make([]string, 0, len(list))
	seenCreators := make(map[string]struct{}, len(list))

	if len(list) == 0 {
		return &domain.BookListResponse{
			Data: []domain.BookResponse{},
			Pagination: domain.Pagination{
				Page:       page,
				Limit:      limit,
				Total:      count,
				TotalPages: (count + limit - 1) / limit,
			},
		}, nil
	}

	for _, v := range list {
		if _, ok := seenCreators[v.UserID]; !ok {
			seenCreators[v.UserID] = struct{}{}
			creatorIDs = append(creatorIDs, v.UserID)
		}
		booksIDs = append(booksIDs, v.ID)
	}

	users, err := s.aClient.GetUsersByIDs(ctx, creatorIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCreatorLookupError, err)
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
			return nil, fmt.Errorf("%w: user_id=%s", ErrCreatorLookupError, v.UserID)
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
			Page:       page,
			Limit:      limit,
			Total:      count,
			TotalPages: (count + limit - 1) / limit,
		},
	}, nil
}

func (s *BookService) Update(ctx context.Context, userID, bookID string, req domain.UpdateBookRequest) (*domain.BookResponse, error) {
	b, err := s.bookRepo.GetByID(ctx, bookID)
	if err != nil {
		if errors.Is(err, repository.ErrBookNotFound) {
			return nil, ErrBookNotFound
		}
		return nil, err
	}

	if b.UserID != userID {
		return nil, ErrNotBookOwner
	}

	if req.Author != nil {
		if strings.TrimSpace(*req.Author) == "" {
			return nil, ErrBookAuthorEmpty
		}
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
		if strings.TrimSpace(*req.Title) == "" {
			return nil, ErrBookTitleEmpty
		}
		b.Title = *req.Title
	}

	u, err := s.aClient.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCreatorLookupError, err)
	}

	reviewsCount, err := s.reviewRepo.GetReviewsCount(ctx, b.ID)
	if err != nil {
		return nil, err
	}

	if req.PublishedYear != nil {
		b.PublishedYear = sql.NullInt32{
			Int32: int32(*req.PublishedYear),
			Valid: true,
		}
	}

	if err = s.bookRepo.Update(ctx, b); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBookNotFound
		}
		return nil, err
	}

	return b.ToResponse(u.ToSummary(), &reviewsCount), nil
}

func (s *BookService) Delete(ctx context.Context, userID, bookID string) error {
	b, err := s.bookRepo.GetByID(ctx, bookID)
	if err != nil {
		if errors.Is(err, repository.ErrBookNotFound) {
			return ErrBookNotFound
		}
		return err
	}

	if b.UserID != userID {
		return ErrNotBookOwner
	}

	return s.bookRepo.Delete(ctx, b.ID)
}
