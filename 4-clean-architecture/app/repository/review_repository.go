package repository

import (
	"context"
	"errors"

	"api-students-db/app/model"
)

var (
	ErrInvalidID    = errors.New("id tidak valid")
	ErrNotOwner     = errors.New("bukan pemilik ulasan")
	ErrOwnReview    = errors.New("tidak dapat memvoting ulasan sendiri")
	ErrAlreadyVoted = errors.New("sudah memvoting ulasan ini")
	ErrReplyLimit   = errors.New("batas balasan tercapai")
)

type ReviewRepository interface {
	Create(ctx context.Context, r model.Review) (model.Review, error)
	FindByCourse(ctx context.Context, q model.ReviewQuery) ([]model.Review, error)
	FindByID(ctx context.Context, id string) (model.Review, error)
	Update(ctx context.Context, id string, userID int, comment string, rating int, tags []string) (model.Review, error)
	Delete(ctx context.Context, id string, userID int, isAdmin bool) error
	VoteHelpful(ctx context.Context, id string, userID int) error
	AddReply(ctx context.Context, id string, reply model.Reply) error
	Summary(ctx context.Context, courseID int) (model.ReviewSummary, error)
}
