package handlers

import (
	"context"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/lizet96/hospital-backend/database"
	"github.com/lizet96/hospital-backend/models"
)

// CrearHorario crea un nuevo horario médico
func CrearHorario(c *fiber.Ctx) error {
	// Solo admin puede crear horarios
	userRole := c.Locals("user_role").(string)
	if userRole != "admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo administradores pueden crear horarios",
		})
	}

	var horario models.Horario
	if err := c.BodyParser(&horario); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Validaciones
	if horario.Turno == "" || horario.IDMedico == 0 || horario.IDConsultorio == 0 {
		return c.Status(400).JSON(fiber.Map{
			"error": "Turno, médico y consultorio son requeridos",
		})
	}

	// Verificar que el médico existe y tiene rol de médico
	var rolNombre string
	err := database.GetDB().QueryRow(context.Background(),
		`SELECT r.nombre FROM Usuario u 
		 JOIN Rol r ON u.id_rol = r.id_rol 
		 WHERE u.id_usuario = $1`, horario.IDMedico).Scan(&rolNombre)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Médico no encontrado",
		})
	}

	if rolNombre != "medico" {
		return c.Status(400).JSON(fiber.Map{
			"error": "El usuario especificado no es un médico",
		})
	}

	// Verificar que el consultorio existe
	var consultorioExiste bool
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM Consultorio WHERE id_consultorio = $1)", horario.IDConsultorio).Scan(&consultorioExiste)
	if err != nil || !consultorioExiste {
		return c.Status(404).JSON(fiber.Map{
			"error": "Consultorio no encontrado",
		})
	}

	// Verificar que no exista un horario duplicado (mismo médico, consultorio y turno)
	var existeHorario bool
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM Horario WHERE id_medico = $1 AND id_consultorio = $2 AND turno = $3)",
		horario.IDMedico, horario.IDConsultorio, horario.Turno).Scan(&existeHorario)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al verificar horario",
		})
	}

	if existeHorario {
		return c.Status(409).JSON(fiber.Map{
			"error": "Ya existe un horario para este médico en este consultorio y turno",
		})
	}

	// Establecer disponibilidad por defecto
	if !horario.ConsultaDisponible {
		horario.ConsultaDisponible = true
	}

	// Insertar horario
	query := `INSERT INTO Horario (turno, id_medico, id_consultorio, consulta_disponible) 
			  VALUES ($1, $2, $3, $4) RETURNING id_horario`

	err = database.GetDB().QueryRow(context.Background(), query,
		horario.Turno, horario.IDMedico, horario.IDConsultorio, horario.ConsultaDisponible).Scan(&horario.IDHorario)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al crear el horario",
		})
	}

	return c.Status(201).JSON(fiber.Map{
		"horario": horario,
		"mensaje": "Horario creado exitosamente",
	})
}

// ObtenerHorarios obtiene todos los horarios (con filtros según el rol)
func ObtenerHorarios(c *fiber.Ctx) error {
	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	var query string
	var args []interface{}

	switch userRole {
	case "admin", "enfermera":
		// Admin y enfermeras pueden ver todos los horarios
		query = `SELECT h.id_horario, h.turno, h.id_medico, h.id_consultorio, h.consulta_disponible,
					 u.nombre as medico_nombre, c.nombre_numero as consultorio_nombre
					 FROM Horario h
					 JOIN Usuario u ON h.id_medico = u.id_usuario
					 JOIN Consultorio c ON h.id_consultorio = c.id_consultorio
					 ORDER BY h.turno, u.nombre`
	case "medico":
		// Médico solo ve sus propios horarios
		query = `SELECT h.id_horario, h.turno, h.id_medico, h.id_consultorio, h.consulta_disponible,
					 u.nombre as medico_nombre, c.nombre_numero as consultorio_nombre
					 FROM Horario h
					 JOIN Usuario u ON h.id_medico = u.id_usuario
					 JOIN Consultorio c ON h.id_consultorio = c.id_consultorio
					 WHERE h.id_medico = $1
					 ORDER BY h.turno`
		args = append(args, userID)
	case "paciente":
		// Pacientes solo ven horarios disponibles
		query = `SELECT h.id_horario, h.turno, h.id_medico, h.id_consultorio, h.consulta_disponible,
					 u.nombre as medico_nombre, c.nombre_numero as consultorio_nombre
					 FROM Horario h
					 JOIN Usuario u ON h.id_medico = u.id_usuario
					 JOIN Consultorio c ON h.id_consultorio = c.id_consultorio
					 WHERE h.consulta_disponible = true
					 ORDER BY h.turno, u.nombre`
	default:
		return c.Status(403).JSON(fiber.Map{
			"error": "No tienes permisos para ver horarios",
		})
	}

	rows, err := database.GetDB().Query(context.Background(), query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener horarios",
		})
	}
	defer rows.Close()

	type HorarioDetalle struct {
		models.Horario
		MedicoNombre      string `json:"medico_nombre"`
		ConsultorioNombre string `json:"consultorio_nombre"`
	}

	var horarios []HorarioDetalle
	log.Println("DEBUG - Iniciando escaneo de filas")
	for rows.Next() {
		var horario HorarioDetalle
		err := rows.Scan(
			&horario.IDHorario, &horario.Turno, &horario.IDMedico,
			&horario.IDConsultorio, &horario.ConsultaDisponible, &horario.FechaHora,
			&horario.MedicoNombre, &horario.ConsultorioNombre,
		)
		if err != nil {
			log.Println("DEBUG - Error en scan:", err)
			return c.Status(400).JSON(fiber.Map{
				"error": "Error al procesar horarios disponibles",
				"details": err.Error(),
			})
		}
		horarios = append(horarios, horario)
	}
	log.Println("DEBUG - Total horarios encontrados:", len(horarios))

	return c.JSON(fiber.Map{
		"horarios": horarios,
		"total":    len(horarios),
	})
}

