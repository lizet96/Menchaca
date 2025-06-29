package models

import (
	"time"
)

// Horario representa la tabla Horario en la base de datos
type Horario struct {
	IDHorario          int  `json:"id_horario" db:"id_horario"`
	Turno              string `json:"turno" db:"turno" validate:"required,max=50"`
	IDMedico           int  `json:"id_medico" db:"id_medico"`
	IDConsultorio      int  `json:"id_consultorio" db:"id_consultorio"`
	ConsultaDisponible bool `json:"consulta_disponible" db:"consulta_disponible"`
	FechaHora          time.Time `json:"fecha_hora" db:"fecha_hora"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}