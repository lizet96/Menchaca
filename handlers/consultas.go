package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lizet96/hospital-backend/database"
	"github.com/lizet96/hospital-backend/models"
)

// CrearConsulta crea una nueva consulta médica
func CrearConsulta(c *fiber.Ctx) error {
	var consulta models.Consulta
	if err := c.BodyParser(&consulta); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Verificar permisos usando el nuevo sistema de roles
	userRole := c.Locals("user_role").(string)
	if userRole != "medico" && userRole != "admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo médicos pueden crear consultas",
		})
	}

	// Si es médico, debe ser el mismo que está en la consulta
	if userRole == "medico" {
		userID := c.Locals("user_id").(int)
		if consulta.IDMedico != userID {
			return c.Status(403).JSON(fiber.Map{
				"error": "No puedes crear consultas para otro médico",
			})
		}
	}

	// Verificar que el horario esté disponible
	var disponible bool
	err := database.GetDB().QueryRow(context.Background(),
		"SELECT consulta_disponible FROM Horario WHERE id_horario = $1", consulta.IDHorario).Scan(&disponible)
	if err != nil || !disponible {
		return c.Status(400).JSON(fiber.Map{
			"error": "Horario no disponible",
		})
	}

	// Insertar consulta (sin campo tipo)
	var nuevoID int
	err = database.GetDB().QueryRow(context.Background(),
		`INSERT INTO Consulta (diagnostico, costo, id_paciente, id_medico, id_horario, fecha, estado, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id_consulta`,
		consulta.Diagnostico, consulta.Costo, consulta.IDPaciente, consulta.IDMedico,
		consulta.IDHorario, time.Now(), "programada", time.Now(), time.Now()).Scan(&nuevoID)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al crear la consulta",
		})
	}

	// Marcar horario como no disponible
	_, err = database.GetDB().Exec(context.Background(),
		"UPDATE Horario SET consulta_disponible = false WHERE id_horario = $1", consulta.IDHorario)
	if err != nil {
		// Log error but don't fail the request
	}

	return c.Status(201).JSON(fiber.Map{
		"mensaje":     "Consulta creada exitosamente",
		"id_consulta": nuevoID,
	})
}

// ObtenerConsultas obtiene las consultas según el rol de usuario
func ObtenerConsultas(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	userRole := c.Locals("user_role").(string)

	var query string
	var args []interface{}

	switch userRole {
	case "admin":
		// Admin puede ver todas las consultas
		query = `SELECT c.id_consulta, c.diagnostico, c.costo, c.id_paciente, c.id_medico, 
				 c.id_horario, c.fecha, c.estado, c.created_at,
				 p.nombre as paciente_nombre, m.nombre as medico_nombre
				 FROM Consulta c
				 JOIN Usuario p ON c.id_paciente = p.id_usuario
				 JOIN Usuario m ON c.id_medico = m.id_usuario
				 ORDER BY c.fecha DESC`
	case "medico":
		// Médico solo ve sus consultas
		query = `SELECT c.id_consulta, c.diagnostico, c.costo, c.id_paciente, c.id_medico,
				 c.id_horario, c.fecha, c.estado, c.created_at,
				 p.nombre as paciente_nombre, m.nombre as medico_nombre
				 FROM Consulta c
				 JOIN Usuario p ON c.id_paciente = p.id_usuario
				 JOIN Usuario m ON c.id_medico = m.id_usuario
				 WHERE c.id_medico = $1
				 ORDER BY c.fecha DESC`
		args = append(args, userID)
	case "paciente":
		// Paciente solo ve sus consultas
		query = `SELECT c.id_consulta, c.diagnostico, c.costo, c.id_paciente, c.id_medico,
				 c.id_horario, c.fecha, c.estado, c.created_at,
				 p.nombre as paciente_nombre, m.nombre as medico_nombre
				 FROM Consulta c
				 JOIN Usuario p ON c.id_paciente = p.id_usuario
				 JOIN Usuario m ON c.id_medico = m.id_usuario
				 WHERE c.id_paciente = $1
				 ORDER BY c.fecha DESC`
		args = append(args, userID)
	default:
		return c.Status(403).JSON(fiber.Map{
			"error": "Tipo de usuario no autorizado",
		})
	}

	rows, err := database.GetDB().Query(context.Background(), query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener consultas",
		})
	}
	defer rows.Close()

	type ConsultaDetalle struct {
		models.Consulta
		PacienteNombre string `json:"paciente_nombre"`
		MedicoNombre   string `json:"medico_nombre"`
	}

	var consultas []ConsultaDetalle
	for rows.Next() {
		var consulta ConsultaDetalle
		err := rows.Scan(&consulta.ID, &consulta.Diagnostico, &consulta.Costo,
			&consulta.IDPaciente, &consulta.IDMedico, &consulta.IDHorario, &consulta.Fecha,
			&consulta.Estado, &consulta.CreatedAt, &consulta.PacienteNombre, &consulta.MedicoNombre)
		if err != nil {
			continue
		}
		consultas = append(consultas, consulta)
	}

	return c.JSON(fiber.Map{
		"consultas": consultas,
		"total":     len(consultas),
	})
}

