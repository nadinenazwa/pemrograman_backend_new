package repository_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"api-students-db/app/model"
	"api-students-db/app/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// setupTestMongo menghubungkan ke MongoDB lokal untuk keperluan testing
// dan membersihkan collection test sebelum dan sesudah pengujian.
func setupTestMongo(t *testing.T) (*mongo.Database, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skipf("MongoDB lokal tidak tersedia, melewati integration test: %v", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("MongoDB ping gagal, melewati integration test: %v", err)
	}

	dbName := fmt.Sprintf("test_student_portfolio_%d", time.Now().UnixNano())
	db := client.Database(dbName)

	cleanup := func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanCancel()
		_ = db.Drop(cleanCtx)
		_ = client.Disconnect(cleanCtx)
	}

	return db, cleanup
}

// TestPortfolioRepository_UpsertDanFind menguji alur dasar pembuatan dan pembacaan portofolio
func TestPortfolioRepository_UpsertDanFind(t *testing.T) {
	db, cleanup := setupTestMongo(t)
	defer cleanup()

	ctx := context.Background()
	repo, err := repository.NewPortfolioRepository(ctx, db)
	if err != nil {
		t.Fatalf("gagal inisialisasi repository: %v", err)
	}

	studentID := 101
	upsertReq := model.UpsertPortfolioRequest{
		Headline: "Junior Backend Developer",
		Bio:      "Tertarik dengan Go dan Clean Architecture",
		Projects: []model.ProjectRequest{
			{
				Title:        "Project Alpha",
				Technologies: []string{"Go", "PostgreSQL"},
				Link:         "https://github.com/example/alpha",
				Year:         2024,
			},
		},
		Certs: []model.CertificateRequest{
			{
				Name:     "Cloud Practitioner",
				Issuer:   "AWS",
				IssuedAt: time.Now(),
			},
		},
	}

	// 1. Simpan (Upsert)
	created, err := repo.Upsert(ctx, studentID, upsertReq)
	if err != nil {
		t.Fatalf("Upsert gagal: %v", err)
	}
	if created.StudentID != studentID {
		t.Errorf("expected StudentID %d, got %d", studentID, created.StudentID)
	}
	if len(created.Projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(created.Projects))
	}
	if len(created.Certs) != 1 {
		t.Errorf("expected 1 cert, got %d", len(created.Certs))
	}

	// 2. Baca kembali (Find)
	found, err := repo.FindByStudentID(ctx, studentID)
	if err != nil {
		t.Fatalf("FindByStudentID gagal: %v", err)
	}
	if found.Headline != upsertReq.Headline {
		t.Errorf("expected headline %q, got %q", upsertReq.Headline, found.Headline)
	}

	// 3. Upsert kedua kali pada studentID yang sama -> harus mengganti isi, BUKAN membuat dokumen baru
	upsertReq2 := model.UpsertPortfolioRequest{
		Headline: "Mid-level Go Developer",
		Bio:      "Bio diperbarui",
		Projects: []model.ProjectRequest{},
		Certs:    []model.CertificateRequest{},
	}
	updated, err := repo.Upsert(ctx, studentID, upsertReq2)
	if err != nil {
		t.Fatalf("Upsert kedua gagal: %v", err)
	}
	if updated.Headline != "Mid-level Go Developer" {
		t.Errorf("expected updated headline, got %q", updated.Headline)
	}

	// Pastikan hanya ada 1 dokumen untuk mahasiswa ini di collection
	count, err := db.Collection("student_portfolios").CountDocuments(ctx, bson.M{"student_id": studentID})
	if err != nil {
		t.Fatalf("CountDocuments gagal: %v", err)
	}
	if count != 1 {
		t.Errorf("expected tepat 1 dokumen di MongoDB, ditemukan %d", count)
	}
}

