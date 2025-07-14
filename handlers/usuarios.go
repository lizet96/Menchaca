package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lizet96/hospital-backend/database"
	"github.com/lizet96/hospital-backend/middleware"
	"github.com/lizet96/hospital-backend/models"
	"golang.org/x/crypto/bcrypt"
)

// Diccionario global de códigos intCode
var IntCodeMessages = map[string]string{
	// Usuarios
	"S01": "Login exitoso",
	"F01": "Login fallido",
	"S02": "Registro exitoso",
	"F02": "Registro fallido",
	"S03": "Actualización de usuario exitosa",
	"F03": "Error al actualizar usuario",
	"S04": "Eliminación de usuario exitosa",
	"F04": "Error al eliminar usuario",
	"S05": "Perfil obtenido exitosamente",
	"F05": "Error al obtener perfil",
	"S06": "Usuarios obtenidos exitosamente",
	"F06": "Error al obtener usuarios",
	// Consultas
	"S10": "Consulta creada exitosamente",
	"F10": "Error al crear consulta",
	"S11": "Consulta obtenida exitosamente",
	"F11": "Error al obtener consulta",
	"S12": "Consulta actualizada exitosamente",
	"F12": "Error al actualizar consulta",
	"S13": "Consulta eliminada exitosamente",
	"F13": "Error al eliminar consulta",
	// Expedientes
	"S20": "Expediente creado exitosamente",
	"F20": "Error al crear expediente",
	"S21": "Expediente obtenido exitosamente",
	"F21": "Error al obtener expediente",
	"S22": "Expediente actualizado exitosamente",
	"F22": "Error al actualizar expediente",
	"S23": "Expediente eliminado exitosamente",
	"F23": "Error al eliminar expediente",
	// Recetas
	"S30": "Receta creada exitosamente",
	"F30": "Error al crear receta",
	"S31": "Receta obtenida exitosamente",
	"F31": "Error al obtener receta",
	"S32": "Receta actualizada exitosamente",
	"F32": "Error al actualizar receta",
	"S33": "Receta eliminada exitosamente",
	"F33": "Error al eliminar receta",
	// Consultorios
	"S40": "Consultorio creado exitosamente",
	"F40": "Error al crear consultorio",
	"S41": "Consultorio obtenido exitosamente",
	"F41": "Error al obtener consultorio",
	"S42": "Consultorio actualizado exitosamente",
	"F42": "Error al actualizar consultorio",
	"S43": "Consultorio eliminado exitosamente",
	"F43": "Error al eliminar consultorio",
	// Horarios
	"S50": "Horario creado exitosamente",
	"F50": "Error al crear horario",
	"S51": "Horario obtenido exitosamente",
	"F51": "Error al obtener horario",
	"S52": "Horario actualizado exitosamente",
	"F52": "Error al actualizar horario",
	"S53": "Horario eliminado exitosamente",
	"F53": "Error al eliminar horario",
	// Reportes
	"S60": "Reporte generado exitosamente",
	"F60": "Error al generar reporte",
	"S61": "Estadísticas obtenidas exitosamente",
	"F61": "Error al obtener estadísticas",
}

type BodyResponse struct {
	IntCode string      `json:"intCode"`
	Data    interface{} `json:"data"`
}

type StandardResponse struct {
	StatusCode int          `json:"statusCode"`
	Body       BodyResponse `json:"body"`
}

