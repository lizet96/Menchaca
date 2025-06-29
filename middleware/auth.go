package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// Clave secreta para firmar los tokens JWT
var jwtSecret = []byte("clave_secreta_muy_segura_aqui")

// Claims personalizados para el JWT
type Claims struct {
	UserID   int    `json:"user_id"`
	UserType string `json:"user_type"`
	jwt.RegisteredClaims
}

// GenerateJWT genera un token JWT para un usuario
func GenerateJWT(userID int, userType string) (string, error) {
	claims := Claims{
		UserID:   userID,
		UserType: userType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// JWTMiddleware middleware para validar tokens JWT
func JWTMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Obtener el token del header Authorization
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(401).JSON(fiber.Map{
				"error": "Token de autorización requerido",
			})
		}

		// Verificar que el token tenga el formato "Bearer <token>"
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			return c.Status(401).JSON(fiber.Map{
				"error": "Formato de token inválido",
			})
		}

		// Validar el token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{
				"error": "Token inválido",
			})
		}

		// Extraer claims y guardarlos en el contexto
		claims, ok := token.Claims.(*Claims)
		if !ok {
			return c.Status(401).JSON(fiber.Map{
				"error": "Claims inválidos",
			})
		}

		// Guardar información del usuario en el contexto
		c.Locals("user_id", claims.UserID)
		c.Locals("user_type", claims.UserType)

		return c.Next()
	}
}

// RequireRole middleware para requerir un rol específico
func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userType, ok := c.Locals("user_type").(string)
		if !ok {
			return c.Status(403).JSON(fiber.Map{
				"error": "Tipo de usuario no encontrado",
			})
		}

		// Verificar si el usuario tiene uno de los roles permitidos
		for _, role := range allowedRoles {
			if userType == role {
				return c.Next()
			}
		}

		return c.Status(403).JSON(fiber.Map{
			"error": "Acceso denegado: permisos insuficientes",
		})
	}
}