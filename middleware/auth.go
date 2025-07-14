package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lizet96/hospital-backend/database"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// Clave secreta para firmar los tokens JWT
var jwtSecret = []byte("clave_secreta_muy_segura_aqui")

// Duraciones de tokens
const (
	AccessTokenDuration  = 10 * time.Minute   // 10 minutos
	RefreshTokenDuration = 7 * 24 * time.Hour // 7 días
)

// Claims personalizados para el JWT
type Claims struct {
	UserID int `json:"user_id"`
	IDRol  int `json:"id_rol"`
	jwt.RegisteredClaims
}

// Función actualizada
func GenerateTokenPair(userID int, idRol int) (string, string, error) {
	// Access Token
	accessClaims := Claims{
		UserID: userID,
		IDRol:  idRol,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "access",
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}

	// Refresh Token
	refreshClaims := Claims{
		UserID: userID,
		IDRol:  idRol,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(RefreshTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "refresh",
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}

// GenerateRefreshTokenString genera un string aleatorio para refresh token
func GenerateRefreshTokenString() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// ValidateToken valida un token JWT
func ValidateToken(tokenString string, expectedType string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, jwt.ErrInvalidKey
	}

	// Verificar el tipo de token usando Subject
	if claims.Subject != expectedType {
		return nil, jwt.ErrInvalidKey
	}

	return claims, nil
}

// JWTMiddleware middleware para validar access tokens
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

		// Validar el access token
		claims, err := ValidateToken(tokenString, "access")
		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"error": "Token inválido o expirado",
			})
		}

		// Obtener información del rol desde la base de datos
		var rolNombre string
		var idRol int
		err = database.GetDB().QueryRow(context.Background(), `
            SELECT u.id_rol, r.nombre 
            FROM Usuario u 
            JOIN Rol r ON u.id_rol = r.id_rol 
            WHERE u.id_usuario = $1 AND r.activo = true
        `, claims.UserID).Scan(&idRol, &rolNombre)

		if err != nil {
			return c.Status(401).JSON(fiber.Map{
				"error": "Usuario o rol no válido",
			})
		}

		// Guardar información del usuario en el contexto
		c.Locals("user_id", claims.UserID)
		c.Locals("user_role", rolNombre)
		c.Locals("id_rol", idRol)

		return c.Next()
	}
}

// Actualizar RequireRole para usar el nuevo sistema
func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole, ok := c.Locals("user_role").(string)
		if !ok {
			return c.Status(403).JSON(fiber.Map{
				"error": "Rol de usuario no encontrado",
			})
		}

		// Verificar si el usuario tiene uno de los roles permitidos
		for _, role := range allowedRoles {
			if userRole == role {
				return c.Next()
			}
		}

		return c.Status(403).JSON(fiber.Map{
			"error": "Acceso denegado: permisos insuficientes",
		})
	}
}

// Eliminar RequirePermissionHybrid completamente y usar solo RequirePermission
func RequirePermission(permiso string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("user_id").(int)
		if !ok {
			return c.Status(401).JSON(fiber.Map{
				"error": "Usuario no autenticado",
			})
		}

		// Verificar permiso en la base de datos
		var tienePermiso bool
		query := `
            SELECT EXISTS(
                SELECT 1 FROM Usuario u
                JOIN Rol r ON u.id_rol = r.id_rol
                JOIN RolPermiso rp ON r.id_rol = rp.id_rol
                JOIN Permiso p ON rp.id_permiso = p.id_permiso
                WHERE u.id_usuario = $1 AND p.nombre = $2 AND r.activo = true
            )
        `

		err := database.GetDB().QueryRow(context.Background(), query, userID, permiso).Scan(&tienePermiso)
		if err != nil || !tienePermiso {
			return c.Status(403).JSON(fiber.Map{
				"error": "Acceso denegado: permisos insuficientes",
			})
		}

		return c.Next()
	}
}

// ValidateStrongPassword valida que la contraseña cumpla con los requisitos de seguridad
func ValidateStrongPassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("la contraseña debe tener al menos 8 caracteres")
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("la contraseña debe contener al menos una letra mayúscula")
	}
	if !hasLower {
		return fmt.Errorf("la contraseña debe contener al menos una letra minúscula")
	}
	if !hasDigit {
		return fmt.Errorf("la contraseña debe contener al menos un número")
	}
	if !hasSpecial {
		return fmt.Errorf("la contraseña debe contener al menos un carácter especial")
	}

	return nil
}

// GenerateMFASecret genera un secreto para MFA
func GenerateMFASecret(email string) (*otp.Key, error) {
	return totp.Generate(totp.GenerateOpts{
		Issuer:      "Hospital Management System",
		AccountName: email,
		SecretSize:  32,
	})
}

// GenerateBackupCodes genera códigos de respaldo para MFA
func GenerateBackupCodes() ([]string, error) {
	codes := make([]string, 10)
	for i := 0; i < 10; i++ {
		bytes := make([]byte, 4)
		if _, err := rand.Read(bytes); err != nil {
			return nil, err
		}
		codes[i] = fmt.Sprintf("%08d", int(bytes[0])<<24|int(bytes[1])<<16|int(bytes[2])<<8|int(bytes[3]))
		codes[i] = codes[i][:8] // Asegurar 8 dígitos
	}
	return codes, nil
}

// ValidateTOTP valida un código TOTP
func ValidateTOTP(secret, code string) bool {
	return totp.Validate(code, secret)
}

// ValidateBackupCode valida un código de respaldo
func ValidateBackupCode(backupCodes, code string) (bool, string) {
	if backupCodes == "" {
		return false, backupCodes
	}

	codes := strings.Split(backupCodes, ",")
	for i, backupCode := range codes {
		if strings.TrimSpace(backupCode) == code {
			// Remover el código usado
			codes = append(codes[:i], codes[i+1:]...)
			return true, strings.Join(codes, ",")
		}
	}
	return false, backupCodes
}
