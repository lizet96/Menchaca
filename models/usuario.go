package models

import (
	"time"
)

// Usuario representa la tabla Usuario en la base de datos
type Usuario struct {
	IDUsuario       int       `json:"id_usuario" db:"id_usuario"`
	Nombre          string    `json:"nombre" db:"nombre"`
	Apellido        string    `json:"apellido" db:"apellido"`
	Email           string    `json:"email" db:"email"`
	Password        string    `json:"password,omitempty" db:"password"`
	FechaNacimiento string    `json:"fecha_nacimiento" db:"fecha_nacimiento"`
	IDRol           int       `json:"id_rol" db:"id_rol"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
	MFAEnabled      bool      `json:"mfa_enabled" db:"mfa_enabled"`
	MFASecret       string    `json:"-" db:"mfa_secret"`
	BackupCodes     string    `json:"-" db:"backup_codes"`
}

// UsuarioResponse representa la respuesta sin datos sensibles
type UsuarioResponse struct {
	ID              int       `json:"id_usuario"`
	Nombre          string    `json:"nombre"`
	Apellido        string    `json:"apellido"`
	FechaNacimiento string    `json:"fecha_nacimiento"`
	Tipo            string    `json:"tipo"`             // Mantener por compatibilidad
	IDRol           *int      `json:"id_rol,omitempty"` // Nuevo campo
	Email           string    `json:"email"`
	CreatedAt       time.Time `json:"created_at"`
}

// LoginRequest representa la solicitud de login
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RefreshToken representa un token de actualización
type RefreshToken struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	IsRevoked bool      `json:"is_revoked" db:"is_revoked"`
}

// LoginResponse representa la respuesta del login con tokens
type LoginResponse struct {
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token"`
	ExpiresIn    int             `json:"expires_in"` // segundos
	Usuario      UsuarioResponse `json:"usuario"`
}

// RefreshRequest para solicitar nuevo token
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RefreshResponse para respuesta de renovación
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// Nuevos tipos para MFA
type MFASetupRequest struct {
	Password string `json:"password" validate:"required"`
}

type MFASetupResponse struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

type MFAVerifyRequest struct {
	Code string `json:"code" validate:"required,len=6"`
}

type LoginMFARequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	MFACode  string `json:"mfa_code,omitempty"` // Opcional en el primer paso
}

// LoginMFAResponse representa la respuesta del login con MFA obligatorio
type LoginMFAResponse struct {
	RequiresMFA  bool            `json:"requires_mfa"`
	QRCodeURL    string          `json:"qr_code_url,omitempty"`  // Para usuarios sin MFA
	Secret       string          `json:"secret,omitempty"`       // Para usuarios sin MFA
	BackupCodes  []string        `json:"backup_codes,omitempty"` // Para usuarios sin MFA
	AccessToken  string          `json:"access_token,omitempty"`
	RefreshToken string          `json:"refresh_token,omitempty"`
	ExpiresIn    int             `json:"expires_in,omitempty"`
	Usuario      UsuarioResponse `json:"usuario,omitempty"`
}
