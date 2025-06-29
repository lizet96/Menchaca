package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lizet96/hospital-backend/database"
	"github.com/lizet96/hospital-backend/middleware"
	"github.com/lizet96/hospital-backend/models"
	"golang.org/x/crypto/bcrypt"
)

// RegistrarUsuario crea un nuevo usuario en el sistema
func RegistrarUsuario(c *fiber.Ctx) error {
	var usuario models.Usuario
	if err := c.BodyParser(&usuario); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Validar que el tipo de usuario sea válido
	tiposValidos := map[string]bool{
		"paciente":  true,
		"medico":    true,
		"enfermera": true,
		"admin":     true,
	}
	if !tiposValidos[usuario.Tipo] {
		return c.Status(400).JSON(fiber.Map{
			"error": "Tipo de usuario inválido",
		})
	}

	// Validar campos requeridos
	if usuario.Nombre == "" || usuario.Apellido == "" || usuario.Email == "" || usuario.Password == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Nombre, apellido, email y contraseña son requeridos",
		})
	}

	// Validar fecha de nacimiento
	if usuario.FechaNacimiento.IsZero() {
		return c.Status(400).JSON(fiber.Map{
			"error": "Fecha de nacimiento es requerida",
		})
	}

	// Verificar si el email ya existe
	var existeEmail int
	err := database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Usuario WHERE email = $1", usuario.Email).Scan(&existeEmail)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error interno del servidor",
		})
	}
	if existeEmail > 0 {
		return c.Status(409).JSON(fiber.Map{
			"error": "El email ya está registrado",
		})
	}

	// Encriptar la contraseña
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(usuario.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al procesar la contraseña",
		})
	}

	// Insertar usuario en la base de datos
	var nuevoID int
	err = database.GetDB().QueryRow(context.Background(),
		`INSERT INTO Usuario (nombre, apellido, fecha_nacimiento, tipo, email, password, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id_usuario`,
		usuario.Nombre, usuario.Apellido, usuario.FechaNacimiento, usuario.Tipo, usuario.Email, string(hashedPassword), time.Now(), time.Now()).Scan(&nuevoID)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al crear el usuario",
		})
	}

	// Crear respuesta sin datos sensibles
	respuesta := models.UsuarioResponse{
		ID:              nuevoID,
		Nombre:          usuario.Nombre,
		Apellido:        usuario.Apellido,
		FechaNacimiento: usuario.FechaNacimiento,
		Tipo:            usuario.Tipo,
		Email:           usuario.Email,
		CreatedAt:       time.Now(),
	}

	return c.Status(201).JSON(fiber.Map{
		"mensaje": "Usuario creado exitosamente",
		"usuario": respuesta,
	})
}

// Login autentica un usuario y devuelve un token JWT
func Login(c *fiber.Ctx) error {
	var loginReq models.LoginRequest
	if err := c.BodyParser(&loginReq); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Buscar usuario por email
	var usuario models.Usuario
	err := database.GetDB().QueryRow(context.Background(),
		"SELECT id_usuario, nombre, tipo, email, password, created_at FROM Usuario WHERE email = $1",
		loginReq.Email).Scan(&usuario.ID, &usuario.Nombre, &usuario.Tipo, &usuario.Email, &usuario.Password, &usuario.CreatedAt)

	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Credenciales inválidas",
		})
	}

	// Verificar contraseña
	err = bcrypt.CompareHashAndPassword([]byte(usuario.Password), []byte(loginReq.Password))
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Credenciales inválidas",
		})
	}

	// Generar token JWT
	token, err := middleware.GenerateJWT(usuario.ID, usuario.Tipo)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al generar token",
		})
	}

	// Crear respuesta
	respuesta := models.LoginResponse{
		Token: token,
		Usuario: models.UsuarioResponse{
			ID:        usuario.ID,
			Nombre:    usuario.Nombre,
			Tipo:      usuario.Tipo,
			Email:     usuario.Email,
			CreatedAt: usuario.CreatedAt,
		},
	}

	return c.JSON(respuesta)
}

// ObtenerUsuarios obtiene todos los usuarios (solo admin)
func ObtenerUsuarios(c *fiber.Ctx) error {
	rows, err := database.GetDB().Query(context.Background(),
		"SELECT id_usuario, nombre, tipo, email, created_at FROM Usuario ORDER BY created_at DESC")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener usuarios",
		})
	}
	defer rows.Close()

	var usuarios []models.UsuarioResponse
	for rows.Next() {
		var usuario models.UsuarioResponse
		err := rows.Scan(&usuario.ID, &usuario.Nombre, &usuario.Tipo, &usuario.Email, &usuario.CreatedAt)
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

	// Verificar permisos: admin puede ver cualquier usuario, otros solo su propio perfil
	userID := c.Locals("user_id").(int)
	userType := c.Locals("user_type").(string)

	if userType != "admin" && userID != id {
		return c.Status(403).JSON(fiber.Map{
			"error": "No tienes permisos para ver este usuario",
		})
	}

	var usuario models.UsuarioResponse
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT id_usuario, nombre, tipo, email, created_at FROM Usuario WHERE id_usuario = $1", id).Scan(
		&usuario.ID, &usuario.Nombre, &usuario.Tipo, &usuario.Email, &usuario.CreatedAt)

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

	// Verificar permisos
	userID := c.Locals("user_id").(int)
	userType := c.Locals("user_type").(string)

	if userType != "admin" && userID != id {
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
		"SELECT id_usuario, nombre, tipo, email, created_at FROM Usuario WHERE id_usuario = $1", userID).Scan(
		&usuario.ID, &usuario.Nombre, &usuario.Tipo, &usuario.Email, &usuario.CreatedAt)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Usuario no encontrado",
		})
	}

	return c.JSON(usuario)
}