// RegistrarUsuario crea un nuevo usuario en el sistema
func RegistrarUsuario(c *fiber.Ctx) error {
	var usuario models.Usuario
	var err error

	if err = c.BodyParser(&usuario); err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": "Datos inválidos"}},
			},
		})
	}

	// Validar contraseña segura
	if err = middleware.ValidateStrongPassword(usuario.Password); err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": err.Error()}},
			},
		})
	}

	// Validar que el id_rol sea válido
	if usuario.IDRol <= 0 {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": "ID de rol es requerido"}},
			},
		})
	}

	// Verificar que el rol existe en la base de datos
	var existeRol int
	var nombreRol string
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*), COALESCE(MAX(nombre), '') FROM rol WHERE id_rol = $1 AND activo = true", usuario.IDRol).Scan(&existeRol, &nombreRol)
	if err != nil || existeRol == 0 {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": "Rol inválido"}},
			},
		})
	}

	// Validar campos requeridos
	if usuario.Nombre == "" || usuario.Apellido == "" || usuario.Email == "" || usuario.Password == "" {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": "Nombre, apellido, email y contraseña son requeridos"}},
			},
		})
	}

	// Validar fecha de nacimiento
	if usuario.FechaNacimiento == "" {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": "Fecha de nacimiento es requerida"}},
			},
		})
	}

	// Validar formato de fecha (opcional pero recomendado)
	if _, err = time.Parse("2006-01-02", usuario.FechaNacimiento); err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": "Formato de fecha inválido. Use YYYY-MM-DD"}},
			},
		})
	}

	// Verificar si el email ya existe
	var existeEmail int
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Usuario WHERE email = $1", usuario.Email).Scan(&existeEmail)
	if err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": "Error al crear el usuario"}},
			},
		})
	}
	if existeEmail > 0 {
		return c.Status(409).JSON(StandardResponse{
			StatusCode: 409,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": "El email ya está registrado"}},
			},
		})
	}

	// Encriptar la contraseña
	var hashedPassword []byte
	hashedPassword, err = bcrypt.GenerateFromPassword([]byte(usuario.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": "Error al procesar la contraseña"}},
			},
		})
	}

	// Insertar usuario en la base de datos (SIN campo tipo)
	var nuevoID int
	err = database.GetDB().QueryRow(context.Background(),
		`INSERT INTO Usuario (nombre, apellido, fecha_nacimiento, id_rol, email, password, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id_usuario`,
		usuario.Nombre, usuario.Apellido, usuario.FechaNacimiento, usuario.IDRol, usuario.Email, string(hashedPassword), time.Now(), time.Now()).Scan(&nuevoID)

	if err != nil {
		// Agregar logging temporal
		fmt.Printf("Error al insertar usuario: %v\n", err)
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": "Error al crear el usuario", "details": err.Error()}},
			},
		})
	}

	// Crear respuesta sin datos sensibles (SIN campo tipo)
	respuesta := models.UsuarioResponse{
		ID:              nuevoID,
		Nombre:          usuario.Nombre,
		Apellido:        usuario.Apellido,
		FechaNacimiento: usuario.FechaNacimiento,
		IDRol:           &usuario.IDRol,
		Email:           usuario.Email,
		CreatedAt:       time.Now(),
	}

	return c.Status(201).JSON(StandardResponse{
		StatusCode: 201,
		Body: BodyResponse{
			IntCode: "S02",
			Data:    []interface{}{fiber.Map{"usuario": respuesta}},
		},
	})
}

