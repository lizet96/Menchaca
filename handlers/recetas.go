package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lizet96/hospital-backend/database"
	"github.com/lizet96/hospital-backend/middleware"
	"github.com/lizet96/hospital-backend/models"
)

// CrearReceta crea una nueva receta médica
func CrearReceta(c *fiber.Ctx) error {
	// El middleware RequirePermission ya verificó el permiso recetas_create
	// No necesitamos validación adicional de rol
	
	userRole := c.Locals("user_role").(string)
	medicoID := c.Locals("user_id").(int)

	var receta models.Receta
	if err := c.BodyParser(&receta); err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F70",
				Data:    []interface{}{fiber.Map{"error": "Datos inválidos"}},
			},
		})
	}

	// Validaciones (eliminar referencia a Dosis)
	if receta.Medicamento == "" || receta.IDPaciente == 0 || receta.IDConsultorio == 0 {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F70",
				Data:    []interface{}{fiber.Map{"error": "Medicamento, paciente y consultorio son requeridos"}},
			},
		})
	}

	// Verificar que el paciente existe y tiene rol de paciente
	var rolNombre string
	err := database.GetDB().QueryRow(context.Background(),
		`SELECT r.nombre FROM Usuario u 
		 JOIN Rol r ON u.id_rol = r.id_rol 
		 WHERE u.id_usuario = $1`, receta.IDPaciente).Scan(&rolNombre)
	if err != nil {
		return c.Status(404).JSON(StandardResponse{
			StatusCode: 404,
			Body: BodyResponse{
				IntCode: "F70",
				Data:    []interface{}{fiber.Map{"error": "Paciente no encontrado"}},
			},
		})
	}

	if rolNombre != "paciente" {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F70",
				Data:    []interface{}{fiber.Map{"error": "El usuario especificado no es un paciente"}},
			},
		})
	}

	// Verificar que el consultorio existe
	var consultorioExiste bool
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM Consultorio WHERE id_consultorio = $1)", receta.IDConsultorio).Scan(&consultorioExiste)
	if err != nil || !consultorioExiste {
		return c.Status(404).JSON(StandardResponse{
			StatusCode: 404,
			Body: BodyResponse{
				IntCode: "F70",
				Data:    []interface{}{fiber.Map{"error": "Consultorio no encontrado"}},
			},
		})
	}

	// Establecer fecha actual si no se proporciona
	if receta.Fecha.IsZero() {
		receta.Fecha = time.Now()
	}

	// Insertar receta (eliminar dosis del INSERT)
	query := `INSERT INTO Receta (fecha, medicamento, id_medico, id_paciente, id_consultorio) 
			  VALUES ($1, $2, $3, $4, $5) RETURNING id_receta`

	err = database.GetDB().QueryRow(context.Background(), query,
		receta.Fecha, receta.Medicamento, medicoID, receta.IDPaciente, receta.IDConsultorio).Scan(&receta.IDReceta)

	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F11",
				Data:    []interface{}{fiber.Map{"error": "Error al crear la receta"}},
			},
		})
	}

	receta.IDMedico = medicoID

	// Log evento de creación de receta
	userEmail := ""
	if email := c.Locals("user_email"); email != nil {
		userEmail = email.(string)
	}
	
	middleware.LogCustomEvent(
		models.LogLevelSuccess,
		"Receta creada",
		userEmail,
		userRole,
		map[string]interface{}{
			"receta_id":      receta.IDReceta,
			"paciente_id":    receta.IDPaciente,
			"medico_id":      medicoID,
			"consultorio_id": receta.IDConsultorio,
			"medicamento":    receta.Medicamento,
			"created_by":     userEmail,
			"action":         "receta_created",
		},
	)

	return c.Status(201).JSON(StandardResponse{
		StatusCode: 201,
		Body: BodyResponse{
			IntCode: "S01",
			Data: []interface{}{
				fiber.Map{
					"receta":  receta,
					"mensaje": "Receta creada exitosamente",
				},
			},
		},
	})
}

