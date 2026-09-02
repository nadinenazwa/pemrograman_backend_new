# 3 – Database & Repository

REST API mahasiswa berbasis **Go + Fiber** yang terhubung ke **PostgreSQL** menggunakan connection pool [`pgx/v5`](https://github.com/jackc/pgx). Modul ini memperkenalkan pola _Repository_ untuk memisahkan logika akses data dari lapisan handler HTTP.

---

## Daftar Isi

- [Struktur Proyek](#struktur-proyek)
- [Skema Tabel](#skema-tabel)
- [Variabel Environment](#variabel-environment)
- [Persiapan Basis Data dari Nol](#persiapan-basis-data-dari-nol)
- [Menjalankan Aplikasi](#menjalankan-aplikasi)
- [Endpoint API](#endpoint-api)

---

## Struktur Proyek

```
3-database-repository/
├── app/
│   ├── model/          # Struct domain (Student, request/response)
│   └── repository/     # Interface + implementasi akses database
├── config/
│   └── env.go          # Helper baca environment variable
├── database/
│   └── postgres.go     # Inisialisasi pgxpool (connection pool)
├── migrations/
│   └── 001_create_students.sql  # DDL skema awal
├── handler.go          # Handler HTTP (Fiber)
├── helper.go           # Fungsi bantu (ctx timeout, validasi, dsb.)
├── main.go             # Entry point – routing & bootstrap
├── .env.example        # Template environment variable
└── go.mod
```

---

## Skema Tabel

### `students`

Tabel utama yang menyimpan data mahasiswa.

| Kolom        | Tipe               | Constraint                        | Keterangan                          |
|--------------|--------------------|-----------------------------------|-------------------------------------|
| `id`         | `SERIAL`           | `PRIMARY KEY`                     | ID unik, auto-increment             |
| `nim`        | `VARCHAR(20)`      | `NOT NULL`                        | Nomor Induk Mahasiswa               |
| `name`       | `VARCHAR(100)`     | `NOT NULL`                        | Nama lengkap mahasiswa              |
| `grade`      | `NUMERIC(5, 2)`    | `NOT NULL`                        | Nilai (0.00 – 100.00)               |
| `is_active`  | `BOOLEAN`          | `NOT NULL DEFAULT TRUE`           | Status keaktifan mahasiswa          |
| `created_at` | `TIMESTAMPTZ`      | `NOT NULL DEFAULT NOW()`          | Waktu data dibuat (timezone-aware)  |

### Indeks

| Nama                        | Kolom          | Tipe    | Keterangan                                     |
|-----------------------------|----------------|---------|------------------------------------------------|
| `students_nim_key`          | `nim`          | UNIQUE  | Mencegah duplikasi NIM tanpa race condition    |
| `students_name_lower_idx`   | `LOWER(name)`  | Biasa   | Mempercepat pencarian nama (case-insensitive)  |

> **DDL lengkap** → [`migrations/001_create_students.sql`](migrations/001_create_students.sql)

---

## Variabel Environment

Salin `.env.example` menjadi `.env` lalu isi nilainya:

```bash
cp .env.example .env
```

| Variabel       | Wajib | Default | Keterangan                                              |
|----------------|-------|---------|---------------------------------------------------------|
| `APP_PORT`     | Ya    | `3000`  | Port server Fiber berjalan                              |
| `DB_HOST`      | Ya    | –       | Hostname/IP server PostgreSQL (mis. `localhost`)        |
| `DB_PORT`      | Ya    | `5432`  | Port PostgreSQL                                         |
| `DB_USER`      | Ya    | –       | Username database PostgreSQL                            |
| `DB_PASSWORD`  | Ya    | –       | Password database PostgreSQL                            |
| `DB_NAME`      | Ya    | –       | Nama database yang digunakan                            |
| `DB_SSLMODE`   | Ya    | –       | Mode SSL koneksi (`disable`, `require`, `verify-full`) |
| `DB_MAX_CONNS` | Tidak | `10`    | Maksimum koneksi dalam pool                             |

Contoh isi `.env` untuk pengembangan lokal:

```env
APP_PORT=3000
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=rahasia
DB_NAME=students_db
DB_SSLMODE=disable
DB_MAX_CONNS=10
```

---

## Persiapan Basis Data dari Nol

### Prasyarat

- [PostgreSQL](https://www.postgresql.org/download/) ≥ 14 sudah terinstal dan berjalan
- [Go](https://go.dev/dl/) ≥ 1.21 sudah terinstal

### Langkah 1 – Buat Database

Masuk ke PostgreSQL via terminal:

```bash
psql -U postgres
```

Kemudian jalankan perintah berikut di dalam sesi `psql`:

```sql
CREATE DATABASE students_db;
```

Keluar dari sesi:

```sql
\q
```

### Langkah 2 – Jalankan Migrasi

Terapkan skema tabel menggunakan file migrasi yang sudah tersedia:

```bash
psql -U postgres -d students_db -f migrations/001_create_students.sql
```

Verifikasi tabel berhasil dibuat:

```bash
psql -U postgres -d students_db -c "\dt"
```

Output yang diharapkan:

```
         List of relations
 Schema |   Name   | Type  |  Owner
--------+----------+-------+----------
 public | students | table | postgres
```

### Langkah 3 – Konfigurasi Environment

```bash
# Windows (PowerShell)
Copy-Item .env.example .env

# Linux / macOS
cp .env.example .env
```

Edit file `.env` dan sesuaikan nilai setiap variabel dengan konfigurasi PostgreSQL lokal Anda.

### Langkah 4 – Instal Dependensi Go

```bash
go mod tidy
```

---

## Menjalankan Aplikasi

```bash
go run .
```

Server akan berjalan di `http://localhost:3000` (atau sesuai `APP_PORT`).

Cek kesehatan server dan koneksi database:

```bash
curl http://localhost:3000/api/v1/health
```

Respons sukses:

```json
{
  "success": true,
  "message": "server dan database berjalan"
}
```

---

## Endpoint API

Base URL: `http://localhost:3000/api/v1`

| Method   | Path              | Deskripsi                              |
|----------|-------------------|----------------------------------------|
| `GET`    | `/health`         | Cek status server & koneksi database   |
| `GET`    | `/students`       | Ambil daftar mahasiswa (dengan paging) |
| `GET`    | `/students/:id`   | Ambil data mahasiswa berdasarkan ID    |
| `POST`   | `/students`       | Tambah mahasiswa baru                  |
| `PUT`    | `/students/:id`   | Ganti seluruh data mahasiswa           |
| `PATCH`  | `/students/:id`   | Perbarui sebagian data mahasiswa       |
| `DELETE` | `/students/:id`   | Hapus mahasiswa berdasarkan ID         |

### Contoh Request Body (`POST` / `PUT`)

```json
{
  "nim": "2024001",
  "name": "Nadine Nazwa",
  "grade": 92.5,
  "is_active": true
}
```

### Contoh Request Body (`PATCH`)

Hanya kirim field yang ingin diubah:

```json
{
  "grade": 88.0,
  "is_active": false
}
```

### Query Parameter `GET /students`

| Parameter | Tipe    | Default | Keterangan                              |
|-----------|---------|---------|-----------------------------------------|
| `page`    | integer | `1`     | Halaman yang diminta                    |
| `limit`   | integer | `10`    | Jumlah data per halaman                 |
| `search`  | string  | –       | Filter nama (case-insensitive)          |

---

## Dependensi Utama

| Paket                    | Versi    | Fungsi                          |
|--------------------------|----------|---------------------------------|
| `github.com/gofiber/fiber/v2` | v2.52 | Framework HTTP                 |
| `github.com/jackc/pgx/v5`    | v5.10 | Driver & connection pool PgSQL |
| `github.com/joho/godotenv`   | v1.5  | Baca file `.env`               |
