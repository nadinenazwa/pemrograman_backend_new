package service

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"api-students-db/app/model"
	"api-students-db/app/repository"
	"api-students-db/helper"
)

type ReviewService struct {
	reviews repository.ReviewRepository
	courses repository.CourseRepository
	perms   *helper.PermissionSet
}

func NewReviewService(
	reviews repository.ReviewRepository,
	courses repository.CourseRepository,
	perms *helper.PermissionSet,
) *ReviewService {
	return &ReviewService{reviews: reviews, courses: courses, perms: perms}
}

// Create godoc
//
// @Summary      Menulis ulasan untuk sebuah kelas
// @Description  Hanya peserta kelas. Satu mahasiswa hanya boleh menulis satu ulasan per kelas.
// @Tags         reviews
// @Accept       json
// @Produce      json
// @Param        id   path      int                       true "ID kelas"
// @Param        body body      model.CreateReviewRequest true "Isi ulasan"
// @Success      201  {object}  model.WebResponse{data=model.Review}
// @Failure      403  {object}  model.ErrorResponse "NOT_ENROLLED"
// @Failure      409  {object}  model.ErrorResponse "ALREADY_REVIEWED"
// @Failure      422  {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /courses/{id}/reviews [post]
func (s *ReviewService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	courseID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.CreateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.NewAppError(fiber.StatusBadRequest, helper.CodeBadRequest, "body bukan JSON yang valid")
	}

	if req.Rating < 1 || req.Rating > 5 {
		return helper.NewValidationError(map[string]string{"rating": "rating harus bernilai antara 1 sampai 5"})
	}
	if len(req.Comment) < 5 {
		return helper.NewValidationError(map[string]string{"comment": "komentar minimal 5 karakter"})
	}

	// 1. Verifikasi kelas ada di PostgreSQL
	_, err := s.courses.FindByID(ctx, courseID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NewAppError(fiber.StatusNotFound, helper.CodeNotFound, "kelas tidak ditemukan")
		}
		return helper.Internal(err)
	}

	// 2. Verifikasi mahasiswa terdaftar di kelas
	enrolled, err := s.courses.IsEnrolled(ctx, courseID, user.UserID)
	if err != nil {
		return helper.Internal(err)
	}
	if !enrolled {
		return helper.NewAppError(fiber.StatusForbidden, "NOT_ENROLLED", "hanya peserta kelas yang dapat menulis ulasan")
	}

	review := model.Review{
		CourseID: courseID,
		UserID:   user.UserID,
		Rating:   req.Rating,
		Comment:  req.Comment,
		Tags:     NormalizeTags(req.Tags),
	}

	created, err := s.reviews.Create(ctx, review)
	if err != nil {
		return translateReviewError(err)
	}

	return helper.Success(c, fiber.StatusCreated, "ulasan berhasil dibuat", created)
}

// ListByCourse godoc
//
// @Summary      Daftar ulasan sebuah kelas, terbaru lebih dulu
// @Description  Cursor pagination: kirim meta.next_before sebagai parameter before untuk halaman berikutnya.
// @Tags         reviews
// @Produce      json
// @Param        id     path     int    true  "ID kelas"
// @Param        tag    query    string false "Saring berdasarkan tag"
// @Param        before query    string false "Cursor: id ulasan terakhir yang sudah dilihat"
// @Param        limit  query    int    false "Jumlah per halaman, 1 sampai 50" default(10)
// @Success      200    {object} model.ReviewPage
// @Failure      400    {object} model.ErrorResponse
// @Security     BearerAuth
// @Router       /courses/{id}/reviews [get]
func (s *ReviewService) ListByCourse(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courseID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	q := model.ReviewQuery{
		CourseID: courseID,
		Tag:      c.Query("tag"),
		Before:   c.Query("before"),
		Limit:    limit,
	}

	reviews, err := s.reviews.FindByCourse(ctx, q)
	if err != nil {
		return translateReviewError(err)
	}

	hasMore := len(reviews) > limit
	if hasMore {
		reviews = reviews[:limit]
	}

	nextBefore := ""
	if len(reviews) > 0 {
		nextBefore = reviews[len(reviews)-1].ID
	}

	return c.JSON(model.ReviewPage{
		Success: true,
		Message: "daftar ulasan berhasil diambil",
		Data:    reviews,
		Meta: model.ReviewPageMeta{
			Limit:      limit,
			HasMore:    hasMore,
			NextBefore: nextBefore,
		},
	})
}

// Summary godoc
//
// @Summary      Ringkasan statistik ulasan satu kelas
// @Tags         reviews
// @Produce      json
// @Param        id   path     int true "ID kelas"
// @Success      200  {object} model.WebResponse{data=model.ReviewSummary}
// @Failure      400  {object} model.ErrorResponse
// @Failure      404  {object} model.ErrorResponse
// @Security     BearerAuth
// @Router       /courses/{id}/reviews/summary [get]
func (s *ReviewService) Summary(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courseID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	summary, err := s.reviews.Summary(ctx, courseID)
	if err != nil {
		return translateReviewError(err)
	}

	return helper.Success(c, fiber.StatusOK, "ringkasan ulasan berhasil diambil", summary)
}