// ActualizarConsulta actualiza una consulta existente
func ActualizarConsulta(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Verificar permisos
	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	if userRole != "medico" && userRole != "admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo médicos pueden actualizar consultas",
		})
	}

	// Si es médico, verificar que sea su consulta
	if userRole == "medico" {
		var medicoConsulta int
		err := database.GetDB().QueryRow(context.Background(),
			"SELECT id_medico FROM Consulta WHERE id_consulta = $1", id).Scan(&medicoConsulta)
		if err != nil || medicoConsulta != userID {
			return c.Status(403).JSON(fiber.Map{
				"error": "No puedes actualizar esta consulta",
			})
		}
	}

	var consulta models.Consulta
	if err := c.BodyParser(&consulta); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Actualizar consulta (sin campo tipo)
	_, err = database.GetDB().Exec(context.Background(),
		`UPDATE Consulta SET diagnostico = $1, costo = $2, estado = $3, updated_at = $4
		 WHERE id_consulta = $5`,
		consulta.Diagnostico, consulta.Costo, consulta.Estado, time.Now(), id)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al actualizar consulta",
		})
	}

	return c.JSON(fiber.Map{
		"mensaje": "Consulta actualizada exitosamente",
	})
}

// ObtenerConsultaPorID obtiene una consulta específica por ID
func ObtenerConsultaPorID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	var consulta models.Consulta
	var nombrePaciente, nombreMedico, nombreConsultorio string

	query := `
		SELECT c.id_consulta, c.id_paciente, c.id_medico, c.id_horario, 
		       c.fecha_consulta, c.motivo, c.diagnostico, c.tratamiento, 
		       c.observaciones, c.estado, c.created_at, c.updated_at,
		       u1.nombre as nombre_paciente, u2.nombre as nombre_medico,
		       co.nombre as nombre_consultorio
		FROM Consulta c
		JOIN Usuario u1 ON c.id_paciente = u1.id_usuario
		JOIN Usuario u2 ON c.id_medico = u2.id_usuario
		JOIN Horario h ON c.id_horario = h.id_horario
		JOIN Consultorio co ON h.id_consultorio = co.id_consultorio
		WHERE c.id_consulta = $1`

	// Agregar filtros según el rol de usuario
	if userRole == "paciente" {
		query += " AND c.id_paciente = $2"
		err = database.GetDB().QueryRow(context.Background(), query, id, userID).Scan(
			&consulta.ID, &consulta.IDPaciente, &consulta.IDMedico, &consulta.IDHorario,
			&consulta.FechaConsulta, &consulta.Motivo, &consulta.Diagnostico, &consulta.Tratamiento,
			&consulta.Observaciones, &consulta.Estado, &consulta.CreatedAt, &consulta.UpdatedAt,
			&nombrePaciente, &nombreMedico, &nombreConsultorio)
	} else if userRole == "medico" {
		query += " AND c.id_medico = $2"
		err = database.GetDB().QueryRow(context.Background(), query, id, userID).Scan(
			&consulta.ID, &consulta.IDPaciente, &consulta.IDMedico, &consulta.IDHorario,
			&consulta.FechaConsulta, &consulta.Motivo, &consulta.Diagnostico, &consulta.Tratamiento,
			&consulta.Observaciones, &consulta.Estado, &consulta.CreatedAt, &consulta.UpdatedAt,
			&nombrePaciente, &nombreMedico, &nombreConsultorio)
	} else {
		// Admin y enfermera pueden ver todas
		err = database.GetDB().QueryRow(context.Background(), query, id).Scan(
			&consulta.ID, &consulta.IDPaciente, &consulta.IDMedico, &consulta.IDHorario,
			&consulta.FechaConsulta, &consulta.Motivo, &consulta.Diagnostico, &consulta.Tratamiento,
			&consulta.Observaciones, &consulta.Estado, &consulta.CreatedAt, &consulta.UpdatedAt,
			&nombrePaciente, &nombreMedico, &nombreConsultorio)
	}

	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Consulta no encontrada",
		})
	}

	return c.JSON(fiber.Map{
		"consulta":           consulta,
		"nombre_paciente":    nombrePaciente,
		"nombre_medico":      nombreMedico,
		"nombre_consultorio": nombreConsultorio,
	})
}

