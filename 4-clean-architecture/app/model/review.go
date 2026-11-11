package model

import "time"

// Review merepresentasikan data ulasan untuk mata kuliah di MongoDB.
type Review struct {
	ID           string    `json:"id" bson:"_id,omitempty"`
	CourseID     int       `json:"course_id" bson:"course_id"`
	UserID       int       `json:"user_id" bson:"user_id"`
	Rating       int       `json:"rating" validate:"required,min=1,max=5" bson:"rating"`
	Comment      string    `json:"comment" validate:"required,min=5,max=1000" bson:"comment"`
	Tags         []string  `json:"tags" bson:"tags"`
	Replies      []Reply   `json:"replies" bson:"replies"`
	HelpfulVotes []int     `json:"helpful_votes" bson:"helpful_votes"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" bson:"updated_at"`
}

// Reply merepresentasikan balasan pada ulasan.
type Reply struct {
	UserID    int       `json:"user_id" bson:"user_id"`
	Message   string    `json:"message" validate:"required,min=2,max=500" bson:"message"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

// ReviewQuery digunakan untuk memfilter dan memaginasi daftar ulasan.
type ReviewQuery struct {
	CourseID int
	Tag      string
	Before   string
	Limit    int
}

// CreateReviewRequest adalah payload untuk membuat ulasan baru.
type CreateReviewRequest struct {
	Rating  int      `json:"rating" validate:"required,min=1,max=5"`
	Comment string   `json:"comment" validate:"required,min=5,max=1000"`
	Tags    []string `json:"tags,omitempty"`
}

// UpdateReviewRequest adalah payload untuk memperbarui ulasan.
type UpdateReviewRequest struct {
	Rating  *int      `json:"rating,omitempty" validate:"omitempty,min=1,max=5"`
	Comment *string   `json:"comment,omitempty" validate:"omitempty,min=5,max=1000"`
	Tags    *[]string `json:"tags,omitempty"`
}

// ReplyRequest adalah payload untuk menambahkan balasan pada ulasan.
type ReplyRequest struct {
	Message string `json:"message" validate:"required,min=2,max=500"`
}

// ReviewSummary merepresentasikan statistik ulasan sebuah kelas.
type ReviewSummary struct {
	CourseID           int            `json:"course_id"`
	AverageRating      float64        `json:"average_rating"`
	TotalReviews       int            `json:"total_reviews"`
	RatingDistribution map[string]int `json:"rating_distribution"`
}

// ReviewPage adalah response GET /courses/{id}/reviews.
type ReviewPage struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    []Review       `json:"data"`
	Meta    ReviewPageMeta `json:"meta"`
}

// ReviewPageMeta berisi cursor pagination metadata untuk ulasan.
type ReviewPageMeta struct {
	Limit      int    `json:"limit"`
	HasMore    bool   `json:"has_more"`
	NextBefore string `json:"next_before"`
}
