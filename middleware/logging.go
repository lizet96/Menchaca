package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lizet96/hospital-backend/database"
	"github.com/lizet96/hospital-backend/models"
)

// LoggingMiddleware captura y registra todas las peticiones HTTP
func LoggingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Continuar con la petición
		err := c.Next()

		// Calcular tiempo de respuesta
		responseTime := int(time.Since(start).Milliseconds())

		// Crear log entry
		logEntry := createLogEntry(c, responseTime)

		// Guardar en base de datos de forma asíncrona
		go saveLogToDB(logEntry)

		return err
	}
}

// createLogEntry crea una entrada de log basada en la petición
func createLogEntry(c *fiber.Ctx, responseTime int) models.CreateLogRequest {
	// Obtener información del usuario si está autenticado
	var email, username, role *string
	if userEmail := c.Locals("user_email"); userEmail != nil {
		if emailStr, ok := userEmail.(string); ok {
			email = &emailStr
			username = &emailStr // Usar email como username por defecto
		}
	}
	if userRole := c.Locals("user_role"); userRole != nil {
		if roleStr, ok := userRole.(string); ok {
			role = &roleStr
		}
	}

	// Obtener IP real del cliente
	ip := c.IP()
	if forwarded := c.Get("X-Forwarded-For"); forwarded != "" {
		ip = strings.Split(forwarded, ",")[0]
	}
	if realIP := c.Get("X-Real-IP"); realIP != "" {
		ip = realIP
	}

	// Obtener User-Agent
	userAgent := c.Get("User-Agent")
	var userAgentPtr *string
	if userAgent != "" {
		userAgentPtr = &userAgent
	}

	// Obtener hostname
	hostname := c.Hostname()
	var hostnamePtr *string
	if hostname != "" {
		hostnamePtr = &hostname
	}

	// Obtener body (solo para métodos POST, PUT, PATCH)
	var bodyPtr *string
	if c.Method() == "POST" || c.Method() == "PUT" || c.Method() == "PATCH" {
		body := string(c.Body())
		if body != "" {
			// Filtrar información sensible
			body = filterSensitiveData(body)
			bodyPtr = &body
		}
	}

	// Obtener parámetros de ruta
	var paramsPtr *string
	if len(c.AllParams()) > 0 {
		paramsJSON, _ := json.Marshal(c.AllParams())
		paramsStr := string(paramsJSON)
		paramsPtr = &paramsStr
	}

	// Obtener query parameters
	var queryPtr *string
	if c.Request().URI().QueryString() != nil {
		queryStr := string(c.Request().URI().QueryString())
		if queryStr != "" {
			queryPtr = &queryStr
		}
	}

	// Obtener URL completa
	url := c.OriginalURL()
	var urlPtr *string
	if url != "" {
		urlPtr = &url
	}

	// Determinar nivel de log basado en status code
	logLevel := determineLogLevel(c.Response().StatusCode())

	// Obtener ambiente
	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		environment = models.EnvironmentDevelopment
	}

	// Obtener PID del proceso
	pid := os.Getpid()

	// Protocolo
	protocol := "HTTP/1.1"

	return models.CreateLogRequest{
		Method:       c.Method(),
		Path:         c.Path(),
		Protocol:     &protocol,
		StatusCode:   c.Response().StatusCode(),
		ResponseTime: &responseTime,
		UserAgent:    userAgentPtr,
		IP:           ip,
		Hostname:     hostnamePtr,
		Body:         bodyPtr,
		Params:       paramsPtr,
		Query:        queryPtr,
		Email:        email,
		Username:     username,
		Role:         role,
		LogLevel:     &logLevel,
		Environment:  &environment,
		PID:          &pid,
		URL:          urlPtr,
	}
}

// filterSensitiveData filtra información sensible del body
func filterSensitiveData(body string) string {
	// Lista de campos sensibles a filtrar
	sensitiveFields := []string{"password", "mfa_code", "secret", "token", "backup_codes"}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		// Si no es JSON válido, retornar truncado
		if len(body) > 1000 {
			return body[:1000] + "...[truncated]"
		}
		return body
	}

	// Filtrar campos sensibles
	for _, field := range sensitiveFields {
		if _, exists := data[field]; exists {
			data[field] = "[FILTERED]"
		}
	}

	filteredJSON, _ := json.Marshal(data)
	filteredBody := string(filteredJSON)

	// Truncar si es muy largo
	if len(filteredBody) > 1000 {
		return filteredBody[:1000] + "...[truncated]"
	}

	return filteredBody
}

// determineLogLevel determina el nivel de log basado en el status code
func determineLogLevel(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return models.LogLevelSuccess
	case statusCode >= 300 && statusCode < 400:
		return models.LogLevelInfo
	case statusCode >= 400 && statusCode < 500:
		return models.LogLevelWarning
	case statusCode >= 500:
		return models.LogLevelError
	default:
		return models.LogLevelInfo
	}
}

// saveLogToDB guarda el log en la base de datos
func saveLogToDB(logEntry models.CreateLogRequest) {
	db := database.GetDB()
	if db == nil {
		fmt.Println("Error: No se pudo obtener conexión a la base de datos para logging")
		return
	}

	query := `
		INSERT INTO logs (
			method, path, protocol, status_code, response_time, user_agent, ip, hostname,
			body, params, query, email, username, role, log_level, environment,
			node_version, pid, timestamp, url, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)
	`

	_, err := db.Exec(context.Background(), query,
		logEntry.Method,
		logEntry.Path,
		logEntry.Protocol,
		logEntry.StatusCode,
		logEntry.ResponseTime,
		logEntry.UserAgent,
		logEntry.IP,
		logEntry.Hostname,
		logEntry.Body,
		logEntry.Params,
		logEntry.Query,
		logEntry.Email,
		logEntry.Username,
		logEntry.Role,
		logEntry.LogLevel,
		logEntry.Environment,
		logEntry.NodeVersion,
		logEntry.PID,
		time.Now(),
		logEntry.URL,
		time.Now(),
	)

	if err != nil {
		fmt.Printf("Error guardando log en base de datos: %v\n", err)
	}
}

// LogCustomEvent permite registrar eventos personalizados
func LogCustomEvent(level, message, userEmail, userRole string, additionalData map[string]interface{}) {
	logEntry := models.CreateLogRequest{
		Method:      "CUSTOM",
		Path:        "/custom-event",
		StatusCode:  200,
		IP:          "127.0.0.1",
		LogLevel:    &level,
		Environment: getEnvironment(),
	}

	if userEmail != "" {
		logEntry.Email = &userEmail
		logEntry.Username = &userEmail
	}

	if userRole != "" {
		logEntry.Role = &userRole
	}

	// Agregar datos adicionales al body
	if additionalData != nil {
		additionalData["message"] = message
		bodyJSON, _ := json.Marshal(additionalData)
		bodyStr := string(bodyJSON)
		logEntry.Body = &bodyStr
	} else {
		logEntry.Body = &message
	}

	go saveLogToDB(logEntry)
}

func getEnvironment() *string {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = models.EnvironmentDevelopment
	}
	return &env
}
