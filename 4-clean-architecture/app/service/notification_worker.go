package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"api-students-db/app/model"
	"api-students-db/app/repository"
	"api-students-db/helper"
)

type NotificationWorker struct {
	users         repository.UserRepository
	notifications repository.NotificationRepository
	courses       repository.CourseRepository
	tx            *repository.TxManager
	mailer        helper.Mailer
	logger        *slog.Logger
}

func NewNotificationWorker(
	users repository.UserRepository,
	notifications repository.NotificationRepository,
	courses repository.CourseRepository,
	tx *repository.TxManager,
	mailer helper.Mailer,
	logger *slog.Logger,
) *NotificationWorker {
	return &NotificationWorker{
		users:         users,
		notifications: notifications,
		courses:       courses,
		tx:            tx,
		mailer:        mailer,
		logger:        logger,
	}
}

// HandleSend memproses task notification:send dari antrean Redis.
// Perbaikan Bug 3: Signature t *asynq.Task (pointer) agar memenuhi antarmuka asynq.HandlerFunc.
func (w *NotificationWorker) HandleSend(ctx context.Context, t *asynq.Task) error {
	var job model.NotificationJob
	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("%w: payload task rusak: %v", asynq.SkipRetry, err)
	}

	// Penanganan event Tugas Mandiri D.2: course.quota_changed
	if job.EventType == model.EventCourseQuotaChanged {
		return w.handleCourseQuotaChanged(ctx, job)
	}

	var event model.EnrollmentEvent
	if err := json.Unmarshal(job.Payload, &event); err != nil {
		return fmt.Errorf("%w: payload event %d rusak: %v", asynq.SkipRetry, job.OutboxID, err)
	}

	user, err := w.users.FindByID(ctx, event.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("%w: user %d tidak ditemukan", asynq.SkipRetry, event.UserID)
	}
	if err != nil {
		return err
	}

	msg, err := BuildMessage(job.EventType, event, user)
	if err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	alreadySent := false
	// Perbaikan Bug 7: Claim dan mailer.Send disatukan di dalam transaksi WithTxOnce
	// Jika mailer gagal sementara, tx di-rollback sehingga retry berikutnya masih dapat mencoba.
	err = w.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
		claimed, err := w.notifications.WithTx(tx).Claim(ctx, model.Notification{
			OutboxID: job.OutboxID,
			UserID:   user.ID,
			Channel:  "email",
			Subject:  msg.Subject,
			Body:     msg.Body,
		})
		if err != nil {
			return err
		}
		if !claimed {
			alreadySent = true
			return nil
		}
		return w.mailer.Send(ctx, msg)
	})

	// Perbaikan Bug 8: Error kegagalan permanen dibungkus dengan %w: %w asynq.SkipRetry
	if errors.Is(err, helper.ErrPermanentDelivery) {
		return fmt.Errorf("%w: %w", asynq.SkipRetry, err)
	}
	if err != nil {
		return err
	}

	retry, _ := asynq.GetRetryCount(ctx)
	w.logger.Info("notification_processed",
		slog.Int64("outbox_id", job.OutboxID),
		slog.String("event", job.EventType),
		slog.Int("retry", retry),
		slog.Bool("already_sent", alreadySent))
	return nil
}

// handleCourseQuotaChanged mengimplementasikan fan-out notifikasi ke seluruh mahasiswa terdaftar (Tugas Mandiri D.2)
func (w *NotificationWorker) handleCourseQuotaChanged(ctx context.Context, job model.NotificationJob) error {
	var event model.CourseQuotaChangedEvent
	if err := json.Unmarshal(job.Payload, &event); err != nil {
		return fmt.Errorf("%w: payload quota changed %d rusak: %v", asynq.SkipRetry, job.OutboxID, err)
	}

	userIDs, err := w.courses.FindEnrolledUserIDs(ctx, event.CourseID)
	if err != nil {
		return err
	}

	allAlreadySent := true
	for _, userID := range userIDs {
		user, err := w.users.FindByID(ctx, userID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				continue
			}
			return err
		}

		msg := model.Message{
			To:      user.Email,
			Subject: fmt.Sprintf("KRS: kuota %s %s berubah menjadi %d", event.CourseCode, event.CourseName, event.NewQuota),
			Body:    fmt.Sprintf("Halo %s, kuota kelas %s %s berubah menjadi %d.", user.Username, event.CourseCode, event.CourseName, event.NewQuota),
		}

		studentAlreadySent := false
		err = w.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
			claimed, err := w.notifications.WithTx(tx).Claim(ctx, model.Notification{
				OutboxID: job.OutboxID,
				UserID:   user.ID,
				Channel:  "email",
				Subject:  msg.Subject,
				Body:     msg.Body,
			})
			if err != nil {
				return err
			}
			if !claimed {
				studentAlreadySent = true
				return nil
			}
			return w.mailer.Send(ctx, msg)
		})

		// Jika gagal permanen untuk user tertentu, abaikan agar tidak menghambat mahasiswa lain
		if errors.Is(err, helper.ErrPermanentDelivery) {
			continue
		}
		if err != nil {
			// Kegagalan sementara: hentikan loop dan picu retry Asynq.
			// Mahasiswa yang sudah menerima email (studentAlreadySent/sudah di-commit)
			// tidak akan dikirimi email ulang pada attempt selanjutnya berkat Claim idempoten.
			return err
		}

		if !studentAlreadySent {
			allAlreadySent = false
		}
	}

	retry, _ := asynq.GetRetryCount(ctx)
	w.logger.Info("notification_processed",
		slog.Int64("outbox_id", job.OutboxID),
		slog.String("event", job.EventType),
		slog.Int("retry", retry),
		slog.Bool("already_sent", allAlreadySent))
	return nil
}