// ObtenerHorarioPorID obtiene un horario específico por ID
// ObtenerHorarioPorID - Línea 191
func ObtenerHorarioPorID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	// Construir query según el rol
	query := `SELECT h.id_horario, h.turno, h.id_medico, h.id_consultorio, h.consulta_disponible,
			  u.nombre as medico_nombre, c.nombre_numero as consultorio_nombre
			  FROM Horario h
			  JOIN Usuario u ON h.id_medico = u.id_usuario
			  JOIN Consultorio c ON h.id_consultorio = c.id_consultorio
			  WHERE h.id_horario = $1`

	var args []interface{}
	args = append(args, id)

	// Agregar filtros según el rol
	switch userRole {
	case "medico":
		query += " AND h.id_medico = $2"
		args = append(args, userID)
	case "paciente":
		query += " AND h.consulta_disponible = true"
	case "admin", "enfermera":
		// Pueden ver cualquier horario
	default:
		return c.Status(403).JSON(fiber.Map{
			"error": "No tienes permisos para ver este horario",
		})
	}

	type HorarioDetalle struct {
		models.Horario
		MedicoNombre      string `json:"medico_nombre"`
		ConsultorioNombre string `json:"consultorio_nombre"`
	}

	var horario HorarioDetalle
	err = database.GetDB().QueryRow(context.Background(), query, args...).Scan(
		&horario.IDHorario, &horario.Turno, &horario.IDMedico,
		&horario.IDConsultorio, &horario.ConsultaDisponible,
		&horario.MedicoNombre, &horario.ConsultorioNombre,
	)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Horario no encontrado",
		})
	}

	return c.JSON(fiber.Map{
		"horario": horario,
	})
}

// ActualizarHorario actualiza un horario existente
func ActualizarHorario(c *fiber.Ctx) error {
	// Solo admin puede actualizar horarios
	userRole := c.Locals("user_role").(string)
	if userRole != "admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo administradores pueden actualizar horarios",
		})
	}

	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Verificar que el horario existe
	var horarioExistente models.Horario
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT id_horario, id_medico, id_consultorio FROM Horario WHERE id_horario = $1", id).Scan(
		&horarioExistente.IDHorario, &horarioExistente.IDMedico, &horarioExistente.IDConsultorio)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Horario no encontrado",
		})
	}

	var horarioActualizado models.Horario
	if err := c.BodyParser(&horarioActualizado); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Validaciones
	if horarioActualizado.Turno == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "El turno es requerido",
		})
	}

	// Si se cambia el médico, verificar que existe y es médico
	if horarioActualizado.IDMedico != 0 && horarioActualizado.IDMedico != horarioExistente.IDMedico {
		var rolNombre string
		err := database.GetDB().QueryRow(context.Background(),
			`SELECT r.nombre FROM Usuario u 
			 JOIN Rol r ON u.id_rol = r.id_rol 
			 WHERE u.id_usuario = $1`, horarioActualizado.IDMedico).Scan(&rolNombre)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{
				"error": "Médico no encontrado",
			})
		}

		if rolNombre != "medico" {
			return c.Status(400).JSON(fiber.Map{
				"error": "El usuario especificado no es un médico",
			})
		}
	} else {
		horarioActualizado.IDMedico = horarioExistente.IDMedico
	}

	// Si se cambia el consultorio, verificar que existe
	if horarioActualizado.IDConsultorio != 0 && horarioActualizado.IDConsultorio != horarioExistente.IDConsultorio {
		var consultorioExiste bool
		err = database.GetDB().QueryRow(context.Background(),
			"SELECT EXISTS(SELECT 1 FROM Consultorio WHERE id_consultorio = $1)", horarioActualizado.IDConsultorio).Scan(&consultorioExiste)
		if err != nil || !consultorioExiste {
			return c.Status(404).JSON(fiber.Map{
				"error": "Consultorio no encontrado",
			})
		}
	} else {
		horarioActualizado.IDConsultorio = horarioExistente.IDConsultorio
	}

	// Verificar que no exista un horario duplicado
	var existeOtroHorario bool
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM Horario WHERE id_medico = $1 AND id_consultorio = $2 AND turno = $3 AND id_horario != $4)",
		horarioActualizado.IDMedico, horarioActualizado.IDConsultorio, horarioActualizado.Turno, id).Scan(&existeOtroHorario)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al verificar horario",
		})
	}

	if existeOtroHorario {
		return c.Status(409).JSON(fiber.Map{
			"error": "Ya existe otro horario para este médico en este consultorio y turno",
		})
	}

	// Actualizar horario
	query := `UPDATE Horario SET turno = $1, id_medico = $2, id_consultorio = $3, consulta_disponible = $4 
			  WHERE id_horario = $5`

	_, err = database.GetDB().Exec(context.Background(), query,
		horarioActualizado.Turno, horarioActualizado.IDMedico, horarioActualizado.IDConsultorio,
		horarioActualizado.ConsultaDisponible, id)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al actualizar el horario",
		})
	}

	return c.JSON(fiber.Map{
		"mensaje": "Horario actualizado exitosamente",
	})
}

