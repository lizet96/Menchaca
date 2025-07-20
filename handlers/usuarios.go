package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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

// RegistrarUsuario crea un nuevo usuario en el sistema
func RegistrarUsuario(c *fiber.Ctx) error {
	log.Printf("🔍 RegistrarUsuario: Iniciando registro de usuario")
	var usuario models.Usuario
	var err error

	if err = c.BodyParser(&usuario); err != nil {
		// Log intento de registro con datos inválidos
		middleware.LogCustomEvent(
			models.LogLevelError,
			"Intento de registro con datos inválidos",
			"",
			"",
			map[string]interface{}{
				"ip":     c.IP(),
				"action": "register_failed_invalid_data",
				"error":  err.Error(),
			},
		)
		log.Printf("❌ RegistrarUsuario: Error parsing body: %v", err)
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": "Los datos enviados no son válidos. Verifique el formato de la información."}},
			},
		})
	}

	// Validar campos requeridos con mensajes específicos
	var missingFields []string
	if strings.TrimSpace(usuario.Nombre) == "" {
		missingFields = append(missingFields, "nombre")
	}
	if strings.TrimSpace(usuario.Apellido) == "" {
		missingFields = append(missingFields, "apellido")
	}
	if strings.TrimSpace(usuario.Email) == "" {
		missingFields = append(missingFields, "correo electrónico")
	}
	if strings.TrimSpace(usuario.Password) == "" {
		missingFields = append(missingFields, "contraseña")
	}
	if strings.TrimSpace(usuario.FechaNacimiento) == "" {
		missingFields = append(missingFields, "fecha de nacimiento")
	}
	if usuario.IDRol <= 0 {
		missingFields = append(missingFields, "rol")
	}

	if len(missingFields) > 0 {
		errorMsg := "Los siguientes campos son obligatorios: " + strings.Join(missingFields, ", ")
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": errorMsg}},
			},
		})
	}

	// Validar formato de email
	if !strings.Contains(usuario.Email, "@") || !strings.Contains(usuario.Email, ".") {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": "El formato del correo electrónico no es válido. Debe contener @ y un dominio válido."}},
			},
		})
	}

	// Validar contraseña segura con mensaje detallado
	if err = middleware.ValidateStrongPassword(usuario.Password); err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": "Contraseña no válida: " + err.Error() + ". La contraseña debe tener al menos 12 caracteres, incluyendo mayúsculas, minúsculas, números y caracteres especiales (!@#$%^&*()_+-=[]{}|;:,.<>?)."}},
			},
		})
	}

	// Validar formato de fecha
	if _, err = time.Parse("2006-01-02", usuario.FechaNacimiento); err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": "El formato de la fecha de nacimiento no es válido. Use el formato YYYY-MM-DD (ejemplo: 1990-12-25)."}},
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
				Data:    []interface{}{fiber.Map{"error": "El rol seleccionado no es válido o no está disponible. Por favor, seleccione un rol válido."}},
			},
		})
	}

	// Verificar si el email ya existe
	var existeEmail int
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Usuario WHERE email = $1", usuario.Email).Scan(&existeEmail)
	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F01",
				Data:    []interface{}{fiber.Map{"error": "Error interno del servidor al verificar el correo electrónico"}},
			},
		})
	}
	if existeEmail > 0 {
		// Log intento de registro con email duplicado
		middleware.LogCustomEvent(
			models.LogLevelWarning,
			"Intento de registro con email duplicado",
			usuario.Email,
			"",
			map[string]interface{}{
				"email":  usuario.Email,
				"ip":     c.IP(),
				"action": "register_failed_duplicate_email",
			},
		)
		return c.Status(409).JSON(StandardResponse{
			StatusCode: 409,
			Body: BodyResponse{
				IntCode: "F02",
				Data:    []interface{}{fiber.Map{"error": "El correo electrónico " + usuario.Email + " ya está registrado. Por favor, use un correo diferente o inicie sesión si ya tiene una cuenta."}},
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

	// Log evento de registro exitoso
	middleware.LogCustomEvent(
		models.LogLevelSuccess,
		"Usuario registrado exitosamente",
		usuario.Email,
		"",
		map[string]interface{}{
			"new_user_id": nuevoID,
			"email":       usuario.Email,
			"ip":          c.IP(),
			"action":      "register_success",
		},
	)

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
	log.Printf("🔍 Login: Iniciando proceso de autenticación")
	var loginReq models.LoginMFARequest // Cambiar a LoginMFARequest
	if err := c.BodyParser(&loginReq); err != nil {
		log.Printf("❌ Login: Error parsing body: %v", err)
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F01",
				Data:    []interface{}{fiber.Map{"error": "Los datos de inicio de sesión no son válidos. Verifique el formato de la información enviada."}},
			},
		})
	}

	// Validar campos requeridos
	if strings.TrimSpace(loginReq.Email) == "" || strings.TrimSpace(loginReq.Password) == "" {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F01",
				Data:    []interface{}{fiber.Map{"error": "El correo electrónico y la contraseña son obligatorios para iniciar sesión."}},
			},
		})
	}

	// Validar formato básico de email
	if !strings.Contains(loginReq.Email, "@") || !strings.Contains(loginReq.Email, ".") {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F01",
				Data:    []interface{}{fiber.Map{"error": "El formato del correo electrónico no es válido."}},
			},
		})
	}

	// Buscar usuario por email (SIN campo tipo)
	log.Printf("🔍 Login: Buscando usuario con email: %s", loginReq.Email)
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
		// Log intento de login fallido - usuario no encontrado
		middleware.LogCustomEvent(
			models.LogLevelWarning,
			"Intento de login con email inexistente",
			loginReq.Email,
			"",
			map[string]interface{}{
				"email":  loginReq.Email,
				"ip":     c.IP(),
				"action": "login_failed_user_not_found",
			},
		)
		log.Printf("❌ Login: Usuario no encontrado para email: %s, error: %v", loginReq.Email, err)
		return c.Status(401).JSON(StandardResponse{
			StatusCode: 401,
			Body: BodyResponse{
				IntCode: "F01",
				Data:    []interface{}{fiber.Map{"error": "No se encontró una cuenta con este correo electrónico. Verifique que el correo sea correcto o regístrese si no tiene una cuenta."}},
			},
		})
	}

	// Asignar valores manejando NULL
	usuario.MFASecret = mfaSecret.String
	usuario.BackupCodes = backupCodes.String

	// Verificar contraseña
	log.Printf("🔍 Login: Verificando contraseña para usuario ID: %d", usuario.IDUsuario)
	err = bcrypt.CompareHashAndPassword([]byte(usuario.Password), []byte(loginReq.Password))
	if err != nil {
		// Log intento de login fallido - contraseña incorrecta
		middleware.LogCustomEvent(
			models.LogLevelWarning,
			"Intento de login con contraseña incorrecta",
			usuario.Email,
			"",
			map[string]interface{}{
				"user_id": usuario.IDUsuario,
				"email":   usuario.Email,
				"ip":      c.IP(),
				"action":  "login_failed_wrong_password",
			},
		)
		log.Printf("❌ Login: Contraseña incorrecta para usuario ID: %d", usuario.IDUsuario)
		return c.Status(401).JSON(StandardResponse{
			StatusCode: 401,
			Body: BodyResponse{
				IntCode: "F01",
				Data:    []interface{}{fiber.Map{"error": "La contraseña ingresada es incorrecta. Por favor, verifique su contraseña e intente nuevamente."}},
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
	log.Printf("✅ Login: Autenticación exitosa para usuario ID: %d, generando tokens", usuario.IDUsuario)
	accessToken, refreshToken, err := middleware.GenerateTokenPair(usuario.IDUsuario, usuario.IDRol)
	if err != nil {
		log.Printf("❌ Login: Error generando tokens para usuario ID: %d, error: %v", usuario.IDUsuario, err)
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

	// Log evento de login exitoso
	middleware.LogCustomEvent(
		models.LogLevelSuccess,
		"Login exitoso",
		usuario.Email,
		rolNombre,
		map[string]interface{}{
			"user_id": usuario.IDUsuario,
			"ip":      c.IP(),
			"action":  "login_success",
		},
	)

	// Respuesta exitosa con tokens
	log.Printf("✅ Login: Login completado exitosamente para usuario ID: %d", usuario.IDUsuario)
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
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F06",
				Data:    []interface{}{fiber.Map{"error": "Error al obtener usuarios"}},
			},
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

	return c.JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S06",
			Data:    []interface{}{fiber.Map{"usuarios": usuarios, "total": len(usuarios)}},
		},
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
	log.Printf("🔍 ActualizarUsuario: Iniciando actualización de usuario")
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		log.Printf("❌ ActualizarUsuario: ID inválido: %v", err)
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
		fmt.Printf("❌ CrearUsuario: Error parsing body: %v\n", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}
	fmt.Printf("✅ CrearUsuario: Body parsed successfully\n")

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
	query := "UPDATE Usuario SET nombre = $1, apellido = $2, email = $3, fecha_nacimiento = $4, id_rol = $5, updated_at = $6"
	args := []interface{}{usuario.Nombre, usuario.Apellido, usuario.Email, usuario.FechaNacimiento, usuario.IDRol, time.Now()}

	// Si hay contraseña, incluirla en la actualización
	if usuario.Password != "" {
		query += ", password = $7 WHERE id_usuario = $8"
		args = append(args, usuario.Password, id)
	} else {
		query += " WHERE id_usuario = $7"
		args = append(args, id)
	}

	_, err = database.GetDB().Exec(context.Background(), query, args...)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al actualizar usuario",
		})
	}

	// Log evento de actualización de usuario
	middleware.LogCustomEvent(
		models.LogLevelInfo,
		"Usuario actualizado",
		usuario.Email,
		"",
		map[string]interface{}{
			"updated_user_id": id,
			"updated_email":   usuario.Email,
			"updated_by":      c.Locals("user_email"),
			"action":          "user_updated",
		},
	)

	return c.JSON(fiber.Map{
		"mensaje": "Usuario actualizado exitosamente",
	})
}