// Login autentica un usuario con MFA obligatorio
func Login(c *fiber.Ctx) error {
	var loginReq models.LoginMFARequest // Cambiar a LoginMFARequest
	if err := c.BodyParser(&loginReq); err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F01",
				Data:    []interface{}{fiber.Map{"error": "Datos inválidos"}},
			},
		})
	}

	// Buscar usuario por email (SIN campo tipo)
	var usuario models.Usuario
	var mfaSecret sql.NullString
	var backupCodes sql.NullString
	var rolNombre string

	err := database.GetDB().QueryRow(context.Background(),
		`SELECT u.id_usuario, u.nombre, u.apellido, u.fecha_nacimiento, u.id_rol, u.email, u.password, 
		        u.mfa_enabled, u.mfa_secret, u.backup_codes, u.created_at, r.nombre
		 FROM Usuario u 
		 JOIN Rol r ON u.id_rol = r.id_rol 
		 WHERE u.email = $1`,
		loginReq.Email).Scan(&usuario.IDUsuario, &usuario.Nombre, &usuario.Apellido, &usuario.FechaNacimiento,
		&usuario.IDRol, &usuario.Email, &usuario.Password, &usuario.MFAEnabled, &mfaSecret, &backupCodes,
		&usuario.CreatedAt, &rolNombre)

	if err != nil {
		return c.Status(401).JSON(StandardResponse{
			StatusCode: 401,
			Body: BodyResponse{
				IntCode: "F01",
				Data:    []interface{}{fiber.Map{"error": "Credenciales inválidas"}},
			},
		})
	}

	// Asignar valores manejando NULL
	usuario.MFASecret = mfaSecret.String
	usuario.BackupCodes = backupCodes.String

	// Verificar contraseña
	err = bcrypt.CompareHashAndPassword([]byte(usuario.Password), []byte(loginReq.Password))
	if err != nil {
		return c.Status(401).JSON(StandardResponse{
			StatusCode: 401,
			Body: BodyResponse{
				IntCode: "F01",
				Data:    []interface{}{fiber.Map{"error": "Credenciales inválidas"}},
			},
		})
	}

	// CASO 1: Usuario NO tiene MFA configurado - Generar automáticamente
	if !usuario.MFAEnabled || usuario.MFASecret == "" {
		if loginReq.MFACode == "" {
			// Primera fase: generar MFA automáticamente
			key, err := middleware.GenerateMFASecret(usuario.Email)
			if err != nil {
				return c.Status(500).JSON(StandardResponse{
					StatusCode: 500,
					Body: BodyResponse{
						IntCode: "F02",
						Data:    []interface{}{fiber.Map{"error": "Error al generar MFA"}},
					},
				})
			}

			// Generar códigos de respaldo
			backupCodes, err := middleware.GenerateBackupCodes()
			if err != nil {
				return c.Status(500).JSON(StandardResponse{
					StatusCode: 500,
					Body: BodyResponse{
						IntCode: "F02",
						Data:    []interface{}{fiber.Map{"error": "Error al generar códigos de respaldo"}},
					},
				})
			}

			// Guardar secreto MFA en la base de datos
			_, err = database.GetDB().Exec(context.Background(),
				"UPDATE Usuario SET mfa_secret = $1, backup_codes = $2 WHERE id_usuario = $3",
				key.Secret(), strings.Join(backupCodes, ","), usuario.IDUsuario)
			if err != nil {
				return c.Status(500).JSON(StandardResponse{
					StatusCode: 500,
					Body: BodyResponse{
						IntCode: "F02",
						Data:    []interface{}{fiber.Map{"error": "Error al guardar MFA"}},
					},
				})
			}

			// Devolver QR para escanear
			return c.JSON(StandardResponse{
				StatusCode: 200,
				Body: BodyResponse{
					IntCode: "S01",
					Data: []interface{}{models.LoginMFAResponse{
						RequiresMFA: true,
						QRCodeURL:   key.URL(),
						Secret:      key.Secret(),
						BackupCodes: backupCodes,
					}},
				},
			})
		} else {
			// Segunda fase: validar código MFA recién configurado
			// Obtener el secreto recién guardado
			var newSecret string
			err := database.GetDB().QueryRow(context.Background(),
				"SELECT mfa_secret FROM Usuario WHERE id_usuario = $1", usuario.IDUsuario).Scan(&newSecret)
			if err != nil {
				return c.Status(500).JSON(StandardResponse{
					StatusCode: 500,
					Body: BodyResponse{
						IntCode: "F02",
						Data:    []interface{}{fiber.Map{"error": "Error interno"}},
					},
				})
			}

			// Validar código TOTP
			if !middleware.ValidateTOTP(newSecret, loginReq.MFACode) {
				return c.Status(401).JSON(StandardResponse{
					StatusCode: 401,
					Body: BodyResponse{
						IntCode: "F01",
						Data:    []interface{}{fiber.Map{"error": "Código MFA inválido"}},
					},
				})
			}

			// Activar MFA después de validación exitosa
			_, err = database.GetDB().Exec(context.Background(),
				"UPDATE Usuario SET mfa_enabled = true WHERE id_usuario = $1", usuario.IDUsuario)
			if err != nil {
				return c.Status(500).JSON(StandardResponse{
					StatusCode: 500,
					Body: BodyResponse{
						IntCode: "F02",
						Data:    []interface{}{fiber.Map{"error": "Error al activar MFA"}},
					},
				})
			}
		}
	} else {
		// CASO 2: Usuario YA tiene MFA configurado
		if loginReq.MFACode == "" {
			// Primera fase: solicitar código MFA
			return c.JSON(StandardResponse{
				StatusCode: 200,
				Body: BodyResponse{
					IntCode: "S01",
					Data: []interface{}{models.LoginMFAResponse{
						RequiresMFA: true,
					}},
				},
			})
		}

		// Segunda fase: validar código MFA existente
		validTOTP := middleware.ValidateTOTP(usuario.MFASecret, loginReq.MFACode)
		validBackup := false
		newBackupCodes := usuario.BackupCodes

		if !validTOTP {
			validBackup, newBackupCodes = middleware.ValidateBackupCode(usuario.BackupCodes, loginReq.MFACode)
			if validBackup {
				// Actualizar códigos de respaldo
				_, err = database.GetDB().Exec(context.Background(),
					"UPDATE Usuario SET backup_codes = $1 WHERE id_usuario = $2",
					newBackupCodes, usuario.IDUsuario)
			}
		}

		if !validTOTP && !validBackup {
			return c.Status(401).JSON(StandardResponse{
				StatusCode: 401,
				Body: BodyResponse{
					IntCode: "F01",
					Data:    []interface{}{fiber.Map{"error": "Código MFA inválido"}},
				},
			})
		}
	}

	// GENERAR TOKENS JWT (usando id_rol)
	accessToken, refreshToken, err := middleware.GenerateTokenPair(usuario.IDUsuario, usuario.IDRol)
	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": "Error al generar tokens"}},
			},
		})
	}

	// Guardar refresh token
	_, err = database.GetDB().Exec(context.Background(),
		`INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`,
		usuario.IDUsuario, refreshToken, time.Now().Add(middleware.RefreshTokenDuration))

	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": "Error al guardar refresh token"}},
			},
		})
	}

	// Respuesta exitosa con tokens
	return c.JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S01",
			Data: []interface{}{models.LoginMFAResponse{
				RequiresMFA:  false,
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
				ExpiresIn:    int(middleware.AccessTokenDuration.Seconds()),
				Usuario: models.UsuarioResponse{
					ID:              usuario.IDUsuario,
					Nombre:          usuario.Nombre,
					Apellido:        usuario.Apellido,
					FechaNacimiento: usuario.FechaNacimiento,
					IDRol:           &usuario.IDRol,
					Email:           usuario.Email,
					CreatedAt:       usuario.CreatedAt,
				},
			}},
		},
	})
}