// ObtenerRecetas obtiene todas las recetas (con filtros según el rol)
func ObtenerRecetas(c *fiber.Ctx) error {
	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	var query string
	var args []interface{}

	switch userRole {
	case "admin":
		// Admin puede ver todas las recetas
		query = `SELECT r.id_receta, r.fecha, r.medicamento, r.id_medico, r.id_paciente, r.id_consultorio,
					 u_medico.nombre as medico_nombre, u_paciente.nombre as paciente_nombre, c.nombre_numero as consultorio_nombre
					 FROM Receta r
					 JOIN Usuario u_medico ON r.id_medico = u_medico.id_usuario
					 JOIN Usuario u_paciente ON r.id_paciente = u_paciente.id_usuario
					 JOIN Consultorio c ON r.id_consultorio = c.id_consultorio
					 ORDER BY r.fecha DESC`
	case "medico":
		// Médico solo ve sus propias recetas
		query = `SELECT r.id_receta, r.fecha, r.medicamento, r.id_medico, r.id_paciente, r.id_consultorio,
					 u_medico.nombre as medico_nombre, u_paciente.nombre as paciente_nombre, c.nombre_numero as consultorio_nombre
					 FROM Receta r
					 JOIN Usuario u_medico ON r.id_medico = u_medico.id_usuario
					 JOIN Usuario u_paciente ON r.id_paciente = u_paciente.id_usuario
					 JOIN Consultorio c ON r.id_consultorio = c.id_consultorio
					 WHERE r.id_medico = $1
					 ORDER BY r.fecha DESC`
		args = append(args, userID)
	case "paciente":
		// Paciente solo ve sus propias recetas
		query = `SELECT r.id_receta, r.fecha, r.medicamento, r.id_medico, r.id_paciente, r.id_consultorio,
					 u_medico.nombre as medico_nombre, u_paciente.nombre as paciente_nombre, c.nombre_numero as consultorio_nombre
					 FROM Receta r
					 JOIN Usuario u_medico ON r.id_medico = u_medico.id_usuario
					 JOIN Usuario u_paciente ON r.id_paciente = u_paciente.id_usuario
					 JOIN Consultorio c ON r.id_consultorio = c.id_consultorio
					 WHERE r.id_paciente = $1
					 ORDER BY r.fecha DESC`
		args = append(args, userID)
	case "enfermera":
		// Enfermera puede ver todas las recetas para administración
		query = `SELECT r.id_receta, r.fecha, r.medicamento, r.id_medico, r.id_paciente, r.id_consultorio,
					 u_medico.nombre as medico_nombre, u_paciente.nombre as paciente_nombre, c.nombre_numero as consultorio_nombre
					 FROM Receta r
					 JOIN Usuario u_medico ON r.id_medico = u_medico.id_usuario
					 JOIN Usuario u_paciente ON r.id_paciente = u_paciente.id_usuario
					 JOIN Consultorio c ON r.id_consultorio = c.id_consultorio
					 ORDER BY r.fecha DESC`
	default:
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F70",
				Data:    []interface{}{fiber.Map{"error": "No tienes permisos para ver recetas"}},
			},
		})
	}

	rows, err := database.GetDB().Query(context.Background(), query, args...)
	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F11",
				Data:    []interface{}{fiber.Map{"error": "Error al obtener recetas"}},
			},
		})
	}
	defer rows.Close()

	type RecetaDetalle struct {
		models.Receta
		MedicoNombre      string `json:"medico_nombre"`
		PacienteNombre    string `json:"paciente_nombre"`
		ConsultorioNombre string `json:"consultorio_nombre"`
	}

	var recetas []RecetaDetalle
	for rows.Next() {
		var receta RecetaDetalle
		err := rows.Scan(
			&receta.IDReceta, &receta.Fecha, &receta.Medicamento,
			&receta.IDMedico, &receta.IDPaciente, &receta.IDConsultorio,
			&receta.MedicoNombre, &receta.PacienteNombre, &receta.ConsultorioNombre,
		)
		if err != nil {
			continue
		}
		recetas = append(recetas, receta)
	}

	// Usar el formato StandardResponse consistente
	return c.Status(200).JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S01",
			Data: []interface{}{
				fiber.Map{
					"recetas": recetas,
					"total":   len(recetas),
				},
			},
		},
	})
}

