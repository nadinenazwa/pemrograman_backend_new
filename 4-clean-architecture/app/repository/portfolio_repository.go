package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"api-students-db/app/model"
)

// ===== Internal BSON document structs (tidak diekspos ke luar package) =====

// portfolioDoc adalah representasi dokumen MongoDB yang sebenarnya.
// BSON tags hanya ada di sini — tidak menyebar ke model atau handler.
type portfolioDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	StudentID int           `bson:"student_id"`
	Headline  string        `bson:"headline"`
	Bio       string        `bson:"bio"`
	Projects  []projectDoc  `bson:"projects"`
	Certs     []certDoc     `bson:"certificates"`
	CreatedAt time.Time     `bson:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at"`
}

type projectDoc struct {
	ID           bson.ObjectID `bson:"_id,omitempty"`
	Title        string        `bson:"title"`
	Technologies []string      `bson:"technologies"` // disimpan lowercase
	Link         string        `bson:"link"`
	Year         int           `bson:"year"`
}

type certDoc struct {
	Name     string    `bson:"name"`
	Issuer   string    `bson:"issuer"`
	IssuedAt time.Time `bson:"issued_at"`
}

// ===== Interface =====

// PortfolioRepository mendefinisikan operasi MongoDB untuk portofolio mahasiswa.
type PortfolioRepository interface {
	// FindByStudentID mencari portofolio berdasarkan student_id.
	// Mengembalikan ErrNotFound jika belum ada.
	FindByStudentID(ctx context.Context, studentID int) (model.StudentPortfolio, error)

	// Upsert membuat atau mengganti seluruh portofolio (semantik PUT).
	// Menjamin satu dokumen per student_id berkat unique index.
	Upsert(ctx context.Context, studentID int, req model.UpsertPortfolioRequest) (model.StudentPortfolio, error)

	// AddProject menambahkan satu proyek ke array projects secara atomik.
	// Mengembalikan (portfolio, false, nil) jika berhasil.
	// Mengembalikan (zero, true, nil) jika batas 20 proyek sudah tercapai.
	// Keamanan: pemeriksaan batas dan push dilakukan dalam satu operasi MongoDB.
	AddProject(ctx context.Context, studentID int, req model.AddProjectRequest) (model.StudentPortfolio, bool, error)

	// DeleteProject menghapus satu proyek dari array projects menggunakan $pull.
	// Mengembalikan (true, nil) jika ditemukan dan dihapus.
	// Mengembalikan (false, nil) jika proyek tidak ditemukan (portofolio tetap ada).
	DeleteProject(ctx context.Context, studentID int, projectID string) (bool, error)

	// TechStats mengembalikan 10 teknologi teratas menggunakan aggregation pipeline.
	// Mengembalikan slice kosong jika collection belum ada data.
	TechStats(ctx context.Context) ([]model.TechStat, error)
}

// ===== Implementasi =====

const (
	collectionName = "student_portfolios"
	maxProjects    = 20
)

type mongoPortfolioRepository struct {
	col *mongo.Collection
}

// NewPortfolioRepository membuat PortfolioRepository baru dan memastikan index sudah terbuat.
// Gagal dengan error jika index tidak dapat dibuat — aplikasi tidak boleh start tanpa index.
func NewPortfolioRepository(ctx context.Context, db *mongo.Database) (PortfolioRepository, error) {
	col := db.Collection(collectionName)
	repo := &mongoPortfolioRepository{col: col}
	if err := repo.ensureIndexes(ctx); err != nil {
		return nil, fmt.Errorf("gagal membuat index MongoDB: %w", err)
	}
	return repo, nil
}

// ensureIndexes membuat semua index yang diperlukan.
// Idempotent: aman dipanggil berulang kali.
func (r *mongoPortfolioRepository) ensureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			// Unique index: satu mahasiswa hanya boleh punya satu portofolio
			Keys:    bson.D{{Key: "student_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_student_id_unique"),
		},
	}
	_, err := r.col.Indexes().CreateMany(ctx, indexes)
	return err
}

// ===== Mapper helpers =====

