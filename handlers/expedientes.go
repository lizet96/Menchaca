package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lizet96/hospital-backend/database"
	"github.com/lizet96/hospital-backend/models"
)

// CrearExpediente crea un nuevo expediente médico
func CrearExpediente(c *fiber.Ctx) error {
	var expediente models.Expediente
	if err := c.BodyParser(&expediente); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Solo médicos y admin pueden crear expedientes
	userType := c.Locals("user_type").(string)
	if userType != "medico" && userType != "admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo médicos pueden crear expedientes",
		})
	}

	// Verificar que el paciente existe
	var existePaciente int
	err := database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Usuario WHERE id_usuario = $1 AND tipo = 'paciente'", expediente.IDPaciente).Scan(&existePaciente)
	if err != nil || existePaciente == 0 {
		return c.Status(400).JSON(fiber.Map{
			"error": "Paciente no encontrado",
		})
	}

	// Verificar que no exista ya un expediente para este paciente
	var existeExpediente int
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Expediente WHERE id_paciente = $1", expediente.IDPaciente).Scan(&existeExpediente)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error interno del servidor",
		})
	}
	if existeExpediente > 0 {
		return c.Status(409).JSON(fiber.Map{
			"error": "Ya existe un expediente para este paciente",
		})
	}

	// Crear expediente
	var nuevoID int
	err = database.GetDB().QueryRow(context.Background(),
		`INSERT INTO Expediente (antecedentes, historial_clinico, seguro, id_paciente, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id_expediente`,
		expediente.Antecedentes, expediente.HistorialClinico, expediente.Seguro, expediente.IDPaciente,
		time.Now(), time.Now()).Scan(&nuevoID)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al crear expediente",
		})
	}

	return c.Status(201).JSON(fiber.Map{
		"mensaje":        "Expediente creado exitosamente",
		"id_expediente": nuevoID,
	})
}

// ObtenerExpedientes obtiene expedientes según permisos del usuario
func ObtenerExpedientes(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	userType := c.Locals("user_type").(string)

	var query string
	var args []interface{}

	switch userType {
	case "admin":
		// Admin puede ver todos los expedientes
		query = `SELECT e.id_expediente, e.antecedentes, e.historial_clinico, e.seguro, 
				 e.id_paciente, e.created_at, e.updated_at, u.nombre as paciente_nombre
				 FROM Expediente e
				 JOIN Usuario u ON e.id_paciente = u.id_usuario
				 ORDER BY e.created_at DESC`
	case "medico":
		// Médico puede ver expedientes de sus pacientes (que han tenido consultas con él)
		query = `SELECT DISTINCT e.id_expediente, e.antecedentes, e.historial_clinico, e.seguro,
				 e.id_paciente, e.created_at, e.updated_at, u.nombre as paciente_nombre
				 FROM Expediente e
				 JOIN Usuario u ON e.id_paciente = u.id_usuario
				 JOIN Consulta c ON e.id_paciente = c.id_paciente
				 WHERE c.id_medico = $1
				 ORDER BY e.created_at DESC`
		args = append(args, userID)
	case "paciente":
		// Paciente solo puede ver su propio expediente
		query = `SELECT e.id_expediente, e.antecedentes, e.historial_clinico, e.seguro,
				 e.id_paciente, e.created_at, e.updated_at, u.nombre as paciente_nombre
				 FROM Expediente e
				 JOIN Usuario u ON e.id_paciente = u.id_usuario
				 WHERE e.id_paciente = $1`
		args = append(args, userID)
	default:
		return c.Status(403).JSON(fiber.Map{
			"error": "Tipo de usuario no autorizado",
		})
	}

	rows, err := database.GetDB().Query(context.Background(), query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener expedientes",
		})
	}
	defer rows.Close()

	type ExpedienteDetalle struct {
		models.Expediente
		PacienteNombre string `json:"paciente_nombre"`
	}

	var expedientes []ExpedienteDetalle
	for rows.Next() {
		var expediente ExpedienteDetalle
		err := rows.Scan(&expediente.ID, &expediente.Antecedentes, &expediente.HistorialClinico,
			&expediente.Seguro, &expediente.IDPaciente, &expediente.CreatedAt, &expediente.UpdatedAt,
			&expediente.PacienteNombre)
		if err != nil {
			continue
		}
		expedientes = append(expedientes, expediente)
	}

	return c.JSON(fiber.Map{
		"expedientes": expedientes,
		"total":       len(expedientes),
	})
}

// ObtenerExpedientePorID obtiene un expediente específico
func ObtenerExpedientePorID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	userID := c.Locals("user_id").(int)
	userType := c.Locals("user_type").(string)

	// Obtener expediente
	var expediente models.Expediente
	var pacienteNombre string
	err = database.GetDB().QueryRow(context.Background(),
		`SELECT e.id_expediente, e.antecedentes, e.historial_clinico, e.seguro,
		 e.id_paciente, e.created_at, e.updated_at, u.nombre
		 FROM Expediente e
		 JOIN Usuario u ON e.id_paciente = u.id_usuario
		 WHERE e.id_expediente = $1`, id).Scan(
		&expediente.ID, &expediente.Antecedentes, &expediente.HistorialClinico,
		&expediente.Seguro, &expediente.IDPaciente, &expediente.CreatedAt,
		&expediente.UpdatedAt, &pacienteNombre)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Expediente no encontrado",
		})
	}

	// Verificar permisos
	switch userType {
	case "admin":
		// Admin puede ver cualquier expediente
	case "medico":
		// Médico solo puede ver expedientes de sus pacientes
		var tieneAcceso int
		err = database.GetDB().QueryRow(context.Background(),
			"SELECT COUNT(*) FROM Consulta WHERE id_paciente = $1 AND id_medico = $2",
			expediente.IDPaciente, userID).Scan(&tieneAcceso)
		if err != nil || tieneAcceso == 0 {
			return c.Status(403).JSON(fiber.Map{
				"error": "No tienes acceso a este expediente",
			})
		}
	case "paciente":
		// Paciente solo puede ver su propio expediente
		if expediente.IDPaciente != userID {
			return c.Status(403).JSON(fiber.Map{
				"error": "No tienes acceso a este expediente",
			})
		}
	default:
		return c.Status(403).JSON(fiber.Map{
			"error": "Tipo de usuario no autorizado",
		})
	}

	return c.JSON(fiber.Map{
		"expediente":      expediente,
		"paciente_nombre": pacienteNombre,
	})
}

