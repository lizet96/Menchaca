package models

import (
	"time"
)

// Receta representa la tabla Receta en la base de datos
type Receta struct {
	IDReceta            int       `json:"id_receta" db:"id_receta"`
	Fecha         time.Time `json:"fecha" db:"fecha"`
	Medicamento   string    `json:"medicamento" db:"medicamento" validate:"required,max=255"`
	Dosis         string    `json:"dosis" db:"dosis" validate:"required,max=100"`
	Instrucciones string    `json:"instrucciones" db:"instrucciones"`
	IDMedico      int       `json:"id_medico" db:"id_medico"`
	IDPaciente    int       `json:"id_paciente" db:"id_paciente"`
	IDConsultorio int       `json:"id_consultorio" db:"id_consultorio"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}