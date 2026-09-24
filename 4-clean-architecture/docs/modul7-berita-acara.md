# Berita Acara Perbaikan Bug Modul 7

Sesuai instruksi, berikut adalah 9 bug yang ditemukan pada kode Bagian B (modul 7 existing code) dan telah diperbaiki:

## Compiler Errors (3 Bug)

1. **Lokasi:** `helper/validator.go`
   - **Jenis:** Compiler Error
   - **Penyebab:** Memanggil `RegisterTagNameFunc` dengan parameter `func(fld interface{ Tag(string) string }) string`, padahal versi baru `validator/v10` membutuhkan `func(fld reflect.StructField) string`.
   - **Perbaikan:** Mengganti signature fungsi anonim menjadi `func(fld reflect.StructField) string` dan menggunakan `fld.Tag.Get("json")`.
   - **Evidence:** Dapat dilihat pada commit history perbaikan `helper/validator.go`.

2. **Lokasi:** `app/service/student_service.go` pada method `Patch`
   - **Jenis:** Compiler Error
   - **Penyebab:** Menggunakan `req.IsEmptyPatch` tanpa tanda kurung `()`, padahal itu adalah pemanggilan method, bukan field statis.
   - **Perbaikan:** Menambahkan tanda kurung menjadi `req.IsEmptyPatch()`.
   - **Evidence:** Telah diperbaiki pada `app/service/student_service.go`.

3. **Lokasi:** `app/service/student_service.go` pada method `List`
   - **Jenis:** Compiler Error
   - **Penyebab:** Assignment kembalian `strconv.Atoi(c.Query("limit", "10"))` menggunakan `limit := ...` sehingga `limit` bertipe `(int, error)` yang menyebabkan type mismatch saat dimasukkan ke struct.
   - **Perbaikan:** Mengubah menjadi `limit, _ := strconv.Atoi(...)`.
   - **Evidence:** Telah diperbaiki pada `app/service/student_service.go`.

## Behavioral Errors (6 Bug)

4. **Lokasi:** `app/service/student_service.go` (Method Create)
   - **Jenis:** Behavioral Error
   - **Penyebab:** Handler POST mengembalikan status 200 OK ketika berhasil membuat data baru.
   - **Perbaikan:** Mengubah response menjadi 201 Created menggunakan `helper.Created` beserta header Location.
   - **Evidence:** Dapat dilihat dari return `helper.Created(c, ...)` pada method Create.

5. **Lokasi:** `app/repository/student_repository.go` (Cursor Pagination)
   - **Jenis:** Behavioral Error
   - **Penyebab:** Pengurutan cursor pagination menggunakan operator `>` yang ditujukan untuk data ASC, padahal query cursor meminta pengurutan DESC (dari terbaru ke terlama). Akibatnya pagination tidak bisa melangkah ke halaman berikutnya.
   - **Perbaikan:** Mengubah operator menjadi `(created_at, id) < ($1, $2)` agar sejalan dengan arah `ORDER BY created_at DESC, id DESC`.
   - **Evidence:** Perbaikan terdapat pada fungsi `FindAllCursor`.

6. **Lokasi:** `app/repository/student_repository.go` (Cursor Pagination)
   - **Jenis:** Behavioral Error
   - **Penyebab:** Selalu mengembalikan `has_more = false` karena tidak mengambil data tambahan (`LIMIT + 1`).
   - **Perbaikan:** Menggunakan teknik `LIMIT + 1` saat query, dan jika baris yang dikembalikan lebih dari Limit asli, berarti `has_more = true`.
   - **Evidence:** Implementasi query LIMIT+1 dan pemotongan slice di `FindAllCursor`.

7. **Lokasi:** `app/service/student_service.go` (Method Patch)
   - **Jenis:** Behavioral Error
   - **Penyebab:** Pengecekan nama kosong `*req.Name == ""` langsung dieksekusi tanpa memeriksa apakah pointer `req.Name != nil`, menyebabkan panic (nil pointer dereference) jika client tidak mengirim nama.
   - **Perbaikan:** Menambahkan pengaman pointer: `if req.Name != nil && *req.Name == ""`.
   - **Evidence:** Telah ditambahkan validasi pointer sebelum dereference di method Patch.

8. **Lokasi:** `app/service/student_service.go` (Content Negotiation)
   - **Jenis:** Behavioral Error
   - **Penyebab:** Pengaturan tipe konten CSV dikerjakan tanpa mengecek header `Accept`, atau secara default langsung me-return CSV walau client meminta JSON.
   - **Perbaikan:** Mengimplementasikan struktur `switch` dengan memeriksa nilai dari `c.Get("Accept")`.
   - **Evidence:** Logika content negotiation diterapkan pada method List di `student_service.go`.

9. **Lokasi:** `middleware/middleware.go` (RequestLogger)
   - **Jenis:** Behavioral Error
   - **Penyebab:** Logger memanggil `c.Response().StatusCode()` sebelum middleware lain atau error handler memproses error (AppError), sehingga status yang tercatat selalu 200 walau ada error 400/500 dari controller.
   - **Perbaikan:** Menambahkan logika inferensi status dari object `err` yang direturn oleh `c.Next()` jika ada (seperti `*AppError`), sehingga logger mencatat kode yang sebenarnya akan dikembalikan ke client.
   - **Evidence:** Perubahan logika `status := ...` pada middleware `RequestLogger`.
