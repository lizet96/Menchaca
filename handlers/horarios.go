package handlers

import (
	"context"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/lizet96/hospital-backend/database"
	"github.com/lizet96/hospital-backend/middleware"
	"github.com/lizet96/hospital-backend/models"
)

// CrearHorario crea un nuevo horario médico
func CrearHorario(c *fiber.Ctx) error {
	// Solo admin puede crear horarios
	userRole := c.Locals("user_role").(string)
	if userRole != "admin" {
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F60",
				Data:    []interface{}{fiber.Map{"error": "Solo administradores pueden crear horarios"}},
			},
		})
	}

	var horario models.Horario
	if err := c.BodyParser(&horario); err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F60",
				Data:    []interface{}{fiber.Map{"error": "Datos inválidos"}},
			},
		})
	}

	// Validaciones
	if horario.Turno == "" || horario.IDMedico == 0 || horario.IDConsultorio == 0 {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F60",
				Data:    []interface{}{fiber.Map{"error": "Turno, médico y consultorio son requeridos"}},
			},
		})
	}

	// Verificar que el médico existe y tiene rol de médico
	var rolNombre string
	err := database.GetDB().QueryRow(context.Background(),
		`SELECT r.nombre FROM Usuario u 
		 JOIN Rol r ON u.id_rol = r.id_rol 
		 WHERE u.id_usuario = $1`, horario.IDMedico).Scan(&rolNombre)
	if err != nil {
		return c.Status(404).JSON(StandardResponse{
			StatusCode: 404,
			Body: BodyResponse{
				IntCode: "F60",
				Data:    []interface{}{fiber.Map{"error": "Médico no encontrado"}},
			},
		})
	}

	if rolNombre != "medico" {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F60",
				Data:    []interface{}{fiber.Map{"error": "El usuario especificado no es un médico"}},
			},
		})
	}

	// Verificar que el consultorio existe
	var consultorioExiste bool
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM Consultorio WHERE id_consultorio = $1)", horario.IDConsultorio).Scan(&consultorioExiste)
	if err != nil || !consultorioExiste {
		return c.Status(404).JSON(StandardResponse{
			StatusCode: 404,
			Body: BodyResponse{
				IntCode: "F60",
				Data:    []interface{}{fiber.Map{"error": "Consultorio no encontrado"}},
			},
		})
	}

	// Verificar que no exista un horario duplicado (mismo médico, consultorio y turno)
	var existeHorario bool
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM Horario WHERE id_medico = $1 AND id_consultorio = $2 AND turno = $3)",
		horario.IDMedico, horario.IDConsultorio, horario.Turno).Scan(&existeHorario)
	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F60",
				Data:    []interface{}{fiber.Map{"error": "Error al verificar horario"}},
			},
		})
	}

	if existeHorario {
		return c.Status(409).JSON(StandardResponse{
			StatusCode: 409,
			Body: BodyResponse{
				IntCode: "F60",
				Data:    []interface{}{fiber.Map{"error": "Ya existe un horario para este médico en este consultorio y turno"}},
			},
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
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F60",
				Data:    []interface{}{fiber.Map{"error": "Error al crear el horario"}},
			},
		})
	}

	// Log evento de creación de horario
	userEmail := ""
	if email := c.Locals("user_email"); email != nil {
		if emailStr, ok := email.(string); ok {
			userEmail = emailStr
		}
	}

	middleware.LogCustomEvent(
		models.LogLevelSuccess,
		"Horario creado",
		userEmail, // Usar la variable segura
		userRole,
		map[string]interface{}{
			"horario_id":     horario.IDHorario,
			"medico_id":      horario.IDMedico,
			"consultorio_id": horario.IDConsultorio,
			"turno":          horario.Turno,
			"created_by":     userEmail,
			"action":         "horario_created",
		},
	)

	return c.Status(201).JSON(StandardResponse{
		StatusCode: 201,
		Body: BodyResponse{
			IntCode: "S60",
			Data:    []interface{}{fiber.Map{"horario": horario, "mensaje": "Horario creado exitosamente"}},
		},
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
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F61",
				Data:    []interface{}{fiber.Map{"error": "No tienes permisos para ver horarios"}},
			},
		})
	}

	rows, err := database.GetDB().Query(context.Background(), query, args...)
	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F61",
				Data:    []interface{}{fiber.Map{"error": "Error al obtener horarios"}},
			},
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
			&horario.IDConsultorio, &horario.ConsultaDisponible,
			&horario.MedicoNombre, &horario.ConsultorioNombre,
		)
		if err != nil {
			log.Println("DEBUG - Error en scan:", err)
			return c.Status(400).JSON(StandardResponse{
				StatusCode: 400,
				Body: BodyResponse{
					IntCode: "F61",
					Data:    []interface{}{fiber.Map{"error": "Error al procesar horarios disponibles", "details": err.Error()}},
				},
			})
		}
		horarios = append(horarios, horario)
	}
	log.Println("DEBUG - Total horarios encontrados:", len(horarios))

	return c.JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S61",
			Data:    []interface{}{fiber.Map{"horarios": horarios, "total": len(horarios)}},
		},
	})
}

