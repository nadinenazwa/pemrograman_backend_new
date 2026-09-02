package main

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// GET /api/v1/students (Daftar Mahasiswa + Paginasi, Search, Sort, Filter)
func GetStudents(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 { page = 1 }

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 { limit = 10 } else if limit > 50 { limit = 50 } // Batas aman max 50

	search := c.Query("search", "")
	sortBy := c.Query("sort", "id")
	orderBy := c.Query("order", "asc")
	isActiveFilter := c.Query("is_active", "")

	var result []Student

	// Filter Search & IsActive
	for _, s := range studentList {
		if search != "" && !strings.Contains(strings.ToLower(s.Name), strings.ToLower(search)) {
			continue
		}
		if isActiveFilter != "" {
			activeBool, err := strconv.ParseBool(isActiveFilter)
			if err == nil && s.IsActive != activeBool {
				continue
			}
		}
		result = append(result, s)
	}

	// Whitelist sorting
	allowedSortFields := map[string]bool{"id": true, "nim": true, "name": true, "grade": true}
	if !allowedSortFields[sortBy] { sortBy = "id" }

	sort.Slice(result, func(i, j int) bool {
		if orderBy == "desc" {
			switch sortBy {
			case "name": return result[i].Name > result[j].Name
			case "grade": return result[i].Grade > result[j].Grade
			case "nim": return result[i].NIM > result[j].NIM
			default: return result[i].ID > result[j].ID
			}
		}
		switch sortBy {
		case "name": return result[i].Name < result[j].Name
		case "grade": return result[i].Grade < result[j].Grade
		case "nim": return result[i].NIM < result[j].NIM
		default: return result[i].ID < result[j].ID
		}
	})

	total := len(result)
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	startIndex := (page - 1) * limit
	endIndex := startIndex + limit

	var pagedData []Student
	if startIndex < total {
		if endIndex > total { endIndex = total }
		pagedData = result[startIndex:endIndex]
	} else {
		pagedData = []Student{}
	}

	return c.Status(fiber.StatusOK).JSON(APIResponse{
		Success: true,
		Message: "Berhasil mengambil daftar mahasiswa",
		Data:    pagedData,
		Meta: &Meta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// GET /api/v1/students/:id
func GetStudentByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false,
			Message: "ID harus berupa angka integer",
		})
	}

	for _, s := range studentList {
		if s.ID == id {
			return c.Status(fiber.StatusOK).JSON(APIResponse{
				Success: true,
				Message: "Data mahasiswa ditemukan",
				Data:    s,
			})
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(APIResponse{
		Success: false,
		Message: "Data mahasiswa tidak ditemukan",
	})
}

// POST /api/v1/students
func CreateStudent(c *fiber.Ctx) error {
	// Cek HTTP 415
	if err := checkContentTypeJSON(c); err != nil { return err }

	var req CreateStudentRequest
	// Cek HTTP 400
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false,
			Message: "Format JSON request body tidak valid",
		})
	}

	// Cek HTTP 422 (Unprocessable Entity)
	errorsMap := make(map[string]string)
	if strings.TrimSpace(req.NIM) == "" { errorsMap["nim"] = "NIM wajib diisi" }
	if strings.TrimSpace(req.Name) == "" { errorsMap["name"] = "Nama wajib diisi" }
	if req.Grade == nil {
		errorsMap["grade"] = "Grade wajib diisi"
	} else if *req.Grade < 0 || *req.Grade > 100 {
		errorsMap["grade"] = "Grade harus bernilai antara 0 sampai 100"
	}
	if req.IsActive == nil { errorsMap["is_active"] = "IsActive wajib diisi" }

	if len(errorsMap) > 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(APIResponse{
			Success: false,
			Message: "Validasi isi permintaan gagal",
			Errors:  errorsMap,
		})
	}

	// Cek HTTP 409 (Conflict - NIM Duplikat)
	if isNIMExists(req.NIM, 0) {
		return c.Status(fiber.StatusConflict).JSON(APIResponse{
			Success: false,
			Message: "NIM sudah terdaftar dalam sistem",
		})
	}

	newStudent := Student{
		ID:       nextID,
		NIM:      req.NIM,
		Name:     req.Name,
		Grade:    *req.Grade,
		IsActive: *req.IsActive,
	}
	nextID++
	studentList = append(studentList, newStudent)

	// Set Header Location sesuai standar HTTP 201 Created
	c.Set("Location", "/api/v1/students/"+strconv.Itoa(newStudent.ID))

	return c.Status(fiber.StatusCreated).JSON(APIResponse{
		Success: true,
		Message: "Mahasiswa berhasil ditambahkan",
		Data:    newStudent,
	})
}