// Get godoc
//
// @Summary      Detail satu ulasan
// @Tags         reviews
// @Produce      json
// @Param        id   path     string true "ID ulasan"
// @Success      200  {object} model.WebResponse{data=model.Review}
// @Failure      400  {object} model.ErrorResponse
// @Failure      404  {object} model.ErrorResponse
// @Security     BearerAuth
// @Router       /reviews/{id} [get]
func (s *ReviewService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id := c.Params("id")
	if id == "" {
		return helper.BadRequest("id ulasan wajib diisi")
	}

	review, err := s.reviews.FindByID(ctx, id)
	if err != nil {
		return translateReviewError(err)
	}

	return helper.Success(c, fiber.StatusOK, "detail ulasan berhasil diambil", review)
}

// Update godoc
//
// @Summary      Mengubah ulasan milik sendiri (PATCH)
// @Tags         reviews
// @Accept       json
// @Produce      json
// @Param        id   path     string                    true "ID ulasan"
// @Param        body body     model.UpdateReviewRequest true "Perubahan ulasan"
// @Success      200  {object} model.WebResponse{data=model.Review}
// @Failure      400  {object} model.ErrorResponse
// @Failure      403  {object} model.ErrorResponse
// @Failure      404  {object} model.ErrorResponse
// @Security     BearerAuth
// @Router       /reviews/{id} [patch]
func (s *ReviewService) Update(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id := c.Params("id")
	var req model.UpdateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body bukan JSON yang valid")
	}

	var comment string
	if req.Comment != nil {
		comment = *req.Comment
	}
	var rating int
	if req.Rating != nil {
		rating = *req.Rating
	}
	var tags []string
	if req.Tags != nil {
		tags = NormalizeTags(*req.Tags)
	}

	updated, err := s.reviews.Update(ctx, id, user.UserID, comment, rating, tags)
	if err != nil {
		return translateReviewError(err)
	}

	return helper.Success(c, fiber.StatusOK, "ulasan berhasil diperbarui", updated)
}

// Delete godoc
//
// @Summary      Menghapus ulasan
// @Tags         reviews
// @Param        id   path string true "ID ulasan"
// @Success      204  "Ulasan berhasil dihapus; tanpa body"
// @Failure      400  {object} model.ErrorResponse
// @Failure      403  {object} model.ErrorResponse
// @Failure      404  {object} model.ErrorResponse
// @Security     BearerAuth
// @Router       /reviews/{id} [delete]
func (s *ReviewService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id := c.Params("id")
	isAdmin := user.Role == "admin" || (s.perms != nil && s.perms.Has(user.Role, "review:moderate"))

	if err := s.reviews.Delete(ctx, id, user.UserID, isAdmin); err != nil {
		return translateReviewError(err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// AddReply godoc
//
// @Summary      Menambahkan balasan pada ulasan
// @Tags         reviews
// @Accept       json
// @Produce      json
// @Param        id   path     string             true "ID ulasan"
// @Param        body body     model.ReplyRequest true "Pesan balasan"
// @Success      201  {object} model.WebResponse{data=model.Reply}
// @Failure      400  {object} model.ErrorResponse
// @Failure      404  {object} model.ErrorResponse
// @Failure      409  {object} model.ErrorResponse "REPLY_LIMIT"
// @Security     BearerAuth
// @Router       /reviews/{id}/replies [post]
func (s *ReviewService) AddReply(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id := c.Params("id")
	var req model.ReplyRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body bukan JSON yang valid")
	}

	reply := model.Reply{
		UserID:  user.UserID,
		Message: req.Message,
	}

	if err := s.reviews.AddReply(ctx, id, reply); err != nil {
		return translateReviewError(err)
	}

	return helper.Success(c, fiber.StatusCreated, "balasan berhasil ditambahkan", reply)
}

// VoteHelpful godoc
//
// @Summary      Menandai ulasan bermanfaat (voting)
// @Tags         reviews
// @Produce      json
// @Param        id   path     string true "ID ulasan"
// @Success      200  {object} model.WebResponse{data=string}
// @Failure      400  {object} model.ErrorResponse
// @Failure      404  {object} model.ErrorResponse
// @Failure      409  {object} model.ErrorResponse "OWN_REVIEW atau ALREADY_VOTED"
// @Security     BearerAuth
// @Router       /reviews/{id}/helpful [post]
func (s *ReviewService) VoteHelpful(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id := c.Params("id")
	if err := s.reviews.VoteHelpful(ctx, id, user.UserID); err != nil {
		return translateReviewError(err)
	}

	return helper.Success(c, fiber.StatusOK, "voting berhasil dicatat", "helpful")
}