// ObtenerConsultasPorPaciente obtiene todas las consultas de un paciente específico
func ObtenerConsultasPorPaciente(c *fiber.Ctx) error {
	pacienteID, err := strconv.Atoi(c.Params("paciente_id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID de paciente inválido",
		})
	}

	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	// Verificar permisos
	if userRole == "paciente" && pacienteID != userID {
		return c.Status(403).JSON(fiber.Map{
			"error": "No puedes ver las consultas de otro paciente",
		})
	}

	query := `
		SELECT c.id_consulta, c.id_paciente, c.id_medico, c.id_horario, 
		       c.fecha_consulta, c.motivo, c.diagnostico, c.tratamiento, 
		       c.observaciones, c.estado, c.created_at, c.updated_at,
		       u2.nombre as nombre_medico, co.nombre as nombre_consultorio
		FROM Consulta c
		JOIN Usuario u2 ON c.id_medico = u2.id_usuario
		JOIN Horario h ON c.id_horario = h.id_horario
		JOIN Consultorio co ON h.id_consultorio = co.id_consultorio
		WHERE c.id_paciente = $1
		ORDER BY c.fecha_consulta DESC`

	rows, err := database.GetDB().Query(context.Background(), query, pacienteID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener consultas",
		})
	}
	defer rows.Close()

	var consultas []map[string]interface{}
	for rows.Next() {
		var consulta models.Consulta
		var nombreMedico, nombreConsultorio string

		err := rows.Scan(
			&consulta.ID, &consulta.IDPaciente, &consulta.IDMedico, &consulta.IDHorario,
			&consulta.FechaConsulta, &consulta.Motivo, &consulta.Diagnostico, &consulta.Tratamiento,
			&consulta.Observaciones, &consulta.Estado, &consulta.CreatedAt, &consulta.UpdatedAt,
			&nombreMedico, &nombreConsultorio)
		if err != nil {
			continue
		}

		consultas = append(consultas, map[string]interface{}{
			"consulta":           consulta,
			"nombre_medico":      nombreMedico,
			"nombre_consultorio": nombreConsultorio,
		})
	}

	return c.JSON(fiber.Map{
		"consultas": consultas,
		"total":     len(consultas),
	})
}