// ObtenerHorarioPorID obtiene un horario específico por ID
// ObtenerHorarioPorID - Línea 191
func ObtenerHorarioPorID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F61",
				Data:    []interface{}{fiber.Map{"error": "ID inválido"}},
			},
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
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F61",
				Data:    []interface{}{fiber.Map{"error": "No tienes permisos para ver este horario"}},
			},
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
		return c.Status(404).JSON(StandardResponse{
			StatusCode: 404,
			Body: BodyResponse{
				IntCode: "F61",
				Data:    []interface{}{fiber.Map{"error": "Horario no encontrado"}},
			},
		})
	}

	return c.JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S61",
			Data:    []interface{}{fiber.Map{"horario": horario}},
		},
	})
}

// ActualizarHorario actualiza un horario existente
func ActualizarHorario(c *fiber.Ctx) error {
	// Solo admin puede actualizar horarios
	userRole := c.Locals("user_role").(string)
	if userRole != "admin" {
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F62",
				Data:    []interface{}{fiber.Map{"error": "Solo administradores pueden actualizar horarios"}},
			},
		})
	}

	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F62",
				Data:    []interface{}{fiber.Map{"error": "ID inválido"}},
			},
		})
	}

	// Verificar que el horario existe
	var horarioExistente models.Horario
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT id_horario, id_medico, id_consultorio FROM Horario WHERE id_horario = $1", id).Scan(
		&horarioExistente.IDHorario, &horarioExistente.IDMedico, &horarioExistente.IDConsultorio)

	if err != nil {
		return c.Status(404).JSON(StandardResponse{
			StatusCode: 404,
			Body: BodyResponse{
				IntCode: "F62",
				Data:    []interface{}{fiber.Map{"error": "Horario no encontrado"}},
			},
		})
	}

	// En el método ActualizarHorario, después del BodyParser
	var horarioActualizado models.Horario
	if err := c.BodyParser(&horarioActualizado); err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F62",
				Data:    []interface{}{fiber.Map{"error": "Datos inválidos"}},
			},
		})
	}

	// ELIMINAR ESTA LÍNEA:
	// horarioActualizado.FechaHora = time.Time{}

	// Validaciones
	if horarioActualizado.Turno == "" {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F62",
				Data:    []interface{}{fiber.Map{"error": "El turno es requerido"}},
			},
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
			return c.Status(404).JSON(StandardResponse{
				StatusCode: 404,
				Body: BodyResponse{
					IntCode: "F62",
					Data:    []interface{}{fiber.Map{"error": "Médico no encontrado"}},
				},
			})
		}

		if rolNombre != "medico" {
			return c.Status(400).JSON(StandardResponse{
				StatusCode: 400,
				Body: BodyResponse{
					IntCode: "F62",
					Data:    []interface{}{fiber.Map{"error": "El usuario especificado no es un médico"}},
				},
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
			return c.Status(404).JSON(StandardResponse{
				StatusCode: 404,
				Body: BodyResponse{
					IntCode: "F62",
					Data:    []interface{}{fiber.Map{"error": "Consultorio no encontrado"}},
				},
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
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F62",
				Data:    []interface{}{fiber.Map{"error": "Error al verificar horario"}},
			},
		})
	}

	if existeOtroHorario {
		return c.Status(409).JSON(StandardResponse{
			StatusCode: 409,
			Body: BodyResponse{
				IntCode: "F62",
				Data:    []interface{}{fiber.Map{"error": "Ya existe otro horario para este médico en este consultorio y turno"}},
			},
		})
	}

	// Actualizar horario
	query := `UPDATE Horario SET turno = $1, id_medico = $2, id_consultorio = $3, consulta_disponible = $4 
			  WHERE id_horario = $5`

	_, err = database.GetDB().Exec(context.Background(), query,
		horarioActualizado.Turno, horarioActualizado.IDMedico, horarioActualizado.IDConsultorio,
		horarioActualizado.ConsultaDisponible, id)

	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F62",
				Data:    []interface{}{fiber.Map{"error": "Error al actualizar el horario"}},
			},
		})
	}

	// Log evento de actualización de horario
	// En la función ActualizarHorario, alrededor de la línea 500-510
	// Obtener email del usuario para logging
	userID := c.Locals("user_id").(int)
	var userEmail string
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT email FROM Usuario WHERE id_usuario = $1", userID).Scan(&userEmail)
	if err != nil {
		userEmail = "unknown" // Fallback si no se puede obtener el email
	}

	// Log de la acción
	middleware.LogCustomEvent(
		models.LogLevelInfo,
		"Horario actualizado",
		userEmail, // Usar la variable userEmail obtenida
		userRole,
		map[string]interface{}{
			"horario_id":     id,
			"medico_id":      horarioActualizado.IDMedico,
			"consultorio_id": horarioActualizado.IDConsultorio,
			"turno":          horarioActualizado.Turno,
			"updated_by":     userEmail, // Usar la variable userEmail
			"action":         "horario_updated",
		},
	)

	return c.JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S62",
			Data:    []interface{}{fiber.Map{"mensaje": "Horario actualizado exitosamente"}},
		},
	})
}

