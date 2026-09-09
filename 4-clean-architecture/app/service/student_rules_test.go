package service

import (
	"testing"

	"api-students-db/app/model"
)

func TestCountTotalPages(t *testing.T) {
	cases := []struct{ total, limit, want int }{
		{0, 10, 0}, {1, 10, 1}, {10, 10, 1}, {11, 10, 2}, {137, 20, 7},
	}
	for _, tc := range cases {
		if got := CountTotalPages(tc.total, tc.limit); got != tc.want {
			t.Errorf("total=%d limit=%d: harap %d, dapat %d", tc.total, tc.limit, tc.want, got)
		}
	}
}

func TestValidateCreate_GradeOutOfRange(t *testing.T) {
	grade := 150.0
	active := true
	errs := ValidateCreate(model.CreateStudentRequest{
		NIM: "2024001", Name: "Sari", Grade: &grade, IsActive: &active,
	})
	if _, ok := errs["grade"]; !ok {
		t.Fatal("grade di luar 0-100 seharusnya menghasilkan error")
	}
}

func TestApplyPatch_OnlyChangesGivenFields(t *testing.T) {
	initial := model.Student{ID: 1, NIM: "2024001", Name: "Sari", Grade: 80, IsActive: true}
	inactive := false

	result := ApplyPatch(initial, model.UpdateStudentRequest{IsActive: &inactive})

	if result.IsActive {
		t.Error("is_active seharusnya berubah menjadi false")
	}
	if result.Name != "Sari" {
		t.Error("field yang tidak dikirim seharusnya tidak berubah")
	}
}