// Línea 363 - En EliminarHorario
func EliminarHorario(c *fiber.Ctx) error {
	// Solo admin puede eliminar horarios
	userRole := c.Locals("user_role").(string)
	if userRole != "admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo administradores pueden eliminar horarios",
		})
	}

	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Verificar si el horario tiene consultas asociadas
	var tieneConsultas bool
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM Consulta WHERE id_horario = $1)", id).Scan(&tieneConsultas)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al verificar consultas asociadas",
		})
	}

	if tieneConsultas {
		return c.Status(409).JSON(fiber.Map{
			"error": "No se puede eliminar el horario porque tiene consultas asociadas",
		})
	}

	// Eliminar horario
	result, err := database.GetDB().Exec(context.Background(),
		"DELETE FROM Horario WHERE id_horario = $1", id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al eliminar el horario",
		})
	}

	if result.RowsAffected() == 0 {
		return c.Status(404).JSON(fiber.Map{
			"error": "Horario no encontrado",
		})
	}

	return c.JSON(fiber.Map{
		"mensaje": "Horario eliminado exitosamente",
	})
}

// CambiarDisponibilidadHorario - Línea 433
func CambiarDisponibilidadHorario(c *fiber.Ctx) error {
	// Admin y médicos pueden cambiar disponibilidad
	userRole := c.Locals("user_role").(string)
	if userRole != "admin" && userRole != "medico" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo administradores y médicos pueden cambiar la disponibilidad",
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

	if userRole == "admin" {
		// Admin puede cambiar cualquier horario
		query = "SELECT id_horario, consulta_disponible FROM Horario WHERE id_horario = $1"
		args = append(args, id)
	} else {
		// Médico solo puede cambiar sus propios horarios
		query = "SELECT id_horario, consulta_disponible FROM Horario WHERE id_horario = $1 AND id_medico = $2"
		args = append(args, id, userID)
	}

	var horarioID int
	var disponibilidadActual bool
	err = database.GetDB().QueryRow(context.Background(), query, args...).Scan(&horarioID, &disponibilidadActual)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Horario no encontrado o no tienes permisos para modificarlo",
		})
	}

	// Obtener nueva disponibilidad del body
	type DisponibilidadRequest struct {
		Disponible bool `json:"disponible"`
	}

	var req DisponibilidadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Actualizar disponibilidad
	_, err = database.GetDB().Exec(context.Background(),
		"UPDATE Horario SET consulta_disponible = $1 WHERE id_horario = $2",
		req.Disponible, id)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al actualizar la disponibilidad",
		})
	}

	var mensaje string
	if req.Disponible {
		mensaje = "Horario marcado como disponible"
	} else {
		mensaje = "Horario marcado como no disponible"
	}

	return c.JSON(fiber.Map{
		"mensaje":                 mensaje,
		"disponibilidad_anterior": disponibilidadActual,
		"disponibilidad_nueva":    req.Disponible,
	})
}

