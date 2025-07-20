package models

import (
	"time"
)

type Log struct {
	IDLog        int       `json:"id_log" db:"id_log"`
	Method       string    `json:"method" db:"method"`
	Path         string    `json:"path" db:"path"`
	Protocol     string    `json:"protocol" db:"protocol"`
	StatusCode   int       `json:"status_code" db:"status_code"`
	ResponseTime *int      `json:"response_time" db:"response_time"`
	UserAgent    *string   `json:"user_agent" db:"user_agent"`
	IP           string    `json:"ip" db:"ip"`
	Hostname     *string   `json:"hostname" db:"hostname"`
	Body         *string   `json:"body" db:"body"`
	Params       *string   `json:"params" db:"params"`
	Query        *string   `json:"query" db:"query"`
	Email        *string   `json:"email" db:"email"`
	Username     *string   `json:"username" db:"username"`
	Role         *string   `json:"role" db:"role"`
	LogLevel     string    `json:"log_level" db:"log_level"`
	Environment  string    `json:"environment" db:"environment"`
	NodeVersion  *string   `json:"node_version" db:"node_version"`
	PID          *int      `json:"pid" db:"pid"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
	URL          *string   `json:"url" db:"url"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type CreateLogRequest struct {
	Method       string  `json:"method" validate:"required,max=10"`
	Path         string  `json:"path" validate:"required,max=500"`
	Protocol     *string `json:"protocol,omitempty"`
	StatusCode   int     `json:"status_code" validate:"required"`
	ResponseTime *int    `json:"response_time,omitempty"`
	UserAgent    *string `json:"user_agent,omitempty"`
	IP           string  `json:"ip" validate:"required,max=45"`
	Hostname     *string `json:"hostname,omitempty"`
	Body         *string `json:"body,omitempty"`
	Params       *string `json:"params,omitempty"`
	Query        *string `json:"query,omitempty"`
	Email        *string `json:"email,omitempty"`
	Username     *string `json:"username,omitempty"`
	Role         *string `json:"role,omitempty"`
	LogLevel     *string `json:"log_level,omitempty"`
	Environment  *string `json:"environment,omitempty"`
	NodeVersion  *string `json:"node_version,omitempty"`
	PID          *int    `json:"pid,omitempty"`
	URL          *string `json:"url,omitempty"`
}

// Constantes para niveles de log
const (
	LogLevelInfo    = "info"
	LogLevelWarning = "warning"
	LogLevelError   = "error"
	LogLevelDebug   = "debug"
	LogLevelSuccess = "success"
)

// Constantes para ambientes
const (
	EnvironmentDevelopment = "development"
	EnvironmentProduction  = "production"
	EnvironmentTesting     = "testing"
)
