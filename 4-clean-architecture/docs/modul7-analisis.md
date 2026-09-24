# Analisis Modul 7 - Advanced API Design

## Penjelasan Implementasi Entitas Students

Berikut adalah penjelasan mengenai implementasi fitur-fitur baru pada endpoint students sesuai requirement Modul 7:

1. **Declarative Validation**
   Validasi kini menggunakan tag struct (contoh: `validate:"required,min=2"`). Hal ini memusatkan aturan validasi di layer model (struct definisi request) sehingga service tidak lagi berisi puluhan baris if-else manual. Validasi ini mengurangi duplikasi kode, mempermudah pengujian, dan menjamin validasi konsisten di seluruh aplikasi.

2. **Custom Validation**
   Karena *go-playground/validator* bawaan tidak memiliki format khusus domain, kita menambahkan custom validator seperti `validnim` (harus berisi angka dan berpanjang 7-20 karakter) dan `strongpassword`. Validator custom ini diregistrasikan ke singleton validator di saat aplikasi startup sehingga dapat digunakan layaknya tag bawaan (`validate:"validnim"`).

3. **PATCH dengan Pointer + Omitnil**
   Karena JSON bisa berisi data kosong (seperti `""` atau `0`) yang berbeda dengan "tidak dikirim sama sekali" (null/absent), field pada struct PATCH diubah menjadi pointer (contoh: `*string`). JSON omitempty akan menghilangkan field jika nil, namun dengan library tertentu atau tag custom `omitnil` kita secara spesifik membedakan null dari *empty state*. Fungsi tambahan seperti `IsEmptyPatch()` diperlukan untuk mengecek secara inter-field apakah ada minimal 1 data yang dikirim (tidak bisa di-handle per-field oleh tag validator).

4. **Cursor Pagination**
   Beralih dari *offset-based* ke *keyset-based* menggunakan kombinasi `(created_at, id)`. Pendekatan ini stabil saat insert berbarengan dan menghapus masalah O(N) penalty pada offset pagination, di mana database tetap perlu me-scan ribuan row hanya untuk melewatinya. Implementasinya juga menggunakan teknik `LIMIT + 1` untuk secara deterministik mendeteksi `has_more` tanpa perlu count query kedua. 

5. **Database Index**
   Untuk menunjang Cursor Pagination yang mengurutkan `created_at DESC, id DESC`, dibuat composite index: `CREATE INDEX idx_students_cursor_pagination ON students (created_at DESC, id DESC)`. Ini memampukan query planer melakukan backward index scan murni tanpa perlu sorting di memory, yang membuat waktu respons tetap O(1) konstan (hanya scan N baris).

6. **EXPLAIN ANALYZE**
   Perintah EXPLAIN ANALYZE menunjukkan bahwa eksekusi pencarian memanfaatkan index. Ketika data tumbuh, execution time dengan cursor + index tetap < 1ms untuk mencari batch berikutnya dibanding offset yang melambat secara logaritmik. Bukti hasil analisis ini dilampirkan pada `docs/modul7-evidence/explain_analyze.txt`.

7. **JSON / CSV Content Negotiation**
   Endpoint `GET /students` telah disesuaikan agar tidak terpaku mem-filter content type via middleware `RequireJSON`. Melalui content negotiation (mengecek header `Accept`), endpoint kini sanggup me-return dua variasi format: JSON (dengan pagination lengkap) maupun *text/csv* (dengan CSV Writer native Go) dalam satu handler logic.

8. **Error Handling (Centralized)**
   Sistem berubah dari cara manual (me-return `helper.Fail(c, 4xx, ...)`) menjadi format struct error terpusat (`AppError`). Handler cukup me-return nilai `err`, yang akan ditangkap oleh `ErrorHandler` di level teratas Fiber. Pola *Bubble up* ini mencegah kebocoran implementasi HTTP status per layer, menyederhanakan kode, dan memudahkan middleware (seperti `RequestLogger`) mencatat status aktual dengan lebih tepat dengan membaca inferensi `*AppError` dari object error.