// TestPortfolioRepository_UniqueIndex menguji bahwa index student_id mencegah dokumen duplikat
func TestPortfolioRepository_UniqueIndex(t *testing.T) {
	db, cleanup := setupTestMongo(t)
	defer cleanup()

	ctx := context.Background()
	_, err := repository.NewPortfolioRepository(ctx, db)
	if err != nil {
		t.Fatalf("gagal inisialisasi repository: %v", err)
	}

	coll := db.Collection("student_portfolios")
	doc1 := bson.M{
		"student_id": 999,
		"headline":   "Doc 1",
		"bio":        "Bio 1",
		"created_at": time.Now(),
		"updated_at": time.Now(),
	}
	_, err = coll.InsertOne(ctx, doc1)
	if err != nil {
		t.Fatalf("Insert pertama gagal: %v", err)
	}

	// Coba insert langsung dokumen kedua dengan student_id yang sama
	doc2 := bson.M{
		"student_id": 999,
		"headline":   "Doc 2 Duplikat",
		"bio":        "Bio 2",
		"created_at": time.Now(),
		"updated_at": time.Now(),
	}
	_, err = coll.InsertOne(ctx, doc2)
	if err == nil {
		t.Fatal("expected duplicate key error dari MongoDB unique index, tapi insert berhasil")
	}
	if !mongo.IsDuplicateKeyError(err) {
		t.Fatalf("expected IsDuplicateKeyError, got: %v", err)
	}
}

