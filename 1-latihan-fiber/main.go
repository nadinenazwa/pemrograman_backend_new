package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
)

// --- SOAL 4: STRUCT STUDENT ---
type Student struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Grade    float64 `json:"grade"`
	IsActive bool    `json:"is_active"`
}

func (s Student) GetInfo() string {
	var status string
	if s.IsActive {
		status = "Aktif"
	} else {
		status = "Tidak Aktif"
	}
	return fmt.Sprintf("ID: %d | Nama: %-22s | Nilai: %.2f | Status: %s", s.ID, s.Name, s.Grade, status)
}

func (s *Student) UpdateGrade(newGrade float64) {
	s.Grade = newGrade
}

func (s *Student) Activate() {
	s.IsActive = true
}

func (s *Student) Deactivate() {
	s.IsActive = false
}

// --- SOAL 3: POINTER ---
func swap(a, b *int) {
	*a, *b = *b, *a
}

func updateSlice(s *[]string, newItem string) {
	*s = append(*s, newItem)
}

// --- MAIN FUNCTION ---
func main() {
	// 1. VARIABEL & MAP
	fmt.Println("=== SOAL 2: VARIABEL & MAP ===")
	var nama string = "Nadine Nazwa Andina"
	var umur int = 20
	var ipk float64 = 3.85
	var isAktif bool = true
	var matkul []string = []string{"Pemrograman Backend", "Basis Data"}

	fmt.Printf("Nama   : %s\nUmur   : %d\nIPK    : %.2f\nAktif  : %v\nMatkul : %v\n\n", nama, umur, ipk, isAktif, matkul)

	nilaiMahasiswa := make(map[string]float64)
	nilaiMahasiswa["Nadine Nazwa Andina"] = 88.5
	nilaiMahasiswa["Adam Ahmad Bimantoro"] = 92.0
	nilaiMahasiswa["Lusiana Ramadhan"] = 78.0

	nilaiMahasiswa["Adam Ahmad Bimantoro"] = 95.0
	delete(nilaiMahasiswa, "Lusiana Ramadhan")

	fmt.Println("Daftar Mahasiswa di Map:")
	for namaMhs, nilai := range nilaiMahasiswa {
		fmt.Printf("- %s : %.2f\n", namaMhs, nilai)
	}
	fmt.Println()

	// 2. POINTER
	fmt.Println("=== SOAL 3: POINTER ===")
	x, y := 10, 50
	fmt.Printf("Sebelum Swap : x = %d, y = %d\n", x, y)
	swap(&x, &y)
	fmt.Printf("Setelah Swap  : x = %d, y = %d\n\n", x, y)

	daftarMahasiswa := []string{"Nadine Nazwa Andina", "Adam Ahmad Bimantoro"}
	updateSlice(&daftarMahasiswa, "Lusiana Ramadhan")
	fmt.Println("Slice Hasil Update :", daftarMahasiswa)
	fmt.Println()

	// 3. STRUCT STUDENT
	fmt.Println("=== SOAL 4: STRUCT STUDENT ===")
	mhs1 := Student{ID: 101, Name: "Nadine Nazwa Andina", Grade: 88.5, IsActive: true}
	mhs2 := Student{ID: 102, Name: "Adam Ahmad Bimantoro", Grade: 80.0, IsActive: false}
	mhs3 := Student{ID: 103, Name: "Lusiana Ramadhan", Grade: 75.5, IsActive: true}

	mhs1.UpdateGrade(95.0)
	mhs2.Activate()
	mhs2.UpdateGrade(90.0)
	mhs3.Deactivate()

	fmt.Println(mhs1.GetInfo())
	fmt.Println(mhs2.GetInfo())
	fmt.Println(mhs3.GetInfo())
	fmt.Println("\n==================================")
	fmt.Println("Server API berjalan di port 3000...")
	fmt.Println("==================================")

	// 4. FIBER REST API
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World! Ini tugas Modul 1 Nadine.")
	})

	app.Get("/api/student", func(c *fiber.Ctx) error {
		mhs := Student{ID: 101, Name: "Nadine Nazwa Andina", Grade: 88.5, IsActive: false}
		mhs.Activate() 
		return c.JSON(mhs)
	})

	log.Fatal(app.Listen(":3000"))
}