// ObtenerRecetaPorID obtiene una receta específica por ID
func ObtenerRecetaPorID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	// Construir query según el rol (eliminar r.dosis)
	query := `SELECT r.id_receta, r.fecha, r.medicamento, r.id_medico, r.id_paciente, r.id_consultorio,
			  u_medico.nombre as medico_nombre, u_paciente.nombre as paciente_nombre, c.nombre_numero as consultorio_nombre
			  FROM Receta r
			  JOIN Usuario u_medico ON r.id_medico = u_medico.id_usuario
			  JOIN Usuario u_paciente ON r.id_paciente = u_paciente.id_usuario
			  JOIN Consultorio c ON r.id_consultorio = c.id_consultorio
			  WHERE r.id_receta = $1`

	var args []interface{}
	args = append(args, id)

	// Agregar filtros según el rol
	switch userRole {
	case "medico":
		query += " AND r.id_medico = $2"
		args = append(args, userID)
	case "paciente":
		query += " AND r.id_paciente = $2"
		args = append(args, userID)
	case "admin", "enfermera":
		// Pueden ver cualquier receta
	default:
		return c.Status(403).JSON(fiber.Map{
			"error": "No tienes permisos para ver esta receta",
		})
	}

	type RecetaDetalle struct {
		models.Receta
		MedicoNombre      string `json:"medico_nombre"`
		PacienteNombre    string `json:"paciente_nombre"`
		ConsultorioNombre string `json:"consultorio_nombre"`
	}

	var receta RecetaDetalle
	err = database.GetDB().QueryRow(context.Background(), query, args...).Scan(
		&receta.IDReceta, &receta.Fecha, &receta.Medicamento,
		&receta.IDMedico, &receta.IDPaciente, &receta.IDConsultorio,
		&receta.MedicoNombre, &receta.PacienteNombre, &receta.ConsultorioNombre,
	)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Receta no encontrada",
		})
	}

	return c.JSON(fiber.Map{
		"receta": receta,
	})
}

// ActualizarReceta actualiza una receta existente
func ActualizarReceta(c *fiber.Ctx) error {
	// Solo médicos pueden actualizar recetas
	userRole := c.Locals("user_role").(string)
	if userRole != "medico" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo médicos pueden actualizar recetas",
		})
	}

	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	medicoID := c.Locals("user_id").(int)

	// Verificar que la receta existe y pertenece al médico
	var recetaExistente models.Receta
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT id_receta, id_medico FROM Receta WHERE id_receta = $1 AND id_medico = $2",
		id, medicoID).Scan(&recetaExistente.IDReceta, &recetaExistente.IDMedico)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Receta no encontrada o no tienes permisos para modificarla",
		})
	}

	var recetaActualizada models.Receta
	if err := c.BodyParser(&recetaActualizada); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Validaciones (eliminar referencia a Dosis)
	if recetaActualizada.Medicamento == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Medicamento es requerido",
		})
	}

	// Actualizar receta (eliminar dosis)
	query := `UPDATE Receta SET medicamento = $1 WHERE id_receta = $2 AND id_medico = $3`

	_, err = database.GetDB().Exec(context.Background(), query,
		recetaActualizada.Medicamento, id, medicoID)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al actualizar la receta",
		})
	}

	// Log evento (eliminar dosis del log)
	middleware.LogCustomEvent(
		models.LogLevelInfo,
		"Receta actualizada",
		c.Locals("user_email").(string),
		userRole,
		map[string]interface{}{
			"receta_id":   id,
			"medico_id":   medicoID,
			"medicamento": recetaActualizada.Medicamento,
			"updated_by":  c.Locals("user_email"),
			"action":      "receta_updated",
		},
	)

	return c.JSON(fiber.Map{
		"mensaje": "Receta actualizada exitosamente",
	})
}