// EliminarUsuario elimina un usuario (solo admin)
func EliminarUsuario(c *fiber.Ctx) error {
	log.Printf("🔍 EliminarUsuario: Iniciando eliminación de usuario")
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		log.Printf("❌ EliminarUsuario: ID inválido: %v", err)
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

	// Obtener email del usuario antes de eliminarlo para el log
	var emailUsuario string
	err = database.GetDB().QueryRow(context.Background(), "SELECT email FROM usuarios WHERE id_usuario = $1", id).Scan(&emailUsuario)
	if err != nil {
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

	// Log evento de eliminación de usuario
	middleware.LogCustomEvent(
		models.LogLevelWarning,
		"Usuario eliminado",
		emailUsuario,
		"",
		map[string]interface{}{
			"deleted_user_id": id,
			"deleted_email":   emailUsuario,
			"deleted_by":      c.Locals("user_email"),
			"action":          "user_deleted",
		},
	)

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
	// Verificar si el usuario está autenticado
	userID, ok := c.Locals("user_id").(int)
	if !ok {
		// Si no hay usuario autenticado, aún así devolver éxito
		// para evitar errores en el frontend
		return c.JSON(fiber.Map{
			"mensaje": "Sesión cerrada exitosamente",
		})
	}

	// Revocar todos los refresh tokens del usuario
	_, err := database.GetDB().Exec(context.Background(),
		"UPDATE refresh_tokens SET is_revoked = true WHERE user_id = $1",
		userID)

	if err != nil {
		// Log del error pero no fallar el logout
		fmt.Printf("⚠️ Error al revocar tokens para usuario %d: %v\n", userID, err)
	}

	return c.JSON(fiber.Map{
		"mensaje": "Sesión cerrada exitosamente",
	})
}

// LimpiarTodasLasSesiones revoca todos los refresh tokens del sistema (función administrativa)
func LimpiarTodasLasSesiones(c *fiber.Ctx) error {
	// Solo admin puede ejecutar esta función
	if !hasPermission(c, "usuarios_delete") {
		return c.Status(403).JSON(fiber.Map{
			"error": "No tienes permisos para ejecutar esta acción",
		})
	}

	// Revocar todos los refresh tokens del sistema
	result, err := database.GetDB().Exec(context.Background(),
		"UPDATE refresh_tokens SET is_revoked = true WHERE is_revoked = false")

	if err != nil {
		fmt.Printf("❌ Error al limpiar sesiones: %v\n", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al limpiar sesiones",
		})
	}

	rowsAffected := result.RowsAffected()
	fmt.Printf("✅ Sesiones limpiadas: %d tokens revocados\n", rowsAffected)

	return c.JSON(fiber.Map{
		"mensaje":          "Todas las sesiones han sido limpiadas exitosamente",
		"tokens_revocados": rowsAffected,
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
	log.Printf("🔍 CrearUsuario: Iniciando creación de usuario")

	// Verificar permisos usando el nuevo sistema
	if !hasPermission(c, "usuarios_create") {
		log.Printf("❌ CrearUsuario: Sin permisos")
		return c.Status(403).JSON(fiber.Map{
			"error": "No tienes permisos para crear usuarios",
		})
	}

	var usuario models.Usuario
	if err := c.BodyParser(&usuario); err != nil {
		log.Printf("❌ CrearUsuario: Error parsing body: %v", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}
	log.Printf("✅ CrearUsuario: Body parsed successfully: %+v", usuario)

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
	log.Printf("🔍 CrearUsuario: Validando contraseña")
	if err := middleware.ValidateStrongPassword(usuario.Password); err != nil {
		log.Printf("❌ CrearUsuario: Error validando contraseña: %v", err)
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	log.Printf("✅ CrearUsuario: Contraseña válida")

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
	log.Printf("🔍 CrearUsuario: Encriptando contraseña")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(usuario.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("❌ CrearUsuario: Error encriptando contraseña: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al procesar contraseña",
		})
	}
	log.Printf("✅ CrearUsuario: Contraseña encriptada")

	// Insertar usuario sin el campo 'tipo'
	query := `
		INSERT INTO Usuario (nombre, apellido, email, password, fecha_nacimiento, id_rol, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7) 
		RETURNING id_usuario
	`

	var nuevoID int
	log.Printf("🔍 CrearUsuario: Ejecutando INSERT query")
	err = database.GetDB().QueryRow(context.Background(), query,
		usuario.Nombre, usuario.Apellido, usuario.Email, string(hashedPassword),
		usuario.FechaNacimiento, usuario.IDRol, time.Now()).Scan(&nuevoID)

	if err != nil {
		log.Printf("❌ CrearUsuario: Error en INSERT: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al crear usuario",
		})
	}
	// Log evento de creación de usuario
	middleware.LogCustomEvent(
		models.LogLevelSuccess,
		"Usuario creado exitosamente",
		usuario.Email,
		"",
		map[string]interface{}{
			"new_user_id": nuevoID,
			"new_email":   usuario.Email,
			"created_by":  c.Locals("user_email"),
			"action":      "user_created",
		},
	)
	log.Printf("✅ CrearUsuario: Usuario creado con ID: %d", nuevoID)

	return c.Status(201).JSON(fiber.Map{
		"message": "Usuario creado exitosamente",
		"user_id": nuevoID,
	})
}

// ObtenerUsuariosPorRol obtiene usuarios por rol
func ObtenerUsuariosPorRol(c *fiber.Ctx) error {
	rolID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID de rol inválido",
		})
	}

	rows, err := database.GetDB().Query(context.Background(),
		`SELECT u.id_usuario, u.nombre, u.apellido, u.fecha_nacimiento, u.id_rol, u.email, u.created_at, r.nombre as rol_nombre
		 FROM Usuario u 
		 JOIN Rol r ON u.id_rol = r.id_rol 
		 WHERE u.id_rol = $1
		 ORDER BY u.created_at DESC`, rolID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener usuarios por rol",
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
		"success": true,
		"data":    usuarios,
		"total":   len(usuarios),
	})
}

