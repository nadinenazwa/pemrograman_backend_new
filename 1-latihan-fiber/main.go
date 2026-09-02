package main

import "fmt"

func swap(a, b *int) {
	*a, *b = *b, *a
}

func updateSlice(s *[]string, newItem string) {
	*s = append(*s, newItem)
}

type Student struct {
	ID       int
	Name     string
	Grade    float64
	IsActive bool
}

func (s *Student) Activate() {
	s.IsActive = true
}

func main() {
	fmt.Println("=== SOAL 2: VARIABEL & MAP ===")
	var nama string = "Nadine Nazwa Andina"
	var umur int = 20
	var ipk float64 = 3.85
	var isAktif bool = true
	var matkul []string = []string{"Pemrograman Backend", "Basis Data"}

	fmt.Printf("Nama: %s\nUmur: %d\nIPK: %.2f\nAktif: %v\nMatkul: %v\n\n", nama, umur, ipk, isAktif, matkul)

	nilaiMhs := make(map[string]float64)
	nilaiMhs["Nadine Nazwa Andina"] = 88.5
	nilaiMhs["Adam Ahmad Bimantoro"] = 92.0

	for nama, nilai := range nilaiMhs {
		fmt.Printf("- %s : %.2f\n", nama, nilai)
	}
	fmt.Println("\n=== SOAL 3: POINTER ===")
	x, y := 10, 50
	fmt.Printf("Sebelum Swap: x=%d, y=%d\n", x, y)
	swap(&x, &y)
	fmt.Printf("Setelah Swap: x=%d, y=%d\n", x, y)

	daftarMhs := []string{"Nadine", "Adam"}
	updateSlice(&daftarMhs, "Lusiana")
	fmt.Println("Slice Hasil Update:", daftarMhs)
	
	fmt.Println("\n=== SOAL 4: STRUCT STUDENT ===")
	mhs1 := Student{ID: 101, Name: "Nadine Nazwa", Grade: 88.5, IsActive: false}
	mhs1.Activate()
	fmt.Printf("ID: %d | Nama: %s | Nilai: %.2f | Aktif: %v\n", mhs1.ID, mhs1.Name, mhs1.Grade, mhs1.IsActive)
	}