// PUT /api/v1/students/:id (Replace Seluruh Field)
func ReplaceStudent(c *fiber.Ctx) error {
	if err := checkContentTypeJSON(c); err != nil { return err }

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false, Message: "ID harus berupa angka integer",
		})
	}

	var req CreateStudentRequest // PUT wajib mengirimkan semua field
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false, Message: "Format JSON tidak valid",
		})
	}

	// Validasi kelengkapan seluruh field (Wajib dikirim ulang semua)
	errorsMap := make(map[string]string)
	if strings.TrimSpace(req.NIM) == "" { errorsMap["nim"] = "NIM wajib diisi" }
	if strings.TrimSpace(req.Name) == "" { errorsMap["name"] = "Nama wajib diisi" }
	if req.Grade == nil { errorsMap["grade"] = "Grade wajib diisi" }
	if req.IsActive == nil { errorsMap["is_active"] = "IsActive wajib diisi" }

	if len(errorsMap) > 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(APIResponse{
			Success: false, Message: "PUT membutuhkan seluruh field dikirim ulang lengkap", Errors: errorsMap,
		})
	}

	if isNIMExists(req.NIM, id) {
		return c.Status(fiber.StatusConflict).JSON(APIResponse{
			Success: false, Message: "NIM sudah terdaftar pada mahasiswa lain",
		})
	}

	for i, s := range studentList {
		if s.ID == id {
			studentList[i].NIM = req.NIM
			studentList[i].Name = req.Name
			studentList[i].Grade = *req.Grade
			studentList[i].IsActive = *req.IsActive

			return c.Status(fiber.StatusOK).JSON(APIResponse{
				Success: true, Message: "Data mahasiswa berhasil diganti secara keseluruhan (PUT)", Data: studentList[i],
			})
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(APIResponse{
		Success: false, Message: "Data mahasiswa tidak ditemukan",
	})
}

// PATCH /api/v1/students/:id (Update Sebagian Field)
func UpdateStudentPartial(c *fiber.Ctx) error {
	if err := checkContentTypeJSON(c); err != nil { return err }

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false, Message: "ID harus berupa angka integer",
		})
	}

	var req UpdateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false, Message: "Format JSON tidak valid",
		})
	}

	if req.Grade != nil && (*req.Grade < 0 || *req.Grade > 100) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(APIResponse{
			Success: false, Message: "Validasi gagal", Errors: map[string]string{"grade": "Grade harus 0-100"},
		})
	}

	if req.NIM != nil && isNIMExists(*req.NIM, id) {
		return c.Status(fiber.StatusConflict).JSON(APIResponse{
			Success: false, Message: "NIM sudah terdaftar pada mahasiswa lain",
		})
	}

	for i, s := range studentList {
		if s.ID == id {
			if req.NIM != nil { studentList[i].NIM = *req.NIM }
			if req.Name != nil { studentList[i].Name = *req.Name }
			if req.Grade != nil { studentList[i].Grade = *req.Grade }
			if req.IsActive != nil { studentList[i].IsActive = *req.IsActive }

			return c.Status(fiber.StatusOK).JSON(APIResponse{
				Success: true, Message: "Sebagian data mahasiswa berhasil diperbarui (PATCH)", Data: studentList[i],
			})
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(APIResponse{
		Success: false, Message: "Data mahasiswa tidak ditemukan",
	})
}

// DELETE /api/v1/students/:id (204 No Content)
func DeleteStudent(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
			Success: false, Message: "ID harus berupa angka integer",
		})
	}

	for i, s := range studentList {
		if s.ID == id {
			studentList = append(studentList[:i], studentList[i+1:]...)
			// HTTP 204 Wajib Tanpa Body
			return c.SendStatus(fiber.StatusNoContent)
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(APIResponse{
		Success: false, Message: "Data mahasiswa tidak ditemukan",
	})
}