// TestPortfolioRepository_Concurrency20ProjectsLimit menguji persyaratan B:
// 1. Siapkan portofolio berisi tepat 18 proyek.
// 2. Kirim 10 request penambahan proyek secara bersamaan (goroutines).
// 3. Pastikan tepat 2 request berhasil dan 8 request ditolak (limitReached).
// 4. Pastikan jumlah akhir proyek tepat 20.
func TestPortfolioRepository_Concurrency20ProjectsLimit(t *testing.T) {
	db, cleanup := setupTestMongo(t)
	defer cleanup()

	ctx := context.Background()
	repo, err := repository.NewPortfolioRepository(ctx, db)
	if err != nil {
		t.Fatalf("gagal inisialisasi repository: %v", err)
	}

	studentID := 202

	// 1. Siapkan portofolio awal dengan tepat 18 proyek
	initialProjects := make([]model.ProjectRequest, 18)
	for i := 0; i < 18; i++ {
		initialProjects[i] = model.ProjectRequest{
			Title:        fmt.Sprintf("Initial Project %d", i+1),
			Technologies: []string{"Go", "PostgreSQL"},
			Year:         2023,
		}
	}

	_, err = repo.Upsert(ctx, studentID, model.UpsertPortfolioRequest{
		Headline: "Portfolio Concurrency Test",
		Bio:      "Bio for testing concurrency limits",
		Projects: initialProjects,
	})
	if err != nil {
		t.Fatalf("gagal membuat portofolio awal dengan 18 proyek: %v", err)
	}

	// 2. Kirim 10 request POST penambahan proyek secara bersamaan
	concurrency := 10
	var successCount int32
	var limitReachedCount int32
	var errorCount int32

	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Tunggu sinyal mulai agar semua goroutine berjalan bersamaan
			<-startBarrier

			req := model.AddProjectRequest{
				Title:        fmt.Sprintf("Concurrent Project %d", workerID),
				Technologies: []string{"Go", "Docker"},
				Year:         2025,
			}

			_, limitReached, err := repo.AddProject(context.Background(), studentID, req)
			if err != nil {
				atomic.AddInt32(&errorCount, 1)
			} else if limitReached {
				atomic.AddInt32(&limitReachedCount, 1)
			} else {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	// Lepaskan barrier untuk memicu 10 goroutine secara serentak
	close(startBarrier)
	wg.Wait()

	// 3. Verifikasi tepat 2 request berhasil dan 8 request ditolak
	if errorCount != 0 {
		t.Fatalf("ditemukan %d error tak terduga selama pengujian konkurensi", errorCount)
	}
	if successCount != 2 {
		t.Errorf("PENGUJIAN B GAGAL: expected tepat 2 request berhasil, tapi got %d", successCount)
	}
	if limitReachedCount != 8 {
		t.Errorf("PENGUJIAN B GAGAL: expected tepat 8 request ditolak (limitReached), tapi got %d", limitReachedCount)
	}

	// 4. Pastikan jumlah akhir proyek tepat 20
	finalPortfolio, err := repo.FindByStudentID(ctx, studentID)
	if err != nil {
		t.Fatalf("gagal mengambil portofolio akhir: %v", err)
	}
	if len(finalPortfolio.Projects) != 20 {
		t.Fatalf("PENGUJIAN B GAGAL: expected jumlah akhir proyek tepat 20, tapi got %d", len(finalPortfolio.Projects))
	}
}

// TestPortfolioRepository_DeleteProject menguji penghapusan proyek dengan $pull
func TestPortfolioRepository_DeleteProject(t *testing.T) {
	db, cleanup := setupTestMongo(t)
	defer cleanup()

	ctx := context.Background()
	repo, err := repository.NewPortfolioRepository(ctx, db)
	if err != nil {
		t.Fatalf("gagal inisialisasi repository: %v", err)
	}

	studentID := 303
	portfolio, err := repo.Upsert(ctx, studentID, model.UpsertPortfolioRequest{
		Headline: "Test Delete",
		Bio:      "Bio",
		Projects: []model.ProjectRequest{
			{Title: "Proyek 1", Technologies: []string{"Go"}, Year: 2024},
			{Title: "Proyek 2", Technologies: []string{"Python"}, Year: 2025},
		},
	})
	if err != nil {
		t.Fatalf("Upsert gagal: %v", err)
	}

	if len(portfolio.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(portfolio.Projects))
	}

	projectToDeleteID := portfolio.Projects[0].ID

	// 1. Hapus Proyek 1
	found, err := repo.DeleteProject(ctx, studentID, projectToDeleteID)
	if err != nil {
		t.Fatalf("DeleteProject gagal: %v", err)
	}
	if !found {
		t.Fatal("expected found=true saat menghapus proyek yang ada")
	}

	// 2. Cek sisa proyek
	updated, err := repo.FindByStudentID(ctx, studentID)
	if err != nil {
		t.Fatalf("FindByStudentID gagal: %v", err)
	}
	if len(updated.Projects) != 1 {
		t.Fatalf("expected 1 project tersisa, got %d", len(updated.Projects))
	}
	if updated.Projects[0].Title != "Proyek 2" {
		t.Errorf("expected proyek tersisa adalah Proyek 2, got %q", updated.Projects[0].Title)
	}

	// 3. Coba hapus lagi dengan ID yang sama -> harus return found=false
	foundAgain, err := repo.DeleteProject(ctx, studentID, projectToDeleteID)
	if err != nil {
		t.Fatalf("DeleteProject kedua gagal: %v", err)
	}
	if foundAgain {
		t.Fatal("expected found=false untuk proyek yang sudah terhapus")
	}
}

