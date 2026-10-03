# SIAKAD Mini API

SIAKAD Mini adalah layanan akademik sederhana (RESTful API) yang dibangun untuk mengelola User, Mahasiswa, Mata Kuliah, dan KRS/Enrollment menggunakan Go, Fiber, dan PostgreSQL.

## Teknologi & Library
- Go (Golang)
- Fiber (Web Framework)
- PostgreSQL (Database)
- pgx (PostgreSQL Driver & Pool)
- golang-jwt/jwt (JWT Authentication)
- bcrypt (Password Hashing)
- go-playground/validator (Input Validation)
- Docker Compose (Database Environment)

## Requirements
- Go 1.20+
- Docker & Docker Compose
- Make (opsional)

## Setup & Instalasi
1. Clone repository ini.
2. Masuk ke direktori project: `cd UTS`
3. Salin file environment: `cp .env.example .env`
4. Sesuaikan konfigurasi di `.env` jika diperlukan.
5. Jalankan PostgreSQL menggunakan Docker:
   ```bash
   docker-compose up -d
   ```
6. Install dependensi:
   ```bash
   go mod tidy
   ```
7. Jalankan Migrasi Database:
   ```bash
   go run cmd/migrate/main.go
   ```
8. Jalankan Seeder Database:
   ```bash
   go run cmd/seed/main.go
   ```

## Menjalankan Server
```bash
go run cmd/api/main.go
```
Server akan berjalan di `http://localhost:3000`.

## Menjalankan Testing
Jalankan perintah berikut untuk menjalankan unit/integration test:
```bash
go test ./tests/... -v
```

## Akun Default (Seed)
**Admin:**
- Email: `admin@siakad.test`
- Password: `admin123`

**Mahasiswa:** (Ada 20 mahasiswa, format email `student1@siakad.test` hingga `student20@siakad.test`)
- Email: `student1@siakad.test`
- Password: `123456789001` (NIM mahasiswa tersebut)

## Business Rule KRS / SKS
- IPK >= 3.00: maksimal 24 SKS
- IPK 2.50 - 2.99: maksimal 21 SKS
- IPK < 2.50: maksimal 18 SKS
- Mahasiswa tidak dapat mengambil mata kuliah yang sama pada tahun akademik yang sama.
- Kuota penuh tidak dapat diambil.
- Total SKS dihitung berdasarkan jumlah sks mata kuliah yang diambil (enrollment).

## Daftar Endpoint Utama
1. `POST /api/v1/auth/login` - Public (Login)
2. `GET /api/v1/auth/me` - Protected (Get Profile)
3. `GET /api/v1/students` - Admin (List Mahasiswa dengan pagination, search, filter, sort)
4. `POST /api/v1/students` - Admin (Create Mahasiswa)
5. `GET /api/v1/students/:id` - Admin/Mahasiswa Ybs (Get Detail Mahasiswa)
6. `PUT /api/v1/students/:id` - Admin (Update Mahasiswa)
7. `DELETE /api/v1/students/:id` - Admin (Soft Delete Mahasiswa)
8. `GET /api/v1/courses` - Protected (List Mata Kuliah)
9. `POST /api/v1/enrollments` - Mahasiswa (Ambil Mata Kuliah / KRS)
10. `DELETE /api/v1/enrollments/:id` - Mahasiswa (Hapus Mata Kuliah / Batal KRS)
