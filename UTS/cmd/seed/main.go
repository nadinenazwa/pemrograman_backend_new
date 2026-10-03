package main

import (
	"context"
	"fmt"
	"log"

	"siakad/config"
	"siakad/internal/models"
	"siakad/internal/repositories"
	"siakad/internal/utils"
)

func main() {
	config.LoadEnv()
	ctx := context.Background()
	pool, err := config.ConnectDB(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	studentRepo := repositories.NewStudentRepository(pool)

	// Seed Admin
	adminPass, _ := utils.HashPassword("admin123")
	_, err = pool.Exec(ctx, `INSERT INTO users (email, password, role) VALUES ('admin@siakad.test', $1, 'admin') ON CONFLICT (email) DO NOTHING`, adminPass)
	if err != nil {
		log.Fatalf("Failed to seed admin: %v", err)
	}

	// Seed Courses (10)
	courses := []models.Course{
		{KodeMK: "CS101", NamaMK: "Pemrograman Dasar", SKS: 3, Semester: "Ganjil", Kuota: 10},
		{KodeMK: "CS102", NamaMK: "Struktur Data", SKS: 4, Semester: "Genap", Kuota: 15},
		{KodeMK: "CS103", NamaMK: "Basis Data", SKS: 3, Semester: "Ganjil", Kuota: 5},
		{KodeMK: "CS104", NamaMK: "Jaringan Komputer", SKS: 3, Semester: "Genap", Kuota: 20},
		{KodeMK: "CS105", NamaMK: "Kecerdasan Buatan", SKS: 4, Semester: "Ganjil", Kuota: 12},
		{KodeMK: "CS106", NamaMK: "Sistem Operasi", SKS: 3, Semester: "Genap", Kuota: 18},
		{KodeMK: "CS107", NamaMK: "Pemrograman Web", SKS: 3, Semester: "Ganjil", Kuota: 2}, // Low quota for testing full
		{KodeMK: "CS108", NamaMK: "Rekayasa Perangkat Lunak", SKS: 4, Semester: "Genap", Kuota: 15},
		{KodeMK: "CS109", NamaMK: "Keamanan Sistem", SKS: 3, Semester: "Ganjil", Kuota: 20},
		{KodeMK: "CS110", NamaMK: "Pemrograman Mobile", SKS: 3, Semester: "Genap", Kuota: 10},
	}
	for _, c := range courses {
		_, err = pool.Exec(ctx, `INSERT INTO courses (kode_mk, nama_mk, sks, semester, kuota) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (kode_mk) DO NOTHING`,
			c.KodeMK, c.NamaMK, c.SKS, c.Semester, c.Kuota)
		if err != nil {
			log.Fatalf("Failed to seed course %s: %v", c.KodeMK, err)
		}
	}

	// Seed Students (20)
	for i := 1; i <= 20; i++ {
		nim := fmt.Sprintf("1234567890%02d", i)
		email := fmt.Sprintf("student%d@siakad.test", i)
		pass, _ := utils.HashPassword(nim)

		s := &models.Student{
			NIM:         nim,
			Nama:        fmt.Sprintf("Mahasiswa %d", i),
			Prodi:       "Teknik Informatika",
			Angkatan:    "2024",
			IPKTerakhir: 3.50, // default good ipk
		}
		if i%3 == 0 {
			s.IPKTerakhir = 2.80 // 2.5 - 2.99
		}
		if i%5 == 0 {
			s.IPKTerakhir = 2.00 // < 2.5
		}
		if i%2 == 0 {
			s.Prodi = "Sistem Informasi"
		}

		err = studentRepo.CreateWithUser(ctx, s, email, pass)
		if err != nil {
			// ignore duplicate error for simple seed running multiple times
			if err.Error() == "ERROR: duplicate key value violates unique constraint \"users_email_key\" (SQLSTATE 23505)" {
				continue
			}
			fmt.Printf("Warning: failed to seed student %s: %v\n", nim, err)
		}
	}

	fmt.Println("Seeding completed successfully.")
}
