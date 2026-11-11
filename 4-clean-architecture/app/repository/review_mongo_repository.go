package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"api-students-db/app/model"
)

type mongoReviewRepository struct {
	col *mongo.Collection
}

func NewReviewMongoRepository(ctx context.Context, db *mongo.Database) (ReviewRepository, error) {
	col := db.Collection("reviews")
	repo := &mongoReviewRepository{col: col}
	if err := repo.ensureIndexes(ctx); err != nil {
		return nil, fmt.Errorf("gagal membuat index ulasan: %w", err)
	}
	return repo, nil
}

func (r *mongoReviewRepository) ensureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "course_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_course_created"),
		},
		{
			Keys:    bson.D{{Key: "course_id", Value: 1}, {Key: "user_id", Value: 1}},
			Options: options.Index().SetName("idx_course_user"),
		},
	}
	_, err := r.col.Indexes().CreateMany(ctx, indexes)
	return err
}

type reviewDoc struct {
	ID           bson.ObjectID `bson:"_id,omitempty"`
	CourseID     int           `bson:"course_id"`
	UserID       int           `bson:"user_id"`
	Rating       int           `bson:"rating"`
	Comment      string        `bson:"comment"`
	Tags         []string      `bson:"tags"`
	Replies      []replyDoc    `bson:"replies"`
	HelpfulVotes []int         `bson:"helpful_votes"`
	CreatedAt    time.Time     `bson:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at"`
}

type replyDoc struct {
	UserID    int       `bson:"user_id"`
	Message   string    `bson:"message"`
	CreatedAt time.Time `bson:"created_at"`
}

func toReviewModel(doc reviewDoc) model.Review {
	replies := make([]model.Reply, 0, len(doc.Replies))
	for _, rep := range doc.Replies {
		replies = append(replies, model.Reply{
			UserID:    rep.UserID,
			Message:   rep.Message,
			CreatedAt: rep.CreatedAt,
		})
	}
	votes := doc.HelpfulVotes
	if votes == nil {
		votes = []int{}
	}
	tags := doc.Tags
	if tags == nil {
		tags = []string{}
	}

	return model.Review{
		ID:           doc.ID.Hex(),
		CourseID:     doc.CourseID,
		UserID:       doc.UserID,
		Rating:       doc.Rating,
		Comment:      doc.Comment,
		Tags:         tags,
		Replies:      replies,
		HelpfulVotes: votes,
		CreatedAt:    doc.CreatedAt,
		UpdatedAt:    doc.UpdatedAt,
	}
}