// ObtenerPacientes obtiene todos los pacientes (usuarios con rol de paciente)
func ObtenerPacientes(c *fiber.Ctx) error {
	rows, err := database.GetDB().Query(context.Background(),
		`SELECT u.id_usuario, u.nombre, u.apellido, u.fecha_nacimiento, u.id_rol, u.email, u.created_at, r.nombre as rol_nombre
		 FROM Usuario u 
		 JOIN Rol r ON u.id_rol = r.id_rol 
		 WHERE r.nombre = 'paciente'
		 ORDER BY u.created_at DESC`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener pacientes",
		})
	}
	defer rows.Close()

	var pacientes []models.UsuarioResponse
	for rows.Next() {
		var paciente models.UsuarioResponse
		var rolNombre string
		err := rows.Scan(&paciente.ID, &paciente.Nombre, &paciente.Apellido, &paciente.FechaNacimiento,
			&paciente.IDRol, &paciente.Email, &paciente.CreatedAt, &rolNombre)
		if err != nil {
			continue
		}
		pacientes = append(pacientes, paciente)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    pacientes,
		"total":   len(pacientes),
	})
}

// ObtenerRoles obtiene todos los roles disponibles
func ObtenerRoles(c *fiber.Ctx) error {
	rows, err := database.GetDB().Query(context.Background(),
		`SELECT id_rol, nombre, descripcion FROM Rol ORDER BY nombre`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener roles",
		})
	}
	defer rows.Close()

	type Rol struct {
		IDRol       int    `json:"id_rol"`
		Nombre      string `json:"nombre"`
		Descripcion string `json:"descripcion"`
	}

	var roles []Rol
	for rows.Next() {
		var rol Rol
		err := rows.Scan(&rol.IDRol, &rol.Nombre, &rol.Descripcion)
		if err != nil {
			continue
		}
		roles = append(roles, rol)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    roles,
		"total":   len(roles),
	})
}
