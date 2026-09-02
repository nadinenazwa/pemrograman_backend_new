package main

import "fmt"

func swap(a, b *int) {
	*a, *b = *b, *a
}

func updateSlice(s *[]string, newItem string) {
	*s = append(*s, newItem)
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
}