// toModel mengonversi internal portfolioDoc ke model.StudentPortfolio (API layer).
func toModel(doc portfolioDoc) model.StudentPortfolio {
	projects := make([]model.Project, 0, len(doc.Projects))
	for _, p := range doc.Projects {
		projects = append(projects, model.Project{
			ID:           p.ID.Hex(),
			Title:        p.Title,
			Technologies: p.Technologies,
			Link:         p.Link,
			Year:         p.Year,
		})
	}

	certs := make([]model.Certificate, 0, len(doc.Certs))
	for _, cert := range doc.Certs {
		certs = append(certs, model.Certificate{
			Name:     cert.Name,
			Issuer:   cert.Issuer,
			IssuedAt: cert.IssuedAt,
		})
	}

	return model.StudentPortfolio{
		ID:        doc.ID.Hex(),
		StudentID: doc.StudentID,
		Headline:  doc.Headline,
		Bio:       doc.Bio,
		Projects:  projects,
		Certs:     certs,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}
}

// normalizeTechs menormalisasi setiap teknologi menjadi lowercase.
// Ini memastikan "Go" dan "go" dihitung sebagai teknologi yang sama di stats.
func normalizeTechs(techs []string) []string {
	result := make([]string, len(techs))
	for i, t := range techs {
		result[i] = strings.ToLower(strings.TrimSpace(t))
	}
	return result
}

// ===== Interface methods =====

func (r *mongoPortfolioRepository) FindByStudentID(
	ctx context.Context, studentID int,
) (model.StudentPortfolio, error) {
	var doc portfolioDoc
	err := r.col.FindOne(ctx, bson.M{"student_id": studentID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.StudentPortfolio{}, ErrNotFound
		}
		return model.StudentPortfolio{}, fmt.Errorf("mengambil portofolio: %w", err)
	}
	return toModel(doc), nil
}

func (r *mongoPortfolioRepository) Upsert(
	ctx context.Context, studentID int, req model.UpsertPortfolioRequest,
) (model.StudentPortfolio, error) {
	now := time.Now().UTC()

	// Konversi request ke internal doc structs
	projects := make([]projectDoc, 0, len(req.Projects))
	for _, p := range req.Projects {
		projects = append(projects, projectDoc{
			ID:           bson.NewObjectID(),
			Title:        p.Title,
			Technologies: normalizeTechs(p.Technologies),
			Link:         p.Link,
			Year:         p.Year,
		})
	}

	certs := make([]certDoc, 0, len(req.Certs))
	for _, c := range req.Certs {
		certs = append(certs, certDoc{
			Name:     c.Name,
			Issuer:   c.Issuer,
			IssuedAt: c.IssuedAt.UTC(),
		})
	}

	filter := bson.M{"student_id": studentID}
	update := bson.M{
		"$set": bson.M{
			"headline":    req.Headline,
			"bio":         req.Bio,
			"projects":    projects,
			"certificates": certs,
			"updated_at":  now,
		},
		"$setOnInsert": bson.M{
			"student_id": studentID,
			"created_at": now,
		},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var doc portfolioDoc
	err := r.col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&doc)
	if err != nil {
		return model.StudentPortfolio{}, fmt.Errorf("upsert portofolio: %w", err)
	}

	return toModel(doc), nil
}

func (r *mongoPortfolioRepository) AddProject(
	ctx context.Context, studentID int, req model.AddProjectRequest,
) (model.StudentPortfolio, bool, error) {
	newProject := projectDoc{
		ID:           bson.NewObjectID(),
		Title:        req.Title,
		Technologies: normalizeTechs(req.Technologies),
		Link:         req.Link,
		Year:         req.Year,
	}
	now := time.Now().UTC()

	// Filter bersyarat: hanya cocok jika array projects belum mencapai maxProjects elemen.
	// Indeks array berbasis 0, jadi "projects.19" tidak ada berarti panjang < 20.
	// Ini menjamin pemeriksaan batas + push terjadi secara atomik di MongoDB.
	filter := bson.M{
		"student_id": studentID,
		fmt.Sprintf("projects.%d", maxProjects-1): bson.M{"$exists": false},
	}

	update := bson.M{
		"$push": bson.M{"projects": newProject},
		"$set":  bson.M{"updated_at": now},
		"$setOnInsert": bson.M{
			"headline":     "",
			"bio":          "",
			"certificates": bson.A{},
			"created_at":   now,
		},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var doc portfolioDoc
	err := r.col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&doc)
	if err != nil {
		// Jika filter tidak cocok (limit sudah tercapai), tidak ada dokumen yang diupdate.
		// Dalam kasus upsert dengan filter yang tidak cocok pada dokumen existing,
		// MongoDB akan mencoba insert baru — tapi unique index mencegahnya.
		// Cek apakah portofolio ada dan sudah full.
		if mongo.IsDuplicateKeyError(err) || errors.Is(err, mongo.ErrNoDocuments) {
			// Cek apakah limit sudah tercapai
			limitReached, checkErr := r.isProjectLimitReached(ctx, studentID)
			if checkErr != nil {
				return model.StudentPortfolio{}, false, fmt.Errorf("cek limit proyek: %w", checkErr)
			}
			if limitReached {
				return model.StudentPortfolio{}, true, nil
			}
			return model.StudentPortfolio{}, false, fmt.Errorf("menambahkan proyek: %w", err)
		}
		return model.StudentPortfolio{}, false, fmt.Errorf("menambahkan proyek: %w", err)
	}

	return toModel(doc), false, nil
}

