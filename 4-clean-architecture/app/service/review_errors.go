package service

import (
	"errors"

	"api-students-db/app/repository"
	"api-students-db/helper"
)

func translateReviewError(err error) error {
	if errors.Is(err, repository.ErrInvalidID) {
		return helper.NewAppErrorWithCause(400, helper.CodeBadRequest, "ID ulasan tidak valid", err)
	}
	if errors.Is(err, repository.ErrNotFound) {
		return helper.NewAppErrorWithCause(404, helper.CodeNotFound, "ulasan tidak ditemukan", err)
	}
	if errors.Is(err, repository.ErrDuplicate) {
		return helper.NewAppErrorWithCause(409, "ALREADY_REVIEWED", "hanya boleh menulis satu ulasan per kelas", err)
	}
	if errors.Is(err, repository.ErrNotOwner) {
		return helper.NewAppErrorWithCause(403, helper.CodeForbidden, "bukan pemilik ulasan", err)
	}
	if errors.Is(err, repository.ErrOwnReview) {
		return helper.NewAppErrorWithCause(409, "OWN_REVIEW", "tidak dapat memvoting ulasan sendiri", err)
	}
	if errors.Is(err, repository.ErrAlreadyVoted) {
		return helper.NewAppErrorWithCause(409, "ALREADY_VOTED", "sudah memvoting ulasan ini", err)
	}
	if errors.Is(err, repository.ErrReplyLimit) {
		return helper.NewAppErrorWithCause(409, "REPLY_LIMIT", "batas balasan tercapai", err)
	}
	return helper.NewAppErrorWithCause(500, helper.CodeInternal, "kesalahan internal MongoDB", err)
}
