package models

// Expediente representa la tabla Expediente en la base de datos
type Expediente struct {
	ID               int    `json:"id_expediente" db:"id_expediente"`
	Antecedentes     string `json:"antecedentes" db:"antecedentes"`
	HistorialClinico string `json:"historial_clinico" db:"historial_clinico"`
	Seguro           string `json:"seguro" db:"seguro"`
	IDPaciente       int    `json:"id_paciente" db:"id_paciente"`
}