// ObtenerUsuarios obtiene todos los usuarios
func ObtenerUsuarios(c *fiber.Ctx) error {
	rows, err := database.GetDB().Query(context.Background(),
		`SELECT u.id_usuario, u.nombre, u.apellido, u.fecha_nacimiento, u.id_rol, u.email, u.created_at, r.nombre as rol_nombre
		 FROM Usuario u 
		 JOIN Rol r ON u.id_rol = r.id_rol 
		 ORDER BY u.created_at DESC`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener usuarios",
		})
	}
	defer rows.Close()

	var usuarios []models.UsuarioResponse
	for rows.Next() {
		var usuario models.UsuarioResponse
		var rolNombre string
		err := rows.Scan(&usuario.ID, &usuario.Nombre, &usuario.Apellido, &usuario.FechaNacimiento,
			&usuario.IDRol, &usuario.Email, &usuario.CreatedAt, &rolNombre)
		if err != nil {
			continue
		}
		usuarios = append(usuarios, usuario)
	}

	return c.JSON(fiber.Map{
		"usuarios": usuarios,
		"total":    len(usuarios),
	})
}

// ObtenerUsuarioPorID obtiene un usuario específico
func ObtenerUsuarioPorID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Verificar permisos usando el nuevo sistema
	userID := c.Locals("user_id").(int)
	if !hasPermission(c, "usuarios_read") && userID != id {
		return c.Status(403).JSON(fiber.Map{
			"error": "No tienes permisos para ver este usuario",
		})
	}

	var usuario models.UsuarioResponse
	err = database.GetDB().QueryRow(context.Background(),
		`SELECT u.id_usuario, u.nombre, u.apellido, u.fecha_nacimiento, u.id_rol, u.email, u.created_at
		 FROM Usuario u WHERE u.id_usuario = $1`, id).Scan(
		&usuario.ID, &usuario.Nombre, &usuario.Apellido, &usuario.FechaNacimiento, &usuario.IDRol, &usuario.Email, &usuario.CreatedAt)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Usuario no encontrado",
		})
	}

	return c.JSON(usuario)
}

// ActualizarUsuario actualiza los datos de un usuario
func ActualizarUsuario(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Verificar permisos usando el nuevo sistema
	userID := c.Locals("user_id").(int)
	if !hasPermission(c, "usuarios_update") && userID != id {
		return c.Status(403).JSON(fiber.Map{
			"error": "No tienes permisos para actualizar este usuario",
		})
	}

	var usuario models.Usuario
	if err := c.BodyParser(&usuario); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Si se está actualizando la contraseña, validarla
	if usuario.Password != "" {
		if err := middleware.ValidateStrongPassword(usuario.Password); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		// Hashear nueva contraseña
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(usuario.Password), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Error al procesar contraseña",
			})
		}
		usuario.Password = string(hashedPassword)
	}

	// Actualizar usuario
	_, err = database.GetDB().Exec(context.Background(),
		"UPDATE Usuario SET nombre = $1, email = $2, updated_at = $3 WHERE id_usuario = $4",
		usuario.Nombre, usuario.Email, time.Now(), id)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al actualizar usuario",
		})
	}

	return c.JSON(fiber.Map{
		"mensaje": "Usuario actualizado exitosamente",
	})
}

