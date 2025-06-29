package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lizet96/hospital-backend/database"
	"github.com/lizet96/hospital-backend/models"
)

// CrearReceta crea una nueva receta médica
func CrearReceta(c *fiber.Ctx) error {
	// Solo médicos pueden crear recetas
	userType := c.Locals("user_type").(string)
	if userType != "medico" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo médicos pueden crear recetas",
		})
	}

	medicoID := c.Locals("user_id").(int)

	var receta models.Receta
	if err := c.BodyParser(&receta); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Validaciones
	if receta.Medicamento == "" || receta.Dosis == "" || receta.IDPaciente == 0 || receta.IDConsultorio == 0 {
		return c.Status(400).JSON(fiber.Map{
			"error": "Medicamento, dosis, paciente y consultorio son requeridos",
		})
	}

	// Verificar que el paciente existe
	var pacienteTipo string
	err := database.GetDB().QueryRow(context.Background(),
		"SELECT tipo FROM Usuario WHERE id_usuario = $1", receta.IDPaciente).Scan(&pacienteTipo)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Paciente no encontrado",
		})
	}

	if pacienteTipo != "paciente" {
		return c.Status(400).JSON(fiber.Map{
			"error": "El usuario especificado no es un paciente",
		})
	}

	// Verificar que el consultorio existe
	var consultorioExiste bool
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM Consultorio WHERE id_consultorio = $1)", receta.IDConsultorio).Scan(&consultorioExiste)
	if err != nil || !consultorioExiste {
		return c.Status(404).JSON(fiber.Map{
			"error": "Consultorio no encontrado",
		})
	}

	// Establecer fecha actual si no se proporciona
	if receta.Fecha.IsZero() {
		receta.Fecha = time.Now()
	}

	// Insertar receta
	query := `INSERT INTO Receta (fecha, medicamento, dosis, id_medico, id_paciente, id_consultorio) 
			  VALUES ($1, $2, $3, $4, $5, $6) RETURNING id_receta`

	err = database.GetDB().QueryRow(context.Background(), query,
		receta.Fecha, receta.Medicamento, receta.Dosis, medicoID, receta.IDPaciente, receta.IDConsultorio).Scan(&receta.IDReceta)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al crear la receta",
		})
	}

	receta.IDMedico = medicoID

	return c.Status(201).JSON(fiber.Map{
		"receta":  receta,
		"mensaje": "Receta creada exitosamente",
	})
}