// TestPortfolioRepository_TechStatsAggregation menguji pipeline agregasi 10 teknologi teratas
func TestPortfolioRepository_TechStatsAggregation(t *testing.T) {
	db, cleanup := setupTestMongo(t)
	defer cleanup()

	ctx := context.Background()
	repo, err := repository.NewPortfolioRepository(ctx, db)
	if err != nil {
		t.Fatalf("gagal inisialisasi repository: %v", err)
	}

	// 1. Agregasi pada collection kosong harus mengembalikan slice kosong tanpa error
	emptyStats, err := repo.TechStats(ctx)
	if err != nil {
		t.Fatalf("TechStats gagal pada database kosong: %v", err)
	}
	if len(emptyStats) != 0 {
		t.Errorf("expected 0 stats pada database kosong, got %d", len(emptyStats))
	}

	// 2. Isi beberapa portofolio mahasiswa dengan variasi teknologi & casing
	// Mahasiswa 1: Go, Docker, PostgreSQL
	_, _ = repo.Upsert(ctx, 1, model.UpsertPortfolioRequest{
		Headline: "S1", Bio: "B1",
		Projects: []model.ProjectRequest{
			{Title: "P1", Technologies: []string{"Go", "Docker", "PostgreSQL"}, Year: 2024},
		},
	})

	// Mahasiswa 2: go (lowercase), DOCKER (uppercase), Redis
	_, _ = repo.Upsert(ctx, 2, model.UpsertPortfolioRequest{
		Headline: "S2", Bio: "B2",
		Projects: []model.ProjectRequest{
			{Title: "P2", Technologies: []string{"go", "DOCKER", "Redis"}, Year: 2025},
		},
	})

	// Mahasiswa 3: Go, Kubernetes
	_, _ = repo.Upsert(ctx, 3, model.UpsertPortfolioRequest{
		Headline: "S3", Bio: "B3",
		Projects: []model.ProjectRequest{
			{Title: "P3", Technologies: []string{"Go", "Kubernetes"}, Year: 2025},
		},
	})

	// Hitungan yang diharapkan (setelah lowercase normalisasi):
	// "go": 3
	// "docker": 2
	// "postgresql": 1
	// "redis": 1
	// "kubernetes": 1

	stats, err := repo.TechStats(ctx)
	if err != nil {
		t.Fatalf("TechStats gagal: %v", err)
	}

	if len(stats) != 5 {
		t.Fatalf("expected 5 distinct technologies, got %d (%+v)", len(stats), stats)
	}

	// Posisi 1 harus "go" dengan count 3
	if stats[0].Technology != "go" || stats[0].Count != 3 {
		t.Errorf("expected ranking 1: {go, 3}, got: %+v", stats[0])
	}

	// Posisi 2 harus "docker" dengan count 2
	if stats[1].Technology != "docker" || stats[1].Count != 2 {
		t.Errorf("expected ranking 2: {docker, 2}, got: %+v", stats[1])
	}
}

// TestPortfolioRepository_NoSQLInjectionPrevention menguji persyaratan C:
// Karakter atau operator injeksi MongoDB seperti {"$gt": ""} tidak boleh dievaluasi sebagai operator
func TestPortfolioRepository_NoSQLInjectionPrevention(t *testing.T) {
	db, cleanup := setupTestMongo(t)
	defer cleanup()

	ctx := context.Background()
	repo, err := repository.NewPortfolioRepository(ctx, db)
	if err != nil {
		t.Fatalf("gagal inisialisasi repository: %v", err)
	}

	studentID := 404
	// Payload dengan operator injeksi mentah sebagai title dan teknologi
	maliciousTitle := `{"$gt": ""}`
	maliciousTech := `{"$ne": null}`

	portfolio, err := repo.Upsert(ctx, studentID, model.UpsertPortfolioRequest{
		Headline: `{"$where": "sleep(5000)"}`,
		Bio:      `Normal bio`,
		Projects: []model.ProjectRequest{
			{
				Title:        maliciousTitle,
				Technologies: []string{maliciousTech},
				Year:         2024,
			},
		},
	})
	if err != nil {
		t.Fatalf("Upsert gagal: %v", err)
	}

	// Pastikan data tersimpan sebagai literal string biasa tanpa mengeksekusi operator
	if portfolio.Headline != `{"$where": "sleep(5000)"}` {
		t.Errorf("expected literal headline, got %q", portfolio.Headline)
	}
	if portfolio.Projects[0].Title != maliciousTitle {
		t.Errorf("expected literal project title, got %q", portfolio.Projects[0].Title)
	}
	if portfolio.Projects[0].Technologies[0] != maliciousTech {
		t.Errorf("expected literal tech, got %q", portfolio.Projects[0].Technologies[0])
	}
}
