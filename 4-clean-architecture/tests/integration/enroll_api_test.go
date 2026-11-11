//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students-db/app/model"
	"api-students-db/app/repository"
	"api-students-db/app/service"
	"api-students-db/config"
	"api-students-db/helper"
	"api-students-db/route"
)

// api adalah aplikasi lengkap yang memakai PostgreSQL sungguhan.
type api struct {
	app *fiber.App
	jwt *helper.JWTManager
}

func newAPI(t *testing.T) api {
	t.Helper()
	raw, err := repository.NewRoleRepository(pool).LoadPermissions(context.Background())
	if err != nil {
		t.Fatalf("memuat permission: %v", err)
	}
	perms := helper.NewPermissionSet(raw)
	jwtManager := helper.NewTestJWTManager("secret-khusus-test-minimal-32-karakter", "test", time.Minute)
	courseService := service.NewCourseService(
		repository.NewCourseRepository(pool),
		repository.NewOutboxRepository(pool),
		repository.NewTxManager(pool),
	)
	app := config.NewApp(slog.New(slog.NewTextHandler(io.Discard, nil)), route.Dependencies{
		Pool:          pool,
		JWT:           jwtManager,
		Permissions:   perms,
		CourseService: courseService,
	})
	return api{app: app, jwt: jwtManager}
}

// enroll mengirim POST /courses/:id/enroll atas nama student.
func (a api) enroll(t *testing.T, student model.User, courseID int) (int, string) {
	t.Helper()
	token, err := a.jwt.GenerateAccessToken(student.ID, student.Username, "user")
	if err != nil {
		t.Errorf("membuat token: %v", err)
		return 0, ""
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/courses/"+strconv.Itoa(courseID)+"/enroll", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.app.Test(req, -1)
	if err != nil {
		t.Errorf("request enroll: %v", err)
		return 0, ""
	}
	defer resp.Body.Close()
	var body struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body.Code
}

// swap mengirim POST /enrollments/swap atas nama student.
func (a api) swap(t *testing.T, student model.User, fromCourseID, toCourseID int) (int, string) {
	t.Helper()
	token, err := a.jwt.GenerateAccessToken(student.ID, student.Username, "user")
	if err != nil {
		t.Errorf("membuat token: %v", err)
		return 0, ""
	}
	payload := fmt.Sprintf(`{"from_course_id":%d,"to_course_id":%d}`, fromCourseID, toCourseID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments/swap", strings.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.app.Test(req, -1)
	if err != nil {
		t.Errorf("request swap: %v", err)
		return 0, ""
	}
	defer resp.Body.Close()
	var body struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body.Code
}

// drop mengirim DELETE /courses/:id/enroll atas nama student.
func (a api) drop(t *testing.T, student model.User, courseID int) (int, string) {
	t.Helper()
	token, err := a.jwt.GenerateAccessToken(student.ID, student.Username, "user")
	if err != nil {
		t.Errorf("membuat token: %v", err)
		return 0, ""
	}
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/courses/"+strconv.Itoa(courseID)+"/enroll", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.app.Test(req, -1)
	if err != nil {
		t.Errorf("request drop: %v", err)
		return 0, ""
	}
	defer resp.Body.Close()
	var body struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp.StatusCode, body.Code
}

func countRows(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", sql, err)
	}
	return n
}

func TestEnrollEndpoint_WritesOutboxOnlyOnSuccess(t *testing.T) {
	a := newAPI(t)
	course := seedCourse(t, 3, 40)
	student := seedStudent(t)

	if status, _ := a.enroll(t, student, course.ID); status != http.StatusCreated {
		t.Fatalf("pendaftaran pertama: status = %d, seharusnya 201", status)
	}
	if status, code := a.enroll(t, student, course.ID); status != http.StatusConflict || code != "ALREADY_ENROLLED" {
		t.Fatalf("pendaftaran kedua: dapat %d %s, seharusnya 409 ALREADY_ENROLLED", status, code)
	}
	events := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
		student.ID, model.EventEnrollmentCreated)
	if events != 1 {
		t.Errorf("event enrollment.created = %d, seharusnya 1 (pendaftaran yang gagal tidak boleh meninggalkan jejak)", events)
	}
}

// Satu mahasiswa mengirim sepuluh pendaftaran SERENTAK ke sepuluh kelas
// 3 SKS. Batasnya 24 SKS, maka tepat delapan yang boleh berhasil.
func TestEnrollEndpoint_ConcurrentCreditLimit(t *testing.T) {
	a := newAPI(t)
	student := seedStudent(t)
	const attempts = 10
	courses := make([]model.Course, attempts)
	for i := range courses {
		courses[i] = seedCourse(t, 3, 40)
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results = map[int]int{}
		start   = make(chan struct{})
	)

	for _, c := range courses {
		wg.Add(1)
		go func(courseID int) {
			defer wg.Done()
			<-start // tunggu aba-aba agar semua request berangkat bersamaan
			status, _ := a.enroll(t, student, courseID)
			mu.Lock()
			results[status]++
			mu.Unlock()
		}(c.ID)
	}

	close(start) // aba-aba: semua goroutine dilepas pada saat yang sama
	wg.Wait()

	maxCourses := service.MaxCreditsPerSemester / 3
	if results[http.StatusCreated] != maxCourses {
		t.Errorf("berhasil = %d, seharusnya %d (hasil lengkap: %v)", results[http.StatusCreated], maxCourses, results)
	}

	enrolled := countRows(t, `SELECT COUNT(*) FROM enrollments WHERE user_id = $1`, student.ID)
	if enrolled != maxCourses {
		t.Errorf("baris enrollments = %d, seharusnya %d", enrolled, maxCourses)
	}
}