// ActualizarExpediente actualiza un expediente existente
func ActualizarExpediente(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Solo médicos y admin pueden actualizar expedientes
	userType := c.Locals("user_type").(string)
	if userType != "medico" && userType != "admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo médicos pueden actualizar expedientes",
		})
	}

	// Si es médico, verificar que tenga acceso al expediente
	if userType == "medico" {
		userID := c.Locals("user_id").(int)
		var idPaciente int
		err := database.GetDB().QueryRow(context.Background(),
			"SELECT id_paciente FROM Expediente WHERE id_expediente = $1", id).Scan(&idPaciente)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{
				"error": "Expediente no encontrado",
			})
		}

		var tieneAcceso int
		err = database.GetDB().QueryRow(context.Background(),
			"SELECT COUNT(*) FROM Consulta WHERE id_paciente = $1 AND id_medico = $2",
			idPaciente, userID).Scan(&tieneAcceso)
		if err != nil || tieneAcceso == 0 {
			return c.Status(403).JSON(fiber.Map{
				"error": "No tienes acceso a este expediente",
			})
		}
	}

	var expediente models.Expediente
	if err := c.BodyParser(&expediente); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Actualizar expediente
	_, err = database.GetDB().Exec(context.Background(),
		`UPDATE Expediente SET antecedentes = $1, historial_clinico = $2, seguro = $3, updated_at = $4
		 WHERE id_expediente = $5`,
		expediente.Antecedentes, expediente.HistorialClinico, expediente.Seguro, time.Now(), id)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al actualizar expediente",
		})
	}

	return c.JSON(fiber.Map{
		"mensaje": "Expediente actualizado exitosamente",
	})
}

// ObtenerExpedientePorPaciente obtiene todos los expedientes de un paciente específico
func ObtenerExpedientePorPaciente(c *fiber.Ctx) error {
	pacienteID, err := strconv.Atoi(c.Params("paciente_id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID de paciente inválido",
		})
	}

	userType := c.Locals("user_type").(string)
	userID := c.Locals("user_id").(int)

	// Verificar permisos
	if userType == "paciente" && pacienteID != userID {
		return c.Status(403).JSON(fiber.Map{
			"error": "No puedes ver los expedientes de otro paciente",
		})
	}

	query := `
		SELECT e.id_expediente, e.id_paciente, e.fecha_creacion, 
		       e.antecedentes_medicos, e.alergias, e.medicamentos_actuales, 
		       e.observaciones, e.created_at, e.updated_at,
		       u.nombre as nombre_paciente, u.email
		FROM Expediente e
		JOIN Usuario u ON e.id_paciente = u.id_usuario
		WHERE e.id_paciente = $1
		ORDER BY e.fecha_creacion DESC`

	rows, err := database.GetDB().Query(context.Background(), query, pacienteID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener expedientes",
		})
	}
	defer rows.Close()

	var expedientes []map[string]interface{}
	for rows.Next() {
		var expediente models.Expediente
		var nombrePaciente, email string

		err := rows.Scan(
			&expediente.ID, &expediente.IDPaciente, &expediente.FechaCreacion,
			&expediente.AntecedentesMedicos, &expediente.Alergias, &expediente.MedicamentosActuales,
			&expediente.Observaciones, &expediente.CreatedAt, &expediente.UpdatedAt,
			&nombrePaciente, &email)
		if err != nil {
			continue
		}

		expedientes = append(expedientes, map[string]interface{}{
			"expediente": expediente,
			"nombre_paciente": nombrePaciente,
			"email": email,
		})
	}

	return c.JSON(fiber.Map{
		"expedientes": expedientes,
		"total": len(expedientes),
	})
}

// EliminarExpediente elimina un expediente médico
func EliminarExpediente(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Solo admin puede eliminar expedientes
	userType := c.Locals("user_type").(string)
	if userType != "admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo administradores pueden eliminar expedientes",
		})
	}

	// Verificar que el expediente existe
	var existe int
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Expediente WHERE id_expediente = $1", id).Scan(&existe)
	if err != nil || existe == 0 {
		return c.Status(404).JSON(fiber.Map{
			"error": "Expediente no encontrado",
		})
	}

	// Eliminar expediente
	_, err = database.GetDB().Exec(context.Background(),
		"DELETE FROM Expediente WHERE id_expediente = $1", id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al eliminar expediente",
		})
	}

	return c.JSON(fiber.Map{
		"mensaje": "Expediente eliminado exitosamente",
	})
}