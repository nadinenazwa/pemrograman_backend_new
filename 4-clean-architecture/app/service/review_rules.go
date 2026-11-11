package service

import (
	"slices"
	"strings"
)

// NormalizeTags menormalkan tag ulasan: huruf kecil, trim spasi, hapus string kosong dan duplikat.
// Menjamin hasil berupa slice kosong ([]string{}) dan bukan nil agar sesuai format JSON.
func NormalizeTags(in []string) []string {
	if in == nil {
		return []string{}
	}
	result := make([]string, 0, len(in))
	for _, raw := range in {
		cleaned := strings.ToLower(strings.TrimSpace(raw))
		if cleaned == "" {
			continue
		}
		if !slices.Contains(result, cleaned) {
			result = append(result, cleaned)
		}
	}
	return result
}