// -----------------------------------------------------------------------------
// TUGAS MANDIRI D.2 — PENAMBAHAN INTEGRATION TESTS
// -----------------------------------------------------------------------------

// Test 1 (D.2.1): POST /enrollments/swap ketika kelas tujuan sudah penuh.
// Response wajib 409 COURSE_FULL, mahasiswa TETAP terdaftar di kelas asal,
// enrolled_count kedua kelas tidak berubah, dan tidak ada event enrollment.swapped di outbox.
func TestSwapEndpoint_TargetCourseFull(t *testing.T) {
	a := newAPI(t)
	fromCourse := seedCourse(t, 3, 40)
	toCourse := seedCourse(t, 3, 1) // Kuota kelas tujuan hanya 1
	student1 := seedStudent(t)
	student2 := seedStudent(t)

	// student1 daftar di fromCourse
	if status, _ := a.enroll(t, student1, fromCourse.ID); status != http.StatusCreated {
		t.Fatalf("setup enroll student1: status = %d, want 201", status)
	}
	// student2 daftar di toCourse sehingga toCourse penuh (1/1)
	if status, _ := a.enroll(t, student2, toCourse.ID); status != http.StatusCreated {
		t.Fatalf("setup enroll student2: status = %d, want 201", status)
	}

	// student1 mencoba swap ke toCourse yang sudah penuh
	status, code := a.swap(t, student1, fromCourse.ID, toCourse.ID)
	if status != http.StatusConflict || code != "COURSE_FULL" {
		t.Fatalf("swap ke kelas penuh: dapat %d %s, seharusnya 409 COURSE_FULL", status, code)
	}

	// Verifikasi database: student1 TETAP di fromCourse
	inFrom := countRows(t, `SELECT COUNT(*) FROM enrollments WHERE user_id = $1 AND course_id = $2`, student1.ID, fromCourse.ID)
	if inFrom != 1 {
		t.Errorf("mahasiswa asal tidak lagi terdaftar di kelas lama: count = %d, want 1", inFrom)
	}

	// Verifikasi database: student1 TIDAK terdaftar di toCourse
	inTo := countRows(t, `SELECT COUNT(*) FROM enrollments WHERE user_id = $1 AND course_id = $2`, student1.ID, toCourse.ID)
	if inTo != 0 {
		t.Errorf("mahasiswa asal terdaftar di kelas tujuan: count = %d, want 0", inTo)
	}

	// Verifikasi enrolled_count kedua kelas tidak berubah (tetap 1 dan 1)
	fromCount := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, fromCourse.ID)
	toCount := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, toCourse.ID)
	if fromCount != 1 || toCount != 1 {
		t.Errorf("enrolled_count berubah: from = %d (want 1), to = %d (want 1)", fromCount, toCount)
	}

	// Verifikasi outbox: tidak ada event enrollment.swapped untuk student1
	swappedEvents := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
		student1.ID, model.EventEnrollmentSwapped)
	if swappedEvents != 0 {
		t.Errorf("ditemukan event enrollment.swapped di outbox: count = %d, want 0", swappedEvents)
	}
}

// Test 2 (D.2.2): DELETE /courses/:id/enroll berhasil keluar.
// Response 204 No Content, baris enrollment hilang, enrolled_count berkurang 1,
// dan outbox memuat tepat 1 event enrollment.dropped.
func TestDropEndpoint_Success(t *testing.T) {
	a := newAPI(t)
	course := seedCourse(t, 3, 40)
	student := seedStudent(t)

	// Mahasiswa mendaftar kelas
	if status, _ := a.enroll(t, student, course.ID); status != http.StatusCreated {
		t.Fatalf("setup enroll: status = %d, want 201", status)
	}

	// Verifikasi awal sebelum drop
	preCount := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, course.ID)
	if preCount != 1 {
		t.Fatalf("pre enrolled_count = %d, want 1", preCount)
	}

	// Mahasiswa drop kelas
	status, _ := a.drop(t, student, course.ID)
	if status != http.StatusNoContent {
		t.Fatalf("drop kelas: status = %d, seharusnya 204", status)
	}

	// Verifikasi baris enrollment hilang
	enrolled := countRows(t, `SELECT COUNT(*) FROM enrollments WHERE user_id = $1 AND course_id = $2`, student.ID, course.ID)
	if enrolled != 0 {
		t.Errorf("baris enrollment masih ada: count = %d, want 0", enrolled)
	}

	// Verifikasi enrolled_count berkurang satu (menjadi 0)
	postCount := countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, course.ID)
	if postCount != 0 {
		t.Errorf("enrolled_count tidak berkurang: post = %d, want 0", postCount)
	}

	// Verifikasi outbox memuat tepat satu event enrollment.dropped
	droppedEvents := countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
		student.ID, model.EventEnrollmentDropped)
	if droppedEvents != 1 {
		t.Errorf("event enrollment.dropped di outbox = %d, seharusnya 1", droppedEvents)
	}
}