// EliminarUsuario elimina un usuario (solo admin)
func EliminarUsuario(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Verificar que el usuario existe
	var existe int
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Usuario WHERE id_usuario = $1", id).Scan(&existe)
	if err != nil || existe == 0 {
		return c.Status(404).JSON(fiber.Map{
			"error": "Usuario no encontrado",
		})
	}

	// Eliminar usuario
	_, err = database.GetDB().Exec(context.Background(),
		"DELETE FROM Usuario WHERE id_usuario = $1", id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al eliminar usuario",
		})
	}

	return c.JSON(fiber.Map{
		"mensaje": "Usuario eliminado exitosamente",
	})
}

// ObtenerPerfil obtiene el perfil del usuario autenticado
func ObtenerPerfil(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)

	var usuario models.UsuarioResponse
	err := database.GetDB().QueryRow(context.Background(),
		`SELECT u.id_usuario, u.nombre, u.apellido, u.fecha_nacimiento, u.id_rol, u.email, u.created_at
		 FROM Usuario u WHERE u.id_usuario = $1`, userID).Scan(
		&usuario.ID, &usuario.Nombre, &usuario.Apellido, &usuario.FechaNacimiento, &usuario.IDRol, &usuario.Email, &usuario.CreatedAt)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Usuario no encontrado",
		})
	}

	return c.JSON(usuario)
}

