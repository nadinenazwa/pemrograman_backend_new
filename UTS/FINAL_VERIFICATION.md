# FINAL VERIFICATION REPORT

Laporan hasil verifikasi runtime *end-to-end* yang dilakukan dengan langsung terhubung ke database PostgreSQL dan menjalankan *server* Fiber secara sesungguhnya menggunakan _smoke test script_ (`smoke.go`).

## 1. Verifikasi Database
Database `siakad_mini` telah dibuat ulang (_drop and create_), dilakukan _migration_, dan _seeding_ secara sukses.
Verifikasi baris (row count) pada PostgreSQL:
- **`users`**: 21 rows (1 Admin + 20 Mahasiswa)
- **`students`**: 20 rows
- **`courses`**: 10 rows

Semua tabel, konstrain (seperti UNIQUE pada `enrollments` dan `students`), serta relasi `ON DELETE CASCADE` berhasil diinisialisasi.

## 2. API Endpoint E2E HTTP Test 

| Area | Test | Expected | Actual | Status |
| ---- | ---- | -------- | ------ | ------ |
| **Auth** | POST `/api/v1/auth/login` valid admin | HTTP 200 (Token) | HTTP 200 | PASS |
| **Auth** | POST `/api/v1/auth/login` valid mahasiswa | HTTP 200 (Token) | HTTP 200 | PASS |
| **Auth** | POST `/api/v1/auth/login` password salah | HTTP 401 | HTTP 401 | PASS |
| **Auth** | POST `/api/v1/auth/login` invalid format | HTTP 422 | HTTP 422 | PASS |
| **Auth** | POST `/api/v1/auth/login` > 5 req / min | HTTP 429 | HTTP 429 | PASS |
| **Auth** | GET `/api/v1/auth/me` with valid token | HTTP 200 | HTTP 200 | PASS |
| **Auth** | GET `/api/v1/auth/me` missing/invalid token | HTTP 401 | HTTP 401 | PASS |
| **Students** | GET `/api/v1/students` as Admin | HTTP 200 | HTTP 200 | PASS |
| **Students** | GET `/api/v1/students` as Mahasiswa | HTTP 403 | HTTP 403 | PASS |
| **Students** | GET `/api/v1/students?page=1&per_page=2` | 2 Items + Meta Pagination | 2 Items | PASS |
| **Students** | GET `/api/v1/students?search=Mahasiswa` | HTTP 200 (Filtered) | HTTP 200 | PASS |
| **Students** | POST `/api/v1/students` as Admin | HTTP 201 (Created) | HTTP 201 | PASS |
| **Students** | POST `/api/v1/students` Duplicate NIM/Email | HTTP 422 | HTTP 422 | PASS |
| **Students** | GET `/api/v1/students/:id` Detail as Admin | HTTP 200 | HTTP 200 | PASS |
| **Students** | GET `/api/v1/students/:id` Detail as Owner | HTTP 200 | HTTP 200 | PASS |
| **Students** | GET `/api/v1/students/:id` Mahasiswa Lain | HTTP 403 | HTTP 403 | PASS |
| **Students** | PUT `/api/v1/students/:id` | HTTP 200 (NIM Not Changed) | HTTP 200 | PASS |
| **Students** | DELETE `/api/v1/students/:id` | HTTP 204 | HTTP 204 | PASS |
| **Students** | GET deleted student | HTTP 404 | HTTP 404 | PASS |
| **Courses** | GET `/api/v1/courses` as Admin | HTTP 200 (Terisi & Sisa_Kuota) | HTTP 200 | PASS |
| **Courses** | GET `/api/v1/courses` as Mahasiswa | HTTP 200 | HTTP 200 | PASS |
| **KRS** | POST `/api/v1/enrollments` Mahasiswa Valid | HTTP 201 | HTTP 201 | PASS |
| **KRS** | POST `/api/v1/enrollments` Duplicate Course | HTTP 409 | HTTP 409 | PASS |
| **KRS** | POST `/api/v1/enrollments` Course Penuh (Kuota) | HTTP 422 ("Kuota penuh") | HTTP 422 | PASS |
| **KRS** | POST `/api/v1/enrollments` Over SKS Limit | HTTP 422 (Sebutkan sisa SKS) | HTTP 422 ("Total SKS melebihi batas. Sisa SKS: 0") | PASS |
| **KRS** | DELETE `/api/v1/enrollments/:id` milik sendiri | HTTP 204 | HTTP 204 | PASS |
| **KRS** | DELETE `/api/v1/enrollments/:id` milik mahasiswa lain | HTTP 403 | HTTP 403 | PASS |
| **KRS** | DELETE `/api/v1/enrollments/:id` tidak ditemukan | HTTP 404 | HTTP 404 | PASS |
| **KRS** | DELETE `/api/v1/enrollments/:id` sebagai admin/non-student | HTTP 403 | HTTP 403 | PASS |

> **Catatan KRS Delete:** Endpoint penghapusan krs sekarang memeriksa otorisasi pemilik data di dalam level _repository_. Mekanisme _ownership check_ ini mampu membedakan dengan benar antara `resource tidak ada` (404) dan `resource bukan milik current student` (403) tanpa mengorbankan _security_.

## 3. Concurrency / Quota
Implementasi telah dikonfirmasi di kode menggunakan `tx.Begin(ctx)` dengan _row level lock_ `SELECT ... FOR UPDATE` saat menarik `courses` sebelum insert data ke `enrollments`. Hal ini mencegah _race condition_ apabila ada beberapa Mahasiswa berlomba mengambil sisa 1 kursi mata kuliah secara eksak bersamaan. Pengujian Quota Limit juga lolos (`422 Kuota Penuh`).

## 4. Format Response
Format response telah tervalidasi menggunakan format seragam:
- **Success:**
```json
{
  "success": true,
  "message": "Data user berhasil diambil",
  "data": { ... }
}
```
- **Error:**
```json
{
  "success": false,
  "message": "Token tidak ada/salah/kedaluwarsa"
}
```
- **Validation Error:**
```json
{
  "success": false,
  "message": "Validasi gagal",
  "errors": {
    "Email": ["Key: 'LoginRequest.Email' Error:Field validation for 'Email' failed on the 'email' tag"]
  }
}
```

## 5. Git Audit
```text
C:\KuliahNadine\Backend\UTS> git status
On branch main
Your branch is ahead of 'origin/main' by 17 commits.

C:\KuliahNadine\Backend\UTS> git log --oneline -n 15
27f54c1 fix(uts): resolve unused imports and compile errors
da4f21b docs(uts): add setup and api documentation
10f260e test(uts): add api and business rule tests
...

C:\KuliahNadine\Backend\UTS> git ls-files | findstr .env
.env.example

C:\KuliahNadine\Backend\UTS> git ls-files | findstr .exe
(Kosong, tidak ada executable code yang tersimpan di remote)
```

FINAL STATUS: READY TO SUBMIT