// isProjectLimitReached mengecek apakah portfolio student sudah mencapai batas proyek.
// Digunakan sebagai fallback saat AddProject gagal karena limit.
func (r *mongoPortfolioRepository) isProjectLimitReached(
	ctx context.Context, studentID int,
) (bool, error) {
	filter := bson.M{
		"student_id":                              studentID,
		fmt.Sprintf("projects.%d", maxProjects-1): bson.M{"$exists": true},
	}
	count, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *mongoPortfolioRepository) DeleteProject(
	ctx context.Context, studentID int, projectID string,
) (bool, error) {
	// Parse projectID ke ObjectID untuk memastikan formatnya valid
	oid, err := bson.ObjectIDFromHex(projectID)
	if err != nil {
		return false, ErrInvalidProjectID
	}

	now := time.Now().UTC()
	// Filter menyertakan projects._id sehingga hanya cocok jika proyek ada di dalam array
	filter := bson.M{
		"student_id":   studentID,
		"projects._id": oid,
	}
	update := bson.M{
		"$pull": bson.M{"projects": bson.M{"_id": oid}},
		"$set":  bson.M{"updated_at": now},
	}

	result, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return false, fmt.Errorf("menghapus proyek: %w", err)
	}

	if result.ModifiedCount > 0 {
		return true, nil
	}

	// Jika tidak termodifikasi, cek apakah portofolio mahasiswa memang ada
	count, err := r.col.CountDocuments(ctx, bson.M{"student_id": studentID})
	if err != nil {
		return false, fmt.Errorf("cek keberadaan portofolio: %w", err)
	}
	if count == 0 {
		return false, ErrNotFound
	}

	return false, nil
}

func (r *mongoPortfolioRepository) TechStats(
	ctx context.Context,
) ([]model.TechStat, error) {
	// Aggregation pipeline:
	// 1. $unwind projects array
	// 2. $unwind technologies array (per project)
	// 3. $group by technology, count occurrences
	// 4. $sort by count descending
	// 5. $limit 10
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$unwind", Value: "$projects"}},
		bson.D{{Key: "$unwind", Value: "$projects.technologies"}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$projects.technologies"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}}}},
		bson.D{{Key: "$limit", Value: 10}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "technology", Value: "$_id"},
			{Key: "count", Value: 1},
		}}},
	}

	cursor, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregasi statistik teknologi: %w", err)
	}
	defer cursor.Close(ctx)

	stats := []model.TechStat{} // inisialisasi slice kosong, bukan nil
	for cursor.Next(ctx) {
		var item struct {
			Technology string `bson:"technology"`
			Count      int    `bson:"count"`
		}
		if err := cursor.Decode(&item); err != nil {
			return nil, fmt.Errorf("decode statistik: %w", err)
		}
		stats = append(stats, model.TechStat{
			Technology: item.Technology,
			Count:      item.Count,
		})
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterasi cursor: %w", err)
	}

	return stats, nil
}

// ===== Sentinel errors khusus portfolio =====

var (
	// ErrInvalidProjectID digunakan saat format project ID bukan ObjectID yang valid.
	ErrInvalidProjectID = errors.New("format project ID tidak valid")
)