// ObtenerConsultasPorMedico obtiene todas las consultas de un médico específico
func ObtenerConsultasPorMedico(c *fiber.Ctx) error {
	medicoID, err := strconv.Atoi(c.Params("medico_id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID de médico inválido",
		})
	}

	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	// Verificar permisos
	if userRole == "medico" && medicoID != userID {
		return c.Status(403).JSON(fiber.Map{
			"error": "No puedes ver las consultas de otro médico",
		})
	}

	query := `
		SELECT c.id_consulta, c.id_paciente, c.id_medico, c.id_horario, 
		       c.fecha_consulta, c.motivo, c.diagnostico, c.tratamiento, 
		       c.observaciones, c.estado, c.created_at, c.updated_at,
		       u1.nombre as nombre_paciente, co.nombre as nombre_consultorio
		FROM Consulta c
		JOIN Usuario u1 ON c.id_paciente = u1.id_usuario
		JOIN Horario h ON c.id_horario = h.id_horario
		JOIN Consultorio co ON h.id_consultorio = co.id_consultorio
		WHERE c.id_medico = $1
		ORDER BY c.fecha_consulta DESC`

	rows, err := database.GetDB().Query(context.Background(), query, medicoID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener consultas",
		})
	}
	defer rows.Close()

	var consultas []map[string]interface{}
	for rows.Next() {
		var consulta models.Consulta
		var nombrePaciente, nombreConsultorio string

		err := rows.Scan(
			&consulta.ID, &consulta.IDPaciente, &consulta.IDMedico, &consulta.IDHorario,
			&consulta.FechaConsulta, &consulta.Motivo, &consulta.Diagnostico, &consulta.Tratamiento,
			&consulta.Observaciones, &consulta.Estado, &consulta.CreatedAt, &consulta.UpdatedAt,
			&nombrePaciente, &nombreConsultorio)
		if err != nil {
			continue
		}

		consultas = append(consultas, map[string]interface{}{
			"consulta":           consulta,
			"nombre_paciente":    nombrePaciente,
			"nombre_consultorio": nombreConsultorio,
		})
	}

	return c.JSON(fiber.Map{
		"consultas": consultas,
		"total":     len(consultas),
	})
}

// CompletarConsulta marca una consulta como completada
func CompletarConsulta(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	// Solo médicos pueden completar consultas
	if userRole != "medico" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo médicos pueden completar consultas",
		})
	}

	// Verificar que la consulta pertenece al médico
	var medicoID int
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT id_medico FROM Consulta WHERE id_consulta = $1", id).Scan(&medicoID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Consulta no encontrada",
		})
	}

	if medicoID != userID {
		return c.Status(403).JSON(fiber.Map{
			"error": "No puedes completar esta consulta",
		})
	}

	// Actualizar estado a completada
	_, err = database.GetDB().Exec(context.Background(),
		"UPDATE Consulta SET estado = 'completada', updated_at = $1 WHERE id_consulta = $2",
		time.Now(), id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al completar consulta",
		})
	}

	return c.JSON(fiber.Map{
		"mensaje": "Consulta completada exitosamente",
	})
}

// CancelarConsulta cancela una consulta y libera el horario
func CancelarConsulta(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Verificar permisos
	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	// Obtener información de la consulta
	var consulta models.Consulta
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT id_consulta, id_paciente, id_medico, id_horario, estado FROM Consulta WHERE id_consulta = $1", id).Scan(
		&consulta.ID, &consulta.IDPaciente, &consulta.IDMedico, &consulta.IDHorario, &consulta.Estado)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Consulta no encontrada",
		})
	}

	// Verificar permisos específicos
	if userRole == "paciente" && consulta.IDPaciente != userID {
		return c.Status(403).JSON(fiber.Map{
			"error": "No puedes cancelar esta consulta",
		})
	}
	if userRole == "medico" && consulta.IDMedico != userID {
		return c.Status(403).JSON(fiber.Map{
			"error": "No puedes cancelar esta consulta",
		})
	}

	// Verificar que la consulta se pueda cancelar
	if consulta.Estado == "completada" || consulta.Estado == "cancelada" {
		return c.Status(400).JSON(fiber.Map{
			"error": "No se puede cancelar una consulta completada o ya cancelada",
		})
	}

	// Cancelar consulta
	_, err = database.GetDB().Exec(context.Background(),
		"UPDATE Consulta SET estado = 'cancelada', updated_at = $1 WHERE id_consulta = $2",
		time.Now(), id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al cancelar consulta",
		})
	}

	// Liberar horario
	_, err = database.GetDB().Exec(context.Background(),
		"UPDATE Horario SET consulta_disponible = true WHERE id_horario = $1", consulta.IDHorario)
	if err != nil {
		// Log error but don't fail the request
	}

	return c.JSON(fiber.Map{
		"mensaje": "Consulta cancelada exitosamente",
	})
}