func (r *mongoReviewRepository) Create(ctx context.Context, rev model.Review) (model.Review, error) {
	now := time.Now().UTC()
	doc := reviewDoc{
		CourseID:     rev.CourseID,
		UserID:       rev.UserID,
		Rating:       rev.Rating,
		Comment:      rev.Comment,
		Tags:         rev.Tags,
		Replies:      []replyDoc{},
		HelpfulVotes: []int{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	res, err := r.col.InsertOne(ctx, doc)
	if err != nil {
		return model.Review{}, err
	}

	doc.ID = res.InsertedID.(bson.ObjectID)
	return toReviewModel(doc), nil
}

func (r *mongoReviewRepository) FindByCourse(ctx context.Context, q model.ReviewQuery) ([]model.Review, error) {
	filter := bson.M{"course_id": q.CourseID}
	if q.Tag != "" {
		filter["tags"] = q.Tag
	}
	if q.Before != "" {
		t, err := time.Parse(time.RFC3339Nano, q.Before)
		if err == nil {
			filter["created_at"] = bson.M{"$lt": t}
		}
	}

	limit := int64(q.Limit)
	if limit <= 0 {
		limit = 10
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit)

	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []reviewDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	result := make([]model.Review, 0, len(docs))
	for _, d := range docs {
		result = append(result, toReviewModel(d))
	}
	return result, nil
}

func (r *mongoReviewRepository) FindByID(ctx context.Context, id string) (model.Review, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return model.Review{}, ErrInvalidID
	}

	var doc reviewDoc
	err = r.col.FindOne(ctx, bson.M{"_id": objID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return model.Review{}, ErrNotFound
	}
	if err != nil {
		return model.Review{}, err
	}

	return toReviewModel(doc), nil
}

func (r *mongoReviewRepository) Update(ctx context.Context, id string, userID int, comment string, rating int, tags []string) (model.Review, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return model.Review{}, ErrInvalidID
	}

	filter := bson.M{"_id": objID, "user_id": userID}
	update := bson.M{
		"$set": bson.M{
			"comment":    comment,
			"rating":     rating,
			"tags":       tags,
			"updated_at": time.Now().UTC(),
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updated reviewDoc
	err = r.col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
	if errors.Is(err, mongo.ErrNoDocuments) {
		// Periksa apakah dokumen ada tetapi bukan milik user
		var exists reviewDoc
		chkErr := r.col.FindOne(ctx, bson.M{"_id": objID}).Decode(&exists)
		if errors.Is(chkErr, mongo.ErrNoDocuments) {
			return model.Review{}, ErrNotFound
		}
		return model.Review{}, ErrNotOwner
	}
	if err != nil {
		return model.Review{}, err
	}

	return toReviewModel(updated), nil
}

func (r *mongoReviewRepository) Delete(ctx context.Context, id string, userID int, isAdmin bool) error {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidID
	}

	filter := bson.M{"_id": objID}
	if !isAdmin {
		filter["user_id"] = userID
	}

	res, err := r.col.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		var exists reviewDoc
		chkErr := r.col.FindOne(ctx, bson.M{"_id": objID}).Decode(&exists)
		if errors.Is(chkErr, mongo.ErrNoDocuments) {
			return ErrNotFound
		}
		return ErrNotOwner
	}
	return nil
}

func (r *mongoReviewRepository) VoteHelpful(ctx context.Context, id string, userID int) error {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidID
	}

	var rev reviewDoc
	if err := r.col.FindOne(ctx, bson.M{"_id": objID}).Decode(&rev); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrNotFound
		}
		return err
	}

	if rev.UserID == userID {
		return ErrOwnReview
	}
	for _, v := range rev.HelpfulVotes {
		if v == userID {
			return ErrAlreadyVoted
		}
	}

	_, err = r.col.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$addToSet": bson.M{"helpful_votes": userID}})
	return err
}

func (r *mongoReviewRepository) AddReply(ctx context.Context, id string, reply model.Reply) error {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidID
	}

	var rev reviewDoc
	if err := r.col.FindOne(ctx, bson.M{"_id": objID}).Decode(&rev); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrNotFound
		}
		return err
	}

	if len(rev.Replies) >= 10 {
		return ErrReplyLimit
	}

	doc := replyDoc{
		UserID:    reply.UserID,
		Message:   reply.Message,
		CreatedAt: time.Now().UTC(),
	}

	_, err = r.col.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$push": bson.M{"replies": doc}})
	return err
}

func (r *mongoReviewRepository) Summary(ctx context.Context, courseID int) (model.ReviewSummary, error) {
	cursor, err := r.col.Find(ctx, bson.M{"course_id": courseID})
	if err != nil {
		return model.ReviewSummary{}, err
	}
	defer cursor.Close(ctx)

	var docs []reviewDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return model.ReviewSummary{}, err
	}

	dist := map[string]int{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0}
	total := len(docs)
	if total == 0 {
		return model.ReviewSummary{
			CourseID:           courseID,
			AverageRating:      0,
			TotalReviews:       0,
			RatingDistribution: dist,
		}, nil
	}

	sum := 0
	for _, d := range docs {
		sum += d.Rating
		key := fmt.Sprintf("%d", d.Rating)
		dist[key]++
	}

	return model.ReviewSummary{
		CourseID:           courseID,
		AverageRating:      float64(sum) / float64(total),
		TotalReviews:       total,
		RatingDistribution: dist,
	}, nil
}
