package models

import (
	"time"
)

// Consulta representa la tabla Consulta en la base de datos
type Consulta struct {
	ID          int       `json:"id_consulta" db:"id_consulta"`
	Tipo        string    `json:"tipo" db:"tipo"`
	Diagnostico string    `json:"diagnostico" db:"diagnostico"`
	Costo       float64   `json:"costo" db:"costo"`
	IDPaciente  int       `json:"id_paciente" db:"id_paciente"`
	IDMedico    int       `json:"id_medico" db:"id_medico"`
	IDHorario   *int      `json:"id_horario" db:"id_horario"` // Changed from int to *int to handle NULL
	Hora        time.Time `json:"hora" db:"hora"`
}

// CitaRequest representa una solicitud para crear una cita
type CitaRequest struct {
	IDPaciente    int       `json:"id_paciente" validate:"required"`
	IDMedico      int       `json:"id_medico" validate:"required"`
	IDConsultorio int       `json:"id_consultorio" validate:"required"`
	FechaHora     time.Time `json:"fecha_hora" validate:"required"`
	Tipo          string    `json:"tipo" validate:"max=50"`
}
