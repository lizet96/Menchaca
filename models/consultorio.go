package models

import (
	"time"
)

// Consultorio representa la tabla Consultorio en la base de datos
type Consultorio struct {
	IDConsultorio int    `json:"id_consultorio" db:"id_consultorio"`
	Ubicacion     string `json:"ubicacion" db:"ubicacion" validate:"required,max=100"`
	NombreNumero  string `json:"nombre_numero" db:"nombre_numero" validate:"required,max=50"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}