// RefreshToken renueva un access token usando un refresh token
func RefreshToken(c *fiber.Ctx) error {
	var refreshReq models.RefreshRequest
	if err := c.BodyParser(&refreshReq); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Validar refresh token
	claims, err := middleware.ValidateToken(refreshReq.RefreshToken, "refresh")
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Refresh token inválido o expirado",
		})
	}

	// Verificar que el refresh token existe en la base de datos y no está revocado
	var exists bool
	err = database.GetDB().QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM refresh_tokens 
         WHERE token = $1 AND user_id = $2 AND expires_at > NOW() AND is_revoked = false)`,
		refreshReq.RefreshToken, claims.UserID).Scan(&exists)

	if err != nil || !exists {
		return c.Status(401).JSON(fiber.Map{
			"error": "Refresh token inválido o revocado",
		})
	}

	// Generar nuevo par de tokens
	newAccessToken, newRefreshToken, err := middleware.GenerateTokenPair(claims.UserID, claims.IDRol)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al generar nuevos tokens",
		})
	}

	// Revocar el refresh token anterior
	_, err = database.GetDB().Exec(context.Background(),
		"UPDATE refresh_tokens SET is_revoked = true WHERE token = $1",
		refreshReq.RefreshToken)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al revocar token anterior",
		})
	}

	// Guardar nuevo refresh token
	_, err = database.GetDB().Exec(context.Background(),
		`INSERT INTO refresh_tokens (user_id, token, expires_at) 
         VALUES ($1, $2, $3)`,
		claims.UserID, newRefreshToken, time.Now().Add(middleware.RefreshTokenDuration))

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al guardar nuevo refresh token",
		})
	}

	// Crear respuesta
	respuesta := models.RefreshResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(middleware.AccessTokenDuration.Seconds()),
	}

	return c.JSON(respuesta)
}

// Logout revoca todos los refresh tokens del usuario
func Logout(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)

	// Revocar todos los refresh tokens del usuario
	_, err := database.GetDB().Exec(context.Background(),
		"UPDATE refresh_tokens SET is_revoked = true WHERE user_id = $1",
		userID)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al cerrar sesión",
		})
	}

	return c.JSON(fiber.Map{
		"mensaje": "Sesión cerrada exitosamente",
	})
}

// SetupMFA configura MFA para el usuario
func SetupMFA(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)

	var req models.MFASetupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Datos inválidos"})
	}

	// Verificar contraseña actual
	var currentPassword string
	err := database.GetDB().QueryRow(context.Background(),
		"SELECT password FROM Usuario WHERE id_usuario = $1", userID).Scan(&currentPassword)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error interno"})
	}

	err = bcrypt.CompareHashAndPassword([]byte(currentPassword), []byte(req.Password))
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Contraseña incorrecta"})
	}

	// Obtener email del usuario
	var email string
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT email FROM Usuario WHERE id_usuario = $1", userID).Scan(&email)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error interno"})
	}

	// Generar secreto MFA
	key, err := middleware.GenerateMFASecret(email)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error al generar MFA"})
	}

	// Generar códigos de respaldo
	backupCodes, err := middleware.GenerateBackupCodes()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error al generar códigos de respaldo"})
	}

	// Guardar secreto (temporalmente, hasta verificación)
	_, err = database.GetDB().Exec(context.Background(),
		"UPDATE Usuario SET mfa_secret = $1, backup_codes = $2 WHERE id_usuario = $3",
		key.Secret(), strings.Join(backupCodes, ","), userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error al guardar MFA"})
	}

	return c.JSON(models.MFASetupResponse{
		Secret:      key.Secret(),
		QRCodeURL:   key.URL(),
		BackupCodes: backupCodes,
	})
}

// VerifyMFA verifica y activa MFA
func VerifyMFA(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)

	var req models.MFAVerifyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Datos inválidos"})
	}

	// Obtener secreto temporal
	var secret string
	err := database.GetDB().QueryRow(context.Background(),
		"SELECT mfa_secret FROM Usuario WHERE id_usuario = $1", userID).Scan(&secret)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error interno"})
	}

	// Validar código TOTP
	if !middleware.ValidateTOTP(secret, req.Code) {
		return c.Status(400).JSON(fiber.Map{"error": "Código MFA inválido"})
	}

	// Activar MFA
	_, err = database.GetDB().Exec(context.Background(),
		"UPDATE Usuario SET mfa_enabled = true WHERE id_usuario = $1", userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error al activar MFA"})
	}

	return c.JSON(fiber.Map{"message": "MFA activado exitosamente"})
}

// DisableMFA desactiva MFA
func DisableMFA(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)

	var req models.MFAVerifyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Datos inválidos"})
	}

	// Obtener datos MFA
	var secret string
	var backupCodes string
	err := database.GetDB().QueryRow(context.Background(),
		"SELECT mfa_secret, backup_codes FROM Usuario WHERE id_usuario = $1", userID).Scan(&secret, &backupCodes)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error interno"})
	}

	// Validar código TOTP o código de respaldo
	valid := middleware.ValidateTOTP(secret, req.Code)
	if !valid {
		validBackup, _ := middleware.ValidateBackupCode(backupCodes, req.Code)
		if !validBackup {
			return c.Status(400).JSON(fiber.Map{"error": "Código inválido"})
		}
	}

	// Desactivar MFA
	_, err = database.GetDB().Exec(context.Background(),
		"UPDATE Usuario SET mfa_enabled = false, mfa_secret = NULL, backup_codes = NULL WHERE id_usuario = $1", userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error al desactivar MFA"})
	}

	return c.JSON(fiber.Map{"message": "MFA desactivado exitosamente"})
}

// LoginWithMFA - Función corregida
func LoginWithMFA(c *fiber.Ctx) error {
	var loginReq models.LoginMFARequest
	if err := c.BodyParser(&loginReq); err != nil {
		fmt.Printf("❌ Error parsing body: %v\n", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	fmt.Printf("🔍 Login attempt for email: %s\n", loginReq.Email)

	// Buscar usuario por email (SIN campo tipo)
	var usuario models.Usuario
	var mfaSecret sql.NullString
	var backupCodes sql.NullString
	var rolNombre string

	err := database.GetDB().QueryRow(context.Background(),
		`SELECT u.id_usuario, u.nombre, u.apellido, u.fecha_nacimiento, u.id_rol, u.email, u.password, 
		        u.mfa_enabled, u.mfa_secret, u.backup_codes, u.created_at, r.nombre
		 FROM Usuario u 
		 JOIN Rol r ON u.id_rol = r.id_rol 
		 WHERE u.email = $1`,
		loginReq.Email).Scan(&usuario.IDUsuario, &usuario.Nombre, &usuario.Apellido, &usuario.FechaNacimiento,
		&usuario.IDRol, &usuario.Email, &usuario.Password, &usuario.MFAEnabled, &mfaSecret, &backupCodes,
		&usuario.CreatedAt, &rolNombre)

	if err != nil {
		fmt.Printf("❌ Database error: %v\n", err)
		return c.Status(401).JSON(fiber.Map{
			"error": "Credenciales inválidas",
		})
	}

	// Asignar valores manejando NULL
	usuario.MFASecret = mfaSecret.String
	usuario.BackupCodes = backupCodes.String

	fmt.Printf(" User found: %s (ID: %d), MFA enabled: %v\n", usuario.Email, usuario.IDUsuario, usuario.MFAEnabled)
	fmt.Printf(" Password length: %d, starts with: %s\n", len(usuario.Password), usuario.Password[:10])

	// Verificar contraseña
	err = bcrypt.CompareHashAndPassword([]byte(usuario.Password), []byte(loginReq.Password))
	if err != nil {
		fmt.Printf(" Password verification failed: %v\n", err)
		return c.Status(401).JSON(fiber.Map{
			"error": "Credenciales inválidas",
		})
	}

	fmt.Printf("✅ Password verified successfully\n")

	// Si MFA está habilitado
	if usuario.MFAEnabled {
		if loginReq.MFACode == "" {
			// Primera fase: solicitar código MFA
			return c.JSON(models.LoginMFAResponse{
				RequiresMFA: true,
			})
		}

		// Segunda fase: validar código MFA
		validTOTP := middleware.ValidateTOTP(usuario.MFASecret, loginReq.MFACode)
		validBackup := false
		newBackupCodes := usuario.BackupCodes

		if !validTOTP {
			validBackup, newBackupCodes = middleware.ValidateBackupCode(usuario.BackupCodes, loginReq.MFACode)
			if validBackup {
				// Actualizar códigos de respaldo
				_, err = database.GetDB().Exec(context.Background(),
					"UPDATE Usuario SET backup_codes = $1 WHERE id_usuario = $2",
					newBackupCodes, usuario.IDUsuario)
			}
		}

		if !validTOTP && !validBackup {
			return c.Status(401).JSON(fiber.Map{
				"error": "Código MFA inválido",
			})
		}
	}

	// Generar tokens JWT (usando id_rol)
	accessToken, refreshToken, err := middleware.GenerateTokenPair(usuario.IDUsuario, usuario.IDRol)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al generar tokens",
		})
	}

	// Guardar refresh token
	_, err = database.GetDB().Exec(context.Background(),
		`INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`,
		usuario.IDUsuario, refreshToken, time.Now().Add(middleware.RefreshTokenDuration))

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al guardar refresh token",
		})
	}

	return c.JSON(models.LoginMFAResponse{
		RequiresMFA:  false,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(middleware.AccessTokenDuration.Seconds()),
		Usuario: models.UsuarioResponse{
			ID:              usuario.IDUsuario,
			Nombre:          usuario.Nombre,
			Apellido:        usuario.Apellido,
			FechaNacimiento: usuario.FechaNacimiento,
			IDRol:           &usuario.IDRol,
			Email:           usuario.Email,
			CreatedAt:       usuario.CreatedAt,
		},
	})
}

// CambiarPassword permite cambiar la contraseña del usuario
func CambiarPassword(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)

	type ChangePasswordRequest struct {
		CurrentPassword string `json:"current_password" validate:"required"`
		NewPassword     string `json:"new_password" validate:"required"`
	}

	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Datos inválidos"})
	}

	// Validar nueva contraseña
	if err := middleware.ValidateStrongPassword(req.NewPassword); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	// Verificar contraseña actual
	var currentPassword string
	err := database.GetDB().QueryRow(context.Background(),
		"SELECT password FROM Usuario WHERE id_usuario = $1", userID).Scan(&currentPassword)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error interno"})
	}

	err = bcrypt.CompareHashAndPassword([]byte(currentPassword), []byte(req.CurrentPassword))
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Contraseña actual incorrecta"})
	}

	// Hashear nueva contraseña
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error al procesar contraseña"})
	}

	// Actualizar contraseña
	_, err = database.GetDB().Exec(context.Background(),
		"UPDATE Usuario SET password = $1, updated_at = CURRENT_TIMESTAMP WHERE id_usuario = $2",
		string(hashedPassword), userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error al actualizar contraseña"})
	}

	return c.JSON(fiber.Map{"message": "Contraseña actualizada exitosamente"})
}

// Función auxiliar para verificar permisos
func hasPermission(c *fiber.Ctx, permiso string) bool {
	userID := c.Locals("user_id").(int)

	var tienePermiso bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM Usuario u
			JOIN Rol r ON u.id_rol = r.id_rol
			JOIN RolPermiso rp ON r.id_rol = rp.id_rol
			JOIN Permiso p ON rp.id_permiso = p.id_permiso
			WHERE u.id_usuario = $1 AND p.nombre = $2
		)
	`

	err := database.GetDB().QueryRow(context.Background(), query, userID, permiso).Scan(&tienePermiso)
	return err == nil && tienePermiso
}

