package models

import (
	"time"
)

// Consulta representa la tabla Consulta en la base de datos
type Consulta struct {
	ID             int       `json:"id_consulta" db:"id_consulta"`
	Tipo           string    `json:"tipo" db:"tipo" validate:"max=100"`
	Diagnostico    string    `json:"diagnostico" db:"diagnostico"`
	Costo          float64   `json:"costo" db:"costo"`
	IDPaciente     int       `json:"id_paciente" db:"id_paciente"`
	IDMedico       int       `json:"id_medico" db:"id_medico"`
	IDHorario      int       `json:"id_horario" db:"id_horario"`
	Fecha          time.Time `json:"fecha" db:"fecha"`
	FechaConsulta  time.Time `json:"fecha_consulta" db:"fecha_consulta"`
	Motivo         string    `json:"motivo" db:"motivo" validate:"max=500"`
	Tratamiento    string    `json:"tratamiento" db:"tratamiento"`
	Observaciones  string    `json:"observaciones" db:"observaciones"`
	Estado         string    `json:"estado" db:"estado" validate:"oneof=programada en_curso completada cancelada"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// CitaRequest representa una solicitud para crear una cita
type CitaRequest struct {
	IDPaciente    int       `json:"id_paciente" validate:"required"`
	IDMedico      int       `json:"id_medico" validate:"required"`
	IDConsultorio int       `json:"id_consultorio" validate:"required"`
	FechaHora     time.Time `json:"fecha_hora" validate:"required"`
	Tipo          string    `json:"tipo" validate:"required,max=50"`
	Motivo        string    `json:"motivo" validate:"max=500"`
}