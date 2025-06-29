package models

import (
	"time"
)

// Expediente representa la tabla Expediente en la base de datos
type Expediente struct {
	ID                   int       `json:"id_expediente" db:"id_expediente"`
	IDPaciente          int       `json:"id_paciente" db:"id_paciente"`
	FechaCreacion       time.Time `json:"fecha_creacion" db:"fecha_creacion"`
	Antecedentes        string    `json:"antecedentes" db:"antecedentes"`
	HistorialClinico    string    `json:"historial_clinico" db:"historial_clinico"`
	Seguro              string    `json:"seguro" db:"seguro"`
	AntecedentesMedicos string    `json:"antecedentes_medicos" db:"antecedentes_medicos"`
	Alergias            string    `json:"alergias" db:"alergias"`
	MedicamentosActuales string   `json:"medicamentos_actuales" db:"medicamentos_actuales"`
	Observaciones       string    `json:"observaciones" db:"observaciones"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}