// ObtenerPermisosPorRol obtiene todos los permisos de un rol específico
func ObtenerPermisosPorRol(c *fiber.Ctx) error {
	idRol, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F06",
				Data:    []interface{}{fiber.Map{"error": "ID de rol inválido"}},
			},
		})
	}

	// Verificar que el rol existe
	var rol struct {
		IDRol       int    `json:"id_rol"`
		Nombre      string `json:"nombre"`
		Descripcion string `json:"descripcion"`
	}

	err = database.GetDB().QueryRow(context.Background(),
		"SELECT id_rol, nombre, descripcion FROM Rol WHERE id_rol = $1 AND activo = true", idRol).Scan(
		&rol.IDRol, &rol.Nombre, &rol.Descripcion)

	if err != nil {
		return c.Status(404).JSON(StandardResponse{
			StatusCode: 404,
			Body: BodyResponse{
				IntCode: "F06",
				Data:    []interface{}{fiber.Map{"error": "Rol no encontrado"}},
			},
		})
	}

	// Obtener permisos del rol
	rows, err := database.GetDB().Query(context.Background(),
		`SELECT p.id_permiso, p.nombre, p.descripcion, p.recurso, p.accion
		 FROM Permiso p
		 JOIN RolPermiso rp ON p.id_permiso = rp.id_permiso
		 WHERE rp.id_rol = $1
		 ORDER BY p.recurso, p.accion`, idRol)

	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F06",
				Data:    []interface{}{fiber.Map{"error": "Error al obtener permisos"}},
			},
		})
	}
	defer rows.Close()

	var permisos []map[string]interface{}
	for rows.Next() {
		var permiso struct {
			IDPermiso   int    `json:"id_permiso"`
			Nombre      string `json:"nombre"`
			Descripcion string `json:"descripcion"`
			Recurso     string `json:"recurso"`
			Accion      string `json:"accion"`
		}

		err := rows.Scan(&permiso.IDPermiso, &permiso.Nombre, &permiso.Descripcion, &permiso.Recurso, &permiso.Accion)
		if err != nil {
			continue
		}

		permisos = append(permisos, map[string]interface{}{
			"id_permiso":  permiso.IDPermiso,
			"nombre":      permiso.Nombre,
			"descripcion": permiso.Descripcion,
			"recurso":     permiso.Recurso,
			"accion":      permiso.Accion,
		})
	}

	// Crear respuesta con rol y permisos
	response := map[string]interface{}{
		"id_rol":      rol.IDRol,
		"nombre":      rol.Nombre,
		"descripcion": rol.Descripcion,
		"permisos":    permisos,
	}

	return c.JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S06",
			Data:    []interface{}{response},
		},
	})
}

