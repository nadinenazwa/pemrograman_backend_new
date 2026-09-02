package main

import "fmt"

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
}