// ObtenerRecetas obtiene todas las recetas (con filtros según el rol)
func ObtenerRecetas(c *fiber.Ctx) error {
	userType := c.Locals("user_type").(string)
	userID := c.Locals("user_id").(int)

	var query string
	var args []interface{}

	switch userType {
	case "admin":
		// Admin puede ver todas las recetas
		query = `SELECT r.id_receta, r.fecha, r.medicamento, r.dosis, r.id_medico, r.id_paciente, r.id_consultorio,
					 u_medico.nombre as medico_nombre, u_paciente.nombre as paciente_nombre, c.nombre_numero as consultorio_nombre
					 FROM Receta r
					 JOIN Usuario u_medico ON r.id_medico = u_medico.id_usuario
					 JOIN Usuario u_paciente ON r.id_paciente = u_paciente.id_usuario
					 JOIN Consultorio c ON r.id_consultorio = c.id_consultorio
					 ORDER BY r.fecha DESC`
	case "medico":
		// Médico solo ve sus propias recetas
		query = `SELECT r.id_receta, r.fecha, r.medicamento, r.dosis, r.id_medico, r.id_paciente, r.id_consultorio,
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
		query = `SELECT r.id_receta, r.fecha, r.medicamento, r.dosis, r.id_medico, r.id_paciente, r.id_consultorio,
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
		query = `SELECT r.id_receta, r.fecha, r.medicamento, r.dosis, r.id_medico, r.id_paciente, r.id_consultorio,
					 u_medico.nombre as medico_nombre, u_paciente.nombre as paciente_nombre, c.nombre_numero as consultorio_nombre
					 FROM Receta r
					 JOIN Usuario u_medico ON r.id_medico = u_medico.id_usuario
					 JOIN Usuario u_paciente ON r.id_paciente = u_paciente.id_usuario
					 JOIN Consultorio c ON r.id_consultorio = c.id_consultorio
					 ORDER BY r.fecha DESC`
	default:
		return c.Status(403).JSON(fiber.Map{
			"error": "No tienes permisos para ver recetas",
		})
	}

	rows, err := database.GetDB().Query(context.Background(), query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener recetas",
		})
	}
	defer rows.Close()

	type RecetaDetalle struct {
		models.Receta
		MedicoNombre     string `json:"medico_nombre"`
		PacienteNombre   string `json:"paciente_nombre"`
		ConsultorioNombre string `json:"consultorio_nombre"`
	}

	var recetas []RecetaDetalle
	for rows.Next() {
		var receta RecetaDetalle
		err := rows.Scan(
			&receta.IDReceta, &receta.Fecha, &receta.Medicamento, &receta.Dosis,
			&receta.IDMedico, &receta.IDPaciente, &receta.IDConsultorio,
			&receta.MedicoNombre, &receta.PacienteNombre, &receta.ConsultorioNombre,
		)
		if err != nil {
			continue
		}
		recetas = append(recetas, receta)
	}

	return c.JSON(fiber.Map{
		"recetas": recetas,
		"total":   len(recetas),
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

	userType := c.Locals("user_type").(string)
	userID := c.Locals("user_id").(int)

	// Construir query según el rol
	query := `SELECT r.id_receta, r.fecha, r.medicamento, r.dosis, r.id_medico, r.id_paciente, r.id_consultorio,
			  u_medico.nombre as medico_nombre, u_paciente.nombre as paciente_nombre, c.nombre_numero as consultorio_nombre
			  FROM Receta r
			  JOIN Usuario u_medico ON r.id_medico = u_medico.id_usuario
			  JOIN Usuario u_paciente ON r.id_paciente = u_paciente.id_usuario
			  JOIN Consultorio c ON r.id_consultorio = c.id_consultorio
			  WHERE r.id_receta = $1`

	var args []interface{}
	args = append(args, id)

	// Agregar filtros según el rol
	switch userType {
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
		MedicoNombre     string `json:"medico_nombre"`
		PacienteNombre   string `json:"paciente_nombre"`
		ConsultorioNombre string `json:"consultorio_nombre"`
	}

	var receta RecetaDetalle
	err = database.GetDB().QueryRow(context.Background(), query, args...).Scan(
		&receta.IDReceta, &receta.Fecha, &receta.Medicamento, &receta.Dosis,
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
	userType := c.Locals("user_type").(string)
	if userType != "medico" {
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

	// Validaciones
	if recetaActualizada.Medicamento == "" || recetaActualizada.Dosis == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Medicamento y dosis son requeridos",
		})
	}

	// Actualizar receta
	query := `UPDATE Receta SET medicamento = $1, dosis = $2 WHERE id_receta = $3 AND id_medico = $4`

	_, err = database.GetDB().Exec(context.Background(), query,
		recetaActualizada.Medicamento, recetaActualizada.Dosis, id, medicoID)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al actualizar la receta",
		})
	}

	return c.JSON(fiber.Map{
		"mensaje": "Receta actualizada exitosamente",
	})
}

// EliminarReceta elimina una receta
func EliminarReceta(c *fiber.Ctx) error {
	// Solo médicos y admin pueden eliminar recetas
	userType := c.Locals("user_type").(string)
	if userType != "medico" && userType != "admin" {
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

	// Verificar permisos
	var query string
	var args []interface{}

	if userType == "admin" {
		// Admin puede eliminar cualquier receta
		query = "DELETE FROM Receta WHERE id_receta = $1"
		args = append(args, id)
	} else {
		// Médico solo puede eliminar sus propias recetas
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

	return c.JSON(fiber.Map{
		"mensaje": "Receta eliminada exitosamente",
	})
}

// ObtenerRecetasPorPaciente obtiene todas las recetas de un paciente específico
func ObtenerRecetasPorPaciente(c *fiber.Ctx) error {
	userType := c.Locals("user_type").(string)
	userID := c.Locals("user_id").(int)

	pacienteIDParam := c.Params("paciente_id")
	pacienteID, err := strconv.Atoi(pacienteIDParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID de paciente inválido",
		})
	}

	// Verificar permisos
	switch userType {
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

	query := `SELECT r.id_receta, r.fecha, r.medicamento, r.dosis, r.id_medico, r.id_paciente, r.id_consultorio,
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
		MedicoNombre     string `json:"medico_nombre"`
		PacienteNombre   string `json:"paciente_nombre"`
		ConsultorioNombre string `json:"consultorio_nombre"`
	}

	var recetas []RecetaDetalle
	for rows.Next() {
		var receta RecetaDetalle
		err := rows.Scan(
			&receta.IDReceta, &receta.Fecha, &receta.Medicamento, &receta.Dosis,
			&receta.IDMedico, &receta.IDPaciente, &receta.IDConsultorio,
			&receta.MedicoNombre, &receta.PacienteNombre, &receta.ConsultorioNombre,
		)
		if err != nil {
			continue
		}
		recetas = append(recetas, receta)
	}

	return c.JSON(fiber.Map{
		"recetas": recetas,
		"total":   len(recetas),
		"paciente_id": pacienteID,
	})
}