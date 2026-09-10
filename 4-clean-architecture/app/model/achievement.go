package model

type Achievement struct {
	IDAchievement int    `json:"id_achievement"`
	NIMStudents   string `json:"nim_students"`
	NamaPrestasi  string `json:"nama_prestasi"`
	Juara         string `json:"juara"`
}