// Línea 363 - En EliminarHorario
func EliminarHorario(c *fiber.Ctx) error {
	// Solo admin puede eliminar horarios
	userRole := c.Locals("user_role").(string)
	if userRole != "admin" {
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F62",
				Data:    []interface{}{fiber.Map{"error": "Solo administradores pueden eliminar horarios"}},
			},
		})
	}

	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F62",
				Data:    []interface{}{fiber.Map{"error": "ID inválido"}},
			},
		})
	}

	// Verificar si el horario tiene consultas asociadas
	var tieneConsultas bool
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM Consulta WHERE id_horario = $1)", id).Scan(&tieneConsultas)
	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F62",
				Data:    []interface{}{fiber.Map{"error": "Error al verificar consultas asociadas"}},
			},
		})
	}

	if tieneConsultas {
		return c.Status(409).JSON(StandardResponse{
			StatusCode: 409,
			Body: BodyResponse{
				IntCode: "F62",
				Data:    []interface{}{fiber.Map{"error": "No se puede eliminar el horario porque tiene consultas asociadas"}},
			},
		})
	}

	// Obtener información del horario antes de eliminarlo
	var medicoID, consultorioID int
	var turno string
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT id_medico, id_consultorio, turno FROM Horario WHERE id_horario = $1", id).Scan(&medicoID, &consultorioID, &turno)
	if err != nil {
		return c.Status(404).JSON(StandardResponse{
			StatusCode: 404,
			Body: BodyResponse{
				IntCode: "F62",
				Data:    []interface{}{fiber.Map{"error": "Horario no encontrado"}},
			},
		})
	}

	// Eliminar horario
	result, err := database.GetDB().Exec(context.Background(),
		"DELETE FROM Horario WHERE id_horario = $1", id)
	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F62",
				Data:    []interface{}{fiber.Map{"error": "Error al eliminar el horario"}},
			},
		})
	}

	if result.RowsAffected() == 0 {
		return c.Status(404).JSON(StandardResponse{
			StatusCode: 404,
			Body: BodyResponse{
				IntCode: "F62",
				Data:    []interface{}{fiber.Map{"error": "Horario no encontrado"}},
			},
		})
	}

	// En la función EliminarHorario, después de obtener la información del horario
	// y antes del logging (alrededor de la línea 620)

	// Obtener email del usuario para logging
	userID := c.Locals("user_id").(int)
	var userEmail string
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT email FROM Usuario WHERE id_usuario = $1", userID).Scan(&userEmail)
	if err != nil {
		userEmail = "unknown" // Fallback si no se puede obtener el email
	}

	// Log evento de eliminación de horario
	middleware.LogCustomEvent(
		models.LogLevelWarning,
		"Horario eliminado",
		userEmail, // Usar la variable userEmail obtenida
		userRole,
		map[string]interface{}{
			"horario_id":     id,
			"medico_id":      medicoID,
			"consultorio_id": consultorioID,
			"turno":          turno,
			"deleted_by":     userEmail, // Usar la variable userEmail
			"action":         "horario_deleted",
		},
	)

	return c.JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S62",
			Data:    []interface{}{fiber.Map{"mensaje": "Horario eliminado exitosamente"}},
		},
	})
}

// CambiarDisponibilidadHorario - Línea 433
func CambiarDisponibilidadHorario(c *fiber.Ctx) error {
	// Admin y médicos pueden cambiar disponibilidad
	userRole := c.Locals("user_role").(string)
	if userRole != "admin" && userRole != "medico" {
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F62",
				Data:    []interface{}{fiber.Map{"error": "Solo administradores y médicos pueden cambiar la disponibilidad"}},
			},
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
	log.Println("DEBUG - *** INICIO ObtenerHorariosDisponibles ***")
	log.Printf("DEBUG - Method: %s, Path: %s", c.Method(), c.Path())
	log.Printf("DEBUG - Headers: %v", c.GetReqHeaders())

	// Verificar usuario y permisos
	userID := c.Locals("user_id")
	userRole := c.Locals("user_role")
	log.Printf("DEBUG - UserID: %v, UserRole: %v", userID, userRole)

	// Verificar si hay query parameters que puedan estar causando problemas
	log.Printf("DEBUG - Query params: %s", c.Request().URI().QueryString())

	// Obtener horarios disponibles para citas
	query := `SELECT h.id_horario, h.turno, h.id_medico, h.id_consultorio, h.consulta_disponible,
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
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F10",
				Data:    []interface{}{fiber.Map{"error": "Error al obtener horarios disponibles", "details": err.Error()}},
			},
		})
	}
	defer rows.Close()

	log.Println("DEBUG - Query ejecutada exitosamente, procesando resultados...")

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
			log.Printf("DEBUG - Error escaneando fila: %v", err)
			continue
		}
		horarios = append(horarios, horario)
	}

	log.Printf("DEBUG - Horarios encontrados: %d", len(horarios))
	log.Printf("DEBUG - Enviando respuesta exitosa")

	return c.JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S10",
			Data:    []interface{}{fiber.Map{"horarios": horarios, "total": len(horarios)}},
		},
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
