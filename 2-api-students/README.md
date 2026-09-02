# Kontrak REST API - Students Management (api-students)

| Metode | Endpoint | Parameter / Query String | Contoh Body Permintaan | Status Response | Contoh Body Response / Header |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **GET** | `/api/v1/students` | `page`, `limit`, `search`, `sort`, `order`, `is_active` | *(Tanpa Body)* | **200 OK** | `{"success": true, "data": [...], "meta": {...}}` |
| **GET** | `/api/v1/students/:id` | `:id` (path param, int) | *(Tanpa Body)* | **200 OK**<br>**400 Bad Request**<br>**404 Not Found** | `{"success": true, "data": {"id": 1, "nim": "230101001", ...}}` |
| **POST** | `/api/v1/students` | Header: `Content-Type: application/json` | `{"nim": "230101004", "name": "Budi", "grade": 85.0, "is_active": true}` | **201 Created**<br>**409 Conflict**<br>**415 Unsupported**<br>**422 Unprocessable** | Header: `Location: /api/v1/students/4`<br>Body: `{"success": true, "data": {...}}` |
| **PUT** | `/api/v1/students/:id` | `:id` (path param, int) | `{"nim": "230101001", "name": "Nadine Nazwa Andina", "grade": 90.0, "is_active": true}` | **200 OK**<br>**400 Bad Request**<br>**404 Not Found**<br>**422 Unprocessable** | `{"success": true, "message": "Data mahasiswa berhasil diganti...", "data": {...}}` |
| **PATCH** | `/api/v1/students/:id` | `:id` (path param, int) | `{"grade": 95.0}` | **200 OK**<br>**400 Bad Request**<br>**404 Not Found**<br>**422 Unprocessable** | `{"success": true, "message": "Sebagian data mahasiswa berhasil diperbarui...", "data": {...}}` |
| **DELETE** | `/api/v1/students/:id` | `:id` (path param, int) | *(Tanpa Body)* | **204 No Content**<br>**400 Bad Request**<br>**404 Not Found** | *(Tanpa Body / Kosong)* |