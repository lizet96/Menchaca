package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// RateLimitConfig configuración para rate limiting
type RateLimitConfig struct {
	Max        int           // Número máximo de requests
	Expiration time.Duration // Ventana de tiempo
	Message    string        // Mensaje de error personalizado
}

// DefaultRateLimit configuración por defecto para rate limiting
var DefaultRateLimit = RateLimitConfig{
	Max:        100,              // 100 requests
	Expiration: 15 * time.Minute, // por 15 minutos
	Message:    "Demasiadas peticiones, intenta más tarde",
}

// StrictRateLimit configuración estricta para endpoints sensibles
var StrictRateLimit = RateLimitConfig{
	Max:        10,               // 10 requests
	Expiration: 15 * time.Minute, // por 15 minutos
	Message:    "Límite de peticiones excedido para este endpoint",
}

// AuthRateLimit configuración para endpoints de autenticación
var AuthRateLimit = RateLimitConfig{
	Max:        20,               // 5 intentos
	Expiration: 30 * time.Minute, // por 15 minutos
	Message:    "Demasiados intentos de login, intenta más tarde",
}

// CreateRateLimiter crea un middleware de rate limiting con la configuración especificada
func CreateRateLimiter(config RateLimitConfig) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        config.Max,
		Expiration: config.Expiration,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Usar IP del cliente como clave
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       true,
				"message":     config.Message,
				"retry_after": int(config.Expiration.Seconds()),
			})
		},
		SkipFailedRequests:     false, // Contar también requests fallidos
		SkipSuccessfulRequests: false, // Contar también requests exitosos
	})
}

// DefaultRateLimiter middleware de rate limiting por defecto
func DefaultRateLimiter() fiber.Handler {
	return CreateRateLimiter(DefaultRateLimit)
}

// StrictRateLimiter middleware de rate limiting estricto
func StrictRateLimiter() fiber.Handler {
	return CreateRateLimiter(StrictRateLimit)
}

// AuthRateLimiter middleware de rate limiting para autenticación
func AuthRateLimiter() fiber.Handler {
	return CreateRateLimiter(AuthRateLimit)
}

// BodySizeLimit middleware para limitar el tamaño del cuerpo de la petición
func BodySizeLimit(maxSize int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if len(c.Body()) > maxSize {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
				"error":    true,
				"message":  "El tamaño de la petición excede el límite permitido",
				"max_size": maxSize,
			})
		}
		return c.Next()
	}
}

// RequestTimeoutMiddleware middleware para timeout de peticiones
func RequestTimeoutMiddleware(timeout time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Crear un contexto con timeout
		ctx := c.Context()
		ctx.SetUserValue("timeout", timeout)
		return c.Next()
	}
}

// SecurityHeaders middleware para agregar headers de seguridad
func SecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Headers de seguridad
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Content-Security-Policy", "default-src 'self'")
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		return c.Next()
	}
}
