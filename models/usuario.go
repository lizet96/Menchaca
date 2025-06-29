package models

import (
	"time"
)

// Usuario representa la tabla Usuario en la base de datos
type Usuario struct {
	ID        int       `json:"id_usuario" db:"id_usuario"`
	Nombre    string    `json:"nombre" db:"nombre" validate:"required,min=2,max=100"`
	Tipo      string    `json:"tipo" db:"tipo" validate:"required,oneof=paciente medico enfermera admin"`
	Email     string    `json:"email" db:"email" validate:"required,email"`
	Password  string    `json:"password,omitempty" db:"password" validate:"required,min=6"`
	Apellido  string    `json:"apellido" db:"apellido" validate:"required,min=2,max=100"`
    FechaNacimiento time.Time `json:"fecha_nacimiento" db:"fecha_nacimiento" validate:"required"`	
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// UsuarioResponse representa la respuesta sin datos sensibles
type UsuarioResponse struct {
	ID              int       `json:"id_usuario"`
	Nombre          string    `json:"nombre"`
	Apellido        string    `json:"apellido"`
	FechaNacimiento time.Time `json:"fecha_nacimiento"`
	Tipo            string    `json:"tipo"`
	Email           string    `json:"email"`
	CreatedAt       time.Time `json:"created_at"`
}

// LoginRequest representa la solicitud de login
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse representa la respuesta del login
type LoginResponse struct {
	Token   string          `json:"token"`
	Usuario UsuarioResponse `json:"usuario"`
}