// EliminarReceta elimina una receta
func EliminarReceta(c *fiber.Ctx) error {
	// Solo médicos y admin pueden eliminar recetas
	userRole := c.Locals("user_role").(string)
	if userRole != "medico" && userRole != "admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo médicos y administradores pueden eliminar recetas",
		})
	}

	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	userID := c.Locals("user_id").(int)

	// Obtener información de la receta antes de eliminarla
	var medicoID, pacienteID, consultorioID int
	var medicamento string
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT id_medico, id_paciente, id_consultorio, medicamento FROM Receta WHERE id_receta = $1", id).Scan(&medicoID, &pacienteID, &consultorioID, &medicamento)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Receta no encontrada",
		})
	}

	// Verificar permisos
	var query string
	var args []interface{}

	if userRole == "admin" {
		// Admin puede eliminar cualquier receta
		query = "DELETE FROM Receta WHERE id_receta = $1"
		args = append(args, id)
	} else {
		// Médico solo puede eliminar sus propias recetas
		if medicoID != userID {
			return c.Status(403).JSON(fiber.Map{
				"error": "No tienes permisos para eliminar esta receta",
			})
		}
		query = "DELETE FROM Receta WHERE id_receta = $1 AND id_medico = $2"
		args = append(args, id, userID)
	}

	result, err := database.GetDB().Exec(context.Background(), query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al eliminar la receta",
		})
	}

	if result.RowsAffected() == 0 {
		return c.Status(404).JSON(fiber.Map{
			"error": "Receta no encontrada o no tienes permisos para eliminarla",
		})
	}

	// Log evento de eliminación de receta
	middleware.LogCustomEvent(
		models.LogLevelWarning,
		"Receta eliminada",
		c.Locals("user_email").(string),
		userRole,
		map[string]interface{}{
			"receta_id":      id,
			"medico_id":      medicoID,
			"paciente_id":    pacienteID,
			"consultorio_id": consultorioID,
			"medicamento":    medicamento,
			"deleted_by":     c.Locals("user_email"),
			"action":         "receta_deleted",
		},
	)

	return c.JSON(fiber.Map{
		"mensaje": "Receta eliminada exitosamente",
	})
}

// ObtenerRecetasPorPaciente obtiene todas las recetas de un paciente específico
func ObtenerRecetasPorPaciente(c *fiber.Ctx) error {
	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	pacienteIDParam := c.Params("paciente_id")
	pacienteID, err := strconv.Atoi(pacienteIDParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID de paciente inválido",
		})
	}

	// Verificar permisos
	switch userRole {
	case "paciente":
		// Paciente solo puede ver sus propias recetas
		if userID != pacienteID {
			return c.Status(403).JSON(fiber.Map{
				"error": "No tienes permisos para ver las recetas de otro paciente",
			})
		}
	case "admin", "medico", "enfermera":
		// Pueden ver recetas de cualquier paciente
	default:
		return c.Status(403).JSON(fiber.Map{
			"error": "No tienes permisos para ver recetas",
		})
	}

	query := `SELECT r.id_receta, r.fecha, r.medicamento, r.id_medico, r.id_paciente, r.id_consultorio,
			  u_medico.nombre as medico_nombre, u_paciente.nombre as paciente_nombre, c.nombre_numero as consultorio_nombre
			  FROM Receta r
			  JOIN Usuario u_medico ON r.id_medico = u_medico.id_usuario
			  JOIN Usuario u_paciente ON r.id_paciente = u_paciente.id_usuario
			  JOIN Consultorio c ON r.id_consultorio = c.id_consultorio
			  WHERE r.id_paciente = $1
			  ORDER BY r.fecha DESC`

	rows, err := database.GetDB().Query(context.Background(), query, pacienteID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener recetas del paciente",
		})
	}
	defer rows.Close()

	type RecetaDetalle struct {
		models.Receta
		MedicoNombre      string `json:"medico_nombre"`
		PacienteNombre    string `json:"paciente_nombre"`
		ConsultorioNombre string `json:"consultorio_nombre"`
	}

	var recetas []RecetaDetalle
	for rows.Next() {
		var receta RecetaDetalle
		err := rows.Scan(
			&receta.IDReceta, &receta.Fecha, &receta.Medicamento,
			&receta.IDMedico, &receta.IDPaciente, &receta.IDConsultorio,
			&receta.MedicoNombre, &receta.PacienteNombre, &receta.ConsultorioNombre,
		)
		if err != nil {
			continue
		}
		recetas = append(recetas, receta)
	}

	return c.JSON(fiber.Map{
		"recetas":     recetas,
		"total":       len(recetas),
		"paciente_id": pacienteID,
	})
}