// ObtenerHorariosDisponibles obtiene solo los horarios disponibles
func ObtenerHorariosDisponibles(c *fiber.Ctx) error {
	// Debug: Log que se está ejecutando la función
	log.Println("DEBUG - Ejecutando ObtenerHorariosDisponibles")
	
	// Obtener horarios disponibles para citas
	query := `SELECT h.id_horario, h.turno, h.id_medico, h.id_consultorio, h.consulta_disponible, h.fecha_hora,
			  u.nombre as medico_nombre, c.nombre_numero as consultorio_nombre
			  FROM Horario h
			  JOIN Usuario u ON h.id_medico = u.id_usuario
			  JOIN Consultorio c ON h.id_consultorio = c.id_consultorio
			  WHERE h.consulta_disponible = true
			  ORDER BY h.turno, u.nombre`

	log.Println("DEBUG - Query:", query)
	
	rows, err := database.GetDB().Query(context.Background(), query)
	if err != nil {
		log.Println("DEBUG - Error en query:", err)
		return c.Status(400).JSON(fiber.Map{
			"error": "Error al obtener horarios disponibles",
			"details": err.Error(),
		})
	}
	defer rows.Close()

	type HorarioDetalle struct {
		models.Horario
		MedicoNombre      string `json:"medico_nombre"`
		ConsultorioNombre string `json:"consultorio_nombre"`
	}

	var horarios []HorarioDetalle
	for rows.Next() {
		var horario HorarioDetalle
		err := rows.Scan(
			&horario.IDHorario, &horario.Turno, &horario.IDMedico,
			&horario.IDConsultorio, &horario.ConsultaDisponible,
			&horario.MedicoNombre, &horario.ConsultorioNombre,
		)
		if err != nil {
			continue
		}
		horarios = append(horarios, horario)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    horarios,
		"total":   len(horarios),
	})
}

// ObtenerHorariosPorMedico - Línea 550
func ObtenerHorariosPorMedico(c *fiber.Ctx) error {
	medicoIDParam := c.Params("medico_id")
	medicoID, err := strconv.Atoi(medicoIDParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID de médico inválido",
		})
	}

	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	// Verificar permisos
	switch userRole {
	case "medico":
		// Médico solo puede ver sus propios horarios
		if userID != medicoID {
			return c.Status(403).JSON(fiber.Map{
				"error": "No tienes permisos para ver los horarios de otro médico",
			})
		}
	case "admin", "enfermera":
		// Pueden ver horarios de cualquier médico
	case "paciente":
		// Pacientes solo ven horarios disponibles
	default:
		return c.Status(403).JSON(fiber.Map{
			"error": "No tienes permisos para ver horarios",
		})
	}

	// Verificar que el médico existe y tiene rol de médico
	var rolNombre string
	err = database.GetDB().QueryRow(context.Background(),
		`SELECT r.nombre FROM Usuario u 
		 JOIN Rol r ON u.id_rol = r.id_rol 
		 WHERE u.id_usuario = $1`, medicoID).Scan(&rolNombre)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Médico no encontrado",
		})
	}

	if rolNombre != "medico" {
		return c.Status(400).JSON(fiber.Map{
			"error": "El usuario especificado no es un médico",
		})
	}

	// Construir query según el rol
	var query string
	if userRole == "paciente" {
		query = `SELECT h.id_horario, h.turno, h.id_medico, h.id_consultorio, h.consulta_disponible,
				 u.nombre as medico_nombre, c.nombre_numero as consultorio_nombre
				 FROM Horario h
				 JOIN Usuario u ON h.id_medico = u.id_usuario
				 JOIN Consultorio c ON h.id_consultorio = c.id_consultorio
				 WHERE h.id_medico = $1 AND h.consulta_disponible = true
				 ORDER BY h.turno`
	} else {
		query = `SELECT h.id_horario, h.turno, h.id_medico, h.id_consultorio, h.consulta_disponible,
				 u.nombre as medico_nombre, c.nombre_numero as consultorio_nombre
				 FROM Horario h
				 JOIN Usuario u ON h.id_medico = u.id_usuario
				 JOIN Consultorio c ON h.id_consultorio = c.id_consultorio
				 WHERE h.id_medico = $1
				 ORDER BY h.turno`
	}

	rows, err := database.GetDB().Query(context.Background(), query, medicoID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener horarios del médico",
		})
	}
	defer rows.Close()

	type HorarioDetalle struct {
		models.Horario
		MedicoNombre      string `json:"medico_nombre"`
		ConsultorioNombre string `json:"consultorio_nombre"`
	}

	var horarios []HorarioDetalle
	for rows.Next() {
		var horario HorarioDetalle
		err := rows.Scan(
			&horario.IDHorario, &horario.Turno, &horario.IDMedico,
			&horario.IDConsultorio, &horario.ConsultaDisponible,
			&horario.MedicoNombre, &horario.ConsultorioNombre,
		)
		if err != nil {
			continue
		}
		horarios = append(horarios, horario)
	}

	return c.JSON(fiber.Map{
		"horarios":  horarios,
		"total":     len(horarios),
		"medico_id": medicoID,
	})
}