// CrearUsuario crea un nuevo usuario
func CrearUsuario(c *fiber.Ctx) error {
	// Verificar permisos usando el nuevo sistema
	if !hasPermission(c, "usuarios_create") {
		return c.Status(403).JSON(fiber.Map{
			"error": "No tienes permisos para crear usuarios",
		})
	}

	var usuario models.Usuario
	if err := c.BodyParser(&usuario); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Validaciones
	if usuario.Nombre == "" || usuario.Apellido == "" || usuario.Email == "" || usuario.Password == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Nombre, apellido, email y contraseña son requeridos",
		})
	}

	// Validar que id_rol sea obligatorio
	if usuario.IDRol <= 0 {
		return c.Status(400).JSON(fiber.Map{
			"error": "El rol es requerido",
		})
	}

	// Verificar que el rol existe
	var rolExiste bool
	err := database.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM rol WHERE id_rol = $1 AND activo = true)", usuario.IDRol).Scan(&rolExiste)
	if err != nil || !rolExiste {
		return c.Status(400).JSON(fiber.Map{
			"error": "Rol no válido",
		})
	}

	// Validar contraseña segura
	if err := middleware.ValidateStrongPassword(usuario.Password); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Verificar si el email ya existe
	var existe int
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Usuario WHERE email = $1", usuario.Email).Scan(&existe)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al verificar email",
		})
	}
	if existe > 0 {
		return c.Status(400).JSON(fiber.Map{
			"error": "El email ya está registrado",
		})
	}

	// Encriptar contraseña
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(usuario.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al procesar contraseña",
		})
	}

	// Insertar usuario sin el campo 'tipo'
	query := `
		INSERT INTO Usuario (nombre, apellido, email, password, fecha_nacimiento, telefono, direccion, id_rol, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP) 
		RETURNING id_usuario
	`

	var nuevoID int
	err = database.GetDB().QueryRow(context.Background(), query,
		usuario.Nombre, usuario.Apellido, usuario.Email, string(hashedPassword),
		usuario.FechaNacimiento, usuario.Telefono, usuario.Direccion, usuario.IDRol).Scan(&nuevoID)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al crear usuario",
		})
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "Usuario creado exitosamente",
		"user_id": nuevoID,
	})
}
