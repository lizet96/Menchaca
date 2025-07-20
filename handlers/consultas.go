package handlers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/lizet96/hospital-backend/database"
	"github.com/lizet96/hospital-backend/middleware"
	"github.com/lizet96/hospital-backend/models"
)

// CrearConsulta crea una nueva consulta médica
func CrearConsulta(c *fiber.Ctx) error {
	var consulta models.Consulta
	if err := c.BodyParser(&consulta); err != nil {
		fmt.Printf("DEBUG - Error parsing body: %v\n", err)
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F10",
				Data:    []interface{}{fiber.Map{"error": "Datos inválidos"}},
			},
		})
	}

	// Debug: Log de los datos recibidos
	fmt.Printf("DEBUG - Datos recibidos: %+v\n", consulta)
	fmt.Printf("DEBUG - Tipo: '%s', Diagnostico: '%s', Costo: %f\n", consulta.Tipo, consulta.Diagnostico, consulta.Costo)
	fmt.Printf("DEBUG - IDPaciente: %d, IDMedico: %d, IDHorario: %v\n", consulta.IDPaciente, consulta.IDMedico, consulta.IDHorario)

	// NUEVO: Debug adicional para identificar el problema
	fmt.Printf("DEBUG - Verificando campos obligatorios...\n")
	fmt.Printf("DEBUG - Tipo vacío: %t\n", consulta.Tipo == "")
	fmt.Printf("DEBUG - Diagnostico vacío: %t\n", consulta.Diagnostico == "")
	fmt.Printf("DEBUG - IDPaciente cero: %t\n", consulta.IDPaciente == 0)
	fmt.Printf("DEBUG - IDMedico cero: %t\n", consulta.IDMedico == 0)

	// Verificar campos obligatorios
	if consulta.Tipo == "" {
		fmt.Printf("DEBUG - ERROR: Tipo de consulta vacío\n")
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F10",
				Data:    []interface{}{fiber.Map{"error": "El tipo de consulta es obligatorio"}},
			},
		})
	}

	// Diagnóstico ya no es obligatorio - validación eliminada

	if consulta.IDPaciente == 0 {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F10",
				Data:    []interface{}{fiber.Map{"error": "El ID del paciente es obligatorio"}},
			},
		})
	}

	if consulta.IDMedico == 0 {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F10",
				Data:    []interface{}{fiber.Map{"error": "El ID del médico es obligatorio"}},
			},
		})
	}

	// Verificar permisos usando el nuevo sistema de roles
	fmt.Printf("DEBUG - Verificando permisos...\n")
	userRole, ok := c.Locals("user_role").(string)
	fmt.Printf("DEBUG - UserRole: '%s', OK: %t\n", userRole, ok)
	if !ok || userRole == "" {
		fmt.Printf("DEBUG - ERROR: Usuario no autenticado o rol no válido\n")
		return c.Status(401).JSON(StandardResponse{
			StatusCode: 401,
			Body: BodyResponse{
				IntCode: "F10",
				Data:    []interface{}{fiber.Map{"error": "Usuario no autenticado o rol no válido"}},
			},
		})
	}
	if userRole != "medico" && userRole != "admin" {
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F10",
				Data:    []interface{}{fiber.Map{"error": "Solo médicos pueden crear consultas"}},
			},
		})
	}

	// Si es médico, debe ser el mismo que está en la consulta
	if userRole == "medico" {
		userID := c.Locals("user_id").(int)
		fmt.Printf("DEBUG - UserID: %d, Consulta.IDMedico: %d\n", userID, consulta.IDMedico)
		if consulta.IDMedico != userID {
			fmt.Printf("DEBUG - ERROR: Médico intentando crear consulta para otro médico\n")
			return c.Status(403).JSON(StandardResponse{
				StatusCode: 403,
				Body: BodyResponse{
					IntCode: "F10",
					Data:    []interface{}{fiber.Map{"error": "No puedes crear consultas para otro médico"}},
				},
			})
		}
	}

	// Verificar que el horario esté disponible (solo si se proporciona id_horario)
	if consulta.IDHorario != nil && *consulta.IDHorario > 0 {
		fmt.Printf("DEBUG - Verificando horario ID: %d\n", *consulta.IDHorario)
		var disponible bool
		err := database.GetDB().QueryRow(context.Background(),
			"SELECT consulta_disponible FROM Horario WHERE id_horario = $1", *consulta.IDHorario).Scan(&disponible)
		if err != nil {
			fmt.Printf("DEBUG - ERROR: Horario no encontrado: %v\n", err)
			return c.Status(400).JSON(StandardResponse{
				StatusCode: 400,
				Body: BodyResponse{
					IntCode: "F10",
					Data:    []interface{}{fiber.Map{"error": "Horario no encontrado"}},
				},
			})
		}
		if !disponible {
			fmt.Printf("DEBUG - ERROR: Horario no disponible\n")
			return c.Status(400).JSON(StandardResponse{
				StatusCode: 400,
				Body: BodyResponse{
					IntCode: "F10",
					Data:    []interface{}{fiber.Map{"error": "Horario no disponible"}},
				},
			})
		}
		fmt.Printf("DEBUG - Horario válido y disponible\n")
	}

	// Insertar consulta (manejando id_horario opcional)
	var nuevoID int
	var err error

	if consulta.IDHorario != nil && *consulta.IDHorario > 0 {
		// Con horario específico
		err = database.GetDB().QueryRow(context.Background(),
			`INSERT INTO Consulta (tipo, diagnostico, costo, id_paciente, id_medico, id_horario, hora)
			 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id_consulta`,
			consulta.Tipo, consulta.Diagnostico, consulta.Costo, consulta.IDPaciente, consulta.IDMedico,
			*consulta.IDHorario, consulta.Hora).Scan(&nuevoID)
	} else {
		// Sin horario específico (NULL)
		err = database.GetDB().QueryRow(context.Background(),
			`INSERT INTO Consulta (tipo, diagnostico, costo, id_paciente, id_medico, id_horario, hora)
			 VALUES ($1, $2, $3, $4, $5, NULL, $6) RETURNING id_consulta`,
			consulta.Tipo, consulta.Diagnostico, consulta.Costo, consulta.IDPaciente, consulta.IDMedico,
			consulta.Hora).Scan(&nuevoID)
	}

	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F10",
				Data:    []interface{}{fiber.Map{"error": "Error al crear la consulta"}},
			},
		})
	}

	// Marcar horario como no disponible (solo si se proporcionó un horario)
	// Comentar este bloque para que los horarios permanezcan disponibles
	/*
		if consulta.IDHorario != nil && *consulta.IDHorario > 0 {
		    _, err = database.GetDB().Exec(context.Background(),
		        "UPDATE Horario SET consulta_disponible = false WHERE id_horario = $1", *consulta.IDHorario)
		    if err != nil {
		        // Log error but don't fail the request
		    }
		}
	*/

	// Log evento de creación de consulta
	userEmail := ""
	if email := c.Locals("user_email"); email != nil {
		if emailStr, ok := email.(string); ok {
			userEmail = emailStr
		}
	}

	middleware.LogCustomEvent(
		models.LogLevelSuccess,
		"Consulta creada exitosamente",
		userEmail,
		userRole,
		map[string]interface{}{
			"consulta_id": nuevoID,
			"paciente_id": consulta.IDPaciente,
			"medico_id":   consulta.IDMedico,
			"horario_id":  consulta.IDHorario,
			"tipo":        consulta.Tipo,
			"action":      "consulta_created",
		},
	)

	return c.Status(201).JSON(StandardResponse{
		StatusCode: 201,
		Body: BodyResponse{
			IntCode: "S10",
			Data:    []interface{}{fiber.Map{"mensaje": "Consulta creada exitosamente", "id_consulta": nuevoID}},
		},
	})
}

// ObtenerConsultas obtiene las consultas según el rol de usuario
func ObtenerConsultas(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	userRole := c.Locals("user_role").(string)

	// Debug: Log para verificar el rol del usuario
	fmt.Printf("DEBUG - UserID: %d, UserRole: '%s'\n", userID, userRole)

	// Debug: Verificar si hay consultas en la base de datos
	var totalConsultas int
	err := database.GetDB().QueryRow(context.Background(), "SELECT COUNT(*) FROM Consulta").Scan(&totalConsultas)
	if err != nil {
		fmt.Printf("DEBUG - Error al contar consultas: %v\n", err)
	} else {
		fmt.Printf("DEBUG - Total consultas en BD: %d\n", totalConsultas)
	}

	var query string
	var args []interface{}

	switch userRole {
	case "admin":
		// Admin puede ver todas las consultas
		query = `SELECT c.id_consulta, c.tipo, c.diagnostico, c.costo, c.id_paciente, c.id_medico, c.id_horario, c.hora, p.nombre as paciente_nombre, m.nombre as medico_nombre, COALESCE(co.nombre_numero, 'Sin asignar') as consultorio_nombre, COALESCE(h.turno, 'Sin horario') as horario_turno FROM Consulta c JOIN Usuario p ON c.id_paciente = p.id_usuario JOIN Usuario m ON c.id_medico = m.id_usuario LEFT JOIN Horario h ON c.id_horario = h.id_horario LEFT JOIN Consultorio co ON h.id_consultorio = co.id_consultorio ORDER BY c.id_consulta DESC`
	case "medico":
		// Médico solo ve sus consultas
		query = `SELECT c.id_consulta, c.tipo, c.diagnostico, c.costo, c.id_paciente, c.id_medico, c.id_horario, c.hora, p.nombre as paciente_nombre, m.nombre as medico_nombre, COALESCE(co.nombre_numero, 'Sin asignar') as consultorio_nombre, COALESCE(h.turno, 'Sin horario') as horario_turno FROM Consulta c JOIN Usuario p ON c.id_paciente = p.id_usuario JOIN Usuario m ON c.id_medico = m.id_usuario LEFT JOIN Horario h ON c.id_horario = h.id_horario LEFT JOIN Consultorio co ON h.id_consultorio = co.id_consultorio WHERE c.id_medico = $1 ORDER BY c.id_consulta DESC`
		args = append(args, userID)
	case "paciente":
		// Paciente solo ve sus consultas
		query = `SELECT c.id_consulta, c.tipo, c.diagnostico, c.costo, c.id_paciente, c.id_medico, c.id_horario, c.hora, p.nombre as paciente_nombre, m.nombre as medico_nombre, COALESCE(co.nombre_numero, 'Sin asignar') as consultorio_nombre, COALESCE(h.turno, 'Sin horario') as horario_turno FROM Consulta c JOIN Usuario p ON c.id_paciente = p.id_usuario JOIN Usuario m ON c.id_medico = m.id_usuario LEFT JOIN Horario h ON c.id_horario = h.id_horario LEFT JOIN Consultorio co ON h.id_consultorio = co.id_consultorio WHERE c.id_paciente = $1 ORDER BY c.id_consulta DESC`
		args = append(args, userID)
	default:
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F11",
				Data:    []interface{}{fiber.Map{"error": "Tipo de usuario no autorizado"}},
			},
		})
	}

	// Debug: Log de la consulta SQL
	// Debug: Log de la consulta SQL
	fmt.Printf("DEBUG - Query: %s\n", query)
	fmt.Printf("DEBUG - Args: %v\n", args)

	rows, err := database.GetDB().Query(context.Background(), query, args...)
	if err != nil {
		fmt.Printf("DEBUG - Error en query: %v\n", err)
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F11",
				Data:    []interface{}{fiber.Map{"error": "Error al obtener consultas"}},
			},
		})
	}
	defer rows.Close()

	type ConsultaDetalle struct {
		models.Consulta
		PacienteNombre    string  `json:"paciente_nombre"`
		MedicoNombre      string  `json:"medico_nombre"`
		ConsultorioNombre *string `json:"consultorio_nombre"`
		HorarioTurno      *string `json:"horario_turno"`
	}

	var consultas []ConsultaDetalle
	fmt.Printf("DEBUG - Iniciando scan de rows\n")
	for rows.Next() {
		var consulta ConsultaDetalle
		err := rows.Scan(&consulta.ID, &consulta.Tipo, &consulta.Diagnostico, &consulta.Costo,
			&consulta.IDPaciente, &consulta.IDMedico, &consulta.IDHorario, &consulta.Hora,
			&consulta.PacienteNombre, &consulta.MedicoNombre, &consulta.ConsultorioNombre, &consulta.HorarioTurno)
		if err != nil {
			fmt.Printf("DEBUG - Error en scan: %v\n", err)
			continue
		}
		consultas = append(consultas, consulta)
		fmt.Printf("DEBUG - Consulta escaneada: ID=%d\n", consulta.ID)
	}

	// Debug: Log del resultado
	fmt.Printf("DEBUG - Total consultas encontradas: %d\n", len(consultas))
	for i, consulta := range consultas {
		fmt.Printf("DEBUG - Consulta %d: ID=%d, Tipo=%s, Paciente=%s, Medico=%s\n",
			i, consulta.ID, consulta.Tipo, consulta.PacienteNombre, consulta.MedicoNombre)
	}

	fmt.Printf("DEBUG - Devolviendo respuesta con %d consultas\n", len(consultas))
	return c.JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S11",
			Data:    []interface{}{fiber.Map{"consultas": consultas, "total": len(consultas)}},
		},
	})
}

// ActualizarConsulta actualiza una consulta existente
func ActualizarConsulta(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F12",
				Data:    []interface{}{fiber.Map{"error": "ID inválido"}},
			},
		})
	}

	// Verificar permisos
	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	if userRole != "medico" && userRole != "admin" {
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F12",
				Data:    []interface{}{fiber.Map{"error": "Solo médicos pueden actualizar consultas"}},
			},
		})
	}

	// Si es médico, verificar que sea su consulta
	if userRole == "medico" {
		var medicoConsulta int
		err := database.GetDB().QueryRow(context.Background(),
			"SELECT id_medico FROM Consulta WHERE id_consulta = $1", id).Scan(&medicoConsulta)
		if err != nil || medicoConsulta != userID {
			return c.Status(403).JSON(StandardResponse{
				StatusCode: 403,
				Body: BodyResponse{
					IntCode: "F12",
					Data:    []interface{}{fiber.Map{"error": "No puedes actualizar esta consulta"}},
				},
			})
		}
	}

	var consulta models.Consulta
	if err := c.BodyParser(&consulta); err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F12",
				Data:    []interface{}{fiber.Map{"error": "Datos inválidos"}},
			},
		})
	}

	// Actualizar consulta (campos editables)
	_, err = database.GetDB().Exec(context.Background(),
		`UPDATE Consulta SET tipo = $1, diagnostico = $2, costo = $3, id_medico = $4, hora = $5, id_horario = $6
		 WHERE id_consulta = $7`,
		consulta.Tipo, consulta.Diagnostico, consulta.Costo, consulta.IDMedico, consulta.Hora, consulta.IDHorario, id)

	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F12",
				Data:    []interface{}{fiber.Map{"error": "Error al actualizar consulta"}},
			},
		})
	}

	// Obtener el email del usuario para el log
	var userEmail string
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT email FROM Usuario WHERE id_usuario = $1", userID).Scan(&userEmail)
	if err != nil {
		userEmail = "unknown@email.com" // Fallback si no se puede obtener el email
	}

	// Log evento de actualización de consulta
	middleware.LogCustomEvent(
		models.LogLevelInfo,
		"Consulta actualizada",
		userEmail, // 🔥 USAR LA VARIABLE LOCAL
		userRole,
		map[string]interface{}{
			"consulta_id": id,
			"tipo":        consulta.Tipo,
			"diagnostico": consulta.Diagnostico,
			"costo":       consulta.Costo,
			"updated_by":  userEmail, // 🔥 USAR LA VARIABLE LOCAL
			"action":      "consulta_updated",
		},
	)

	return c.JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S12",
			Data:    []interface{}{fiber.Map{"mensaje": "Consulta actualizada exitosamente"}},
		},
	})
}

// ObtenerConsultaPorID obtiene una consulta específica por ID
func ObtenerConsultaPorID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F11",
				Data:    []interface{}{fiber.Map{"error": "ID inválido"}},
			},
		})
	}

	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	var consulta models.Consulta
	var nombrePaciente, nombreMedico, nombreConsultorio string

	query := `
		SELECT c.id_consulta, c.tipo, c.diagnostico, c.costo, c.id_paciente, c.id_medico,
		       c.id_horario, c.hora,
		       u1.nombre as nombre_paciente, u2.nombre as nombre_medico,
		       co.nombre_numero as nombre_consultorio
		FROM Consulta c
		JOIN Usuario u1 ON c.id_paciente = u1.id_usuario
		JOIN Usuario u2 ON c.id_medico = u2.id_usuario
		LEFT JOIN Horario h ON c.id_horario = h.id_horario
		LEFT JOIN Consultorio co ON h.id_consultorio = co.id_consultorio
		WHERE c.id_consulta = $1`

	// Agregar filtros según el rol de usuario
	if userRole == "paciente" {
		query += " AND c.id_paciente = $2"
		err = database.GetDB().QueryRow(context.Background(), query, id, userID).Scan(
			&consulta.ID, &consulta.Tipo, &consulta.Diagnostico, &consulta.Costo, &consulta.IDPaciente, &consulta.IDMedico, &consulta.IDHorario, &consulta.Hora,
			&nombrePaciente, &nombreMedico, &nombreConsultorio)
	} else if userRole == "medico" {
		query += " AND c.id_medico = $2"
		err = database.GetDB().QueryRow(context.Background(), query, id, userID).Scan(
			&consulta.ID, &consulta.Tipo, &consulta.Diagnostico, &consulta.Costo, &consulta.IDPaciente, &consulta.IDMedico, &consulta.IDHorario, &consulta.Hora,
			&nombrePaciente, &nombreMedico, &nombreConsultorio)
	} else {
		// Admin y enfermera pueden ver todas
		err = database.GetDB().QueryRow(context.Background(), query, id).Scan(
			&consulta.ID, &consulta.Tipo, &consulta.Diagnostico, &consulta.Costo, &consulta.IDPaciente, &consulta.IDMedico, &consulta.IDHorario, &consulta.Hora,
			&nombrePaciente, &nombreMedico, &nombreConsultorio)
	}

	if err != nil {
		return c.Status(404).JSON(StandardResponse{
			StatusCode: 404,
			Body: BodyResponse{
				IntCode: "F11",
				Data:    []interface{}{fiber.Map{"error": "Consulta no encontrada"}},
			},
		})
	}

	return c.JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S11",
			Data:    []interface{}{fiber.Map{"consulta": consulta, "nombre_paciente": nombrePaciente, "nombre_medico": nombreMedico, "nombre_consultorio": nombreConsultorio}},
		},
	})
}

// ObtenerConsultasPorPaciente obtiene todas las consultas de un paciente específico
func ObtenerConsultasPorPaciente(c *fiber.Ctx) error {
	pacienteID, err := strconv.Atoi(c.Params("paciente_id"))
	if err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F11",
				Data:    []interface{}{fiber.Map{"error": "ID de paciente inválido"}},
			},
		})
	}

	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	// Verificar permisos
	if userRole == "paciente" && pacienteID != userID {
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F11",
				Data:    []interface{}{fiber.Map{"error": "No puedes ver las consultas de otro paciente"}},
			},
		})
	}

	query := `
		SELECT c.id_consulta, c.tipo, c.diagnostico, c.costo, c.id_paciente, c.id_medico, c.id_horario, c.hora,
		       u2.nombre as nombre_medico, co.nombre_numero as nombre_consultorio
		FROM Consulta c
		JOIN Usuario u2 ON c.id_medico = u2.id_usuario
		LEFT JOIN Horario h ON c.id_horario = h.id_horario
		LEFT JOIN Consultorio co ON h.id_consultorio = co.id_consultorio
		WHERE c.id_paciente = $1
		ORDER BY c.id_consulta DESC`

	rows, err := database.GetDB().Query(context.Background(), query, pacienteID)
	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F11",
				Data:    []interface{}{fiber.Map{"error": "Error al obtener consultas"}},
			},
		})
	}
	defer rows.Close()

	var consultas []map[string]interface{}
	for rows.Next() {
		var consulta models.Consulta
		var nombreMedico, nombreConsultorio string

		err := rows.Scan(
			&consulta.ID, &consulta.Tipo, &consulta.Diagnostico, &consulta.Costo, &consulta.IDPaciente, &consulta.IDMedico, &consulta.IDHorario, &consulta.Hora,
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

	return c.JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S11",
			Data:    []interface{}{fiber.Map{"consultas": consultas, "total": len(consultas)}},
		},
	})
}

// ObtenerConsultasPorMedico obtiene todas las consultas de un médico específico
func ObtenerConsultasPorMedico(c *fiber.Ctx) error {
	medicoID, err := strconv.Atoi(c.Params("medico_id"))
	if err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F11",
				Data:    []interface{}{fiber.Map{"error": "ID de médico inválido"}},
			},
		})
	}

	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	// Verificar permisos
	if userRole == "medico" && medicoID != userID {
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F11",
				Data:    []interface{}{fiber.Map{"error": "No puedes ver las consultas de otro médico"}},
			},
		})
	}

	query := `
		SELECT c.id_consulta, c.tipo, c.diagnostico, c.costo, c.id_paciente, c.id_medico, c.id_horario, c.hora,
		       u1.nombre as nombre_paciente, co.nombre_numero as nombre_consultorio
		FROM Consulta c
		JOIN Usuario u1 ON c.id_paciente = u1.id_usuario
		LEFT JOIN Horario h ON c.id_horario = h.id_horario
		LEFT JOIN Consultorio co ON h.id_consultorio = co.id_consultorio
		WHERE c.id_medico = $1
		ORDER BY c.id_consulta DESC`

	rows, err := database.GetDB().Query(context.Background(), query, medicoID)
	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F11",
				Data:    []interface{}{fiber.Map{"error": "Error al obtener consultas"}},
			},
		})
	}
	defer rows.Close()

	var consultas []map[string]interface{}
	for rows.Next() {
		var consulta models.Consulta
		var nombrePaciente, nombreConsultorio string

		err := rows.Scan(
			&consulta.ID, &consulta.Tipo, &consulta.Diagnostico, &consulta.Costo, &consulta.IDPaciente, &consulta.IDMedico, &consulta.IDHorario, &consulta.Hora,
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

	return c.JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S11",
			Data:    []interface{}{fiber.Map{"consultas": consultas, "total": len(consultas)}},
		},
	})
}

// CompletarConsulta marca una consulta como completada
func CompletarConsulta(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F12",
				Data:    []interface{}{fiber.Map{"error": "ID inválido"}},
			},
		})
	}

	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	// Solo médicos pueden completar consultas
	if userRole != "medico" {
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F12",
				Data:    []interface{}{fiber.Map{"error": "Solo médicos pueden completar consultas"}},
			},
		})
	}

	// Verificar que la consulta pertenece al médico
	var medicoID int
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT id_medico FROM Consulta WHERE id_consulta = $1", id).Scan(&medicoID)
	if err != nil {
		return c.Status(404).JSON(StandardResponse{
			StatusCode: 404,
			Body: BodyResponse{
				IntCode: "F12",
				Data:    []interface{}{fiber.Map{"error": "Consulta no encontrada"}},
			},
		})
	}

	if medicoID != userID {
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F12",
				Data:    []interface{}{fiber.Map{"error": "No puedes completar esta consulta"}},
			},
		})
	}

	// Como no existe campo estado en la tabla, solo retornamos éxito
	// La lógica de completar consulta se manejará a nivel de aplicación
	return c.JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S12",
			Data:    []interface{}{fiber.Map{"mensaje": "Consulta completada exitosamente"}},
		},
	})
}

// EliminarConsulta elimina una consulta permanentemente
func EliminarConsulta(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F13",
				Data:    []interface{}{fiber.Map{"error": "ID inválido"}},
			},
		})
	}

	// Verificar permisos
	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	if userRole != "admin" && userRole != "medico" {
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F13",
				Data:    []interface{}{fiber.Map{"error": "Solo administradores y médicos pueden eliminar consultas"}},
			},
		})
	}

	// Obtener información de la consulta antes de eliminarla
	var consulta models.Consulta
	var pacienteID, medicoID, horarioID int
	var tipo, diagnostico string
	var costo float64
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT id_consulta, tipo, diagnostico, costo, id_paciente, id_medico, id_horario FROM Consulta WHERE id_consulta = $1", id).Scan(
		&consulta.ID, &tipo, &diagnostico, &costo, &pacienteID, &medicoID, &horarioID)

	if err != nil {
		return c.Status(404).JSON(StandardResponse{
			StatusCode: 404,
			Body: BodyResponse{
				IntCode: "F13",
				Data:    []interface{}{fiber.Map{"error": "Consulta no encontrada"}},
			},
		})
	}

	// Si es médico, verificar que sea su consulta
	if userRole == "medico" && medicoID != userID {
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F13",
				Data:    []interface{}{fiber.Map{"error": "No puedes eliminar esta consulta"}},
			},
		})
	}

	// Eliminar la consulta
	_, err = database.GetDB().Exec(context.Background(),
		"DELETE FROM Consulta WHERE id_consulta = $1", id)

	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F13",
				Data:    []interface{}{fiber.Map{"error": "Error al eliminar la consulta"}},
			},
		})
	}

	// Liberar el horario
	_, err = database.GetDB().Exec(context.Background(),
		"UPDATE Horario SET consulta_disponible = true WHERE id_horario = $1", horarioID)
	if err != nil {
		// Log error but don't fail the request
	}

	// Obtener el email del usuario para el log
	var userEmail string
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT email FROM Usuario WHERE id_usuario = $1", userID).Scan(&userEmail)
	if err != nil {
		userEmail = "unknown@email.com" // Fallback si no se puede obtener el email
	}

	// Log evento de eliminación de consulta
	middleware.LogCustomEvent(
		models.LogLevelWarning,
		"Consulta eliminada",
		userEmail, // 🔥 USAR LA VARIABLE LOCAL
		userRole,
		map[string]interface{}{
			"consulta_id": id,
			"paciente_id": pacienteID,
			"medico_id":   medicoID,
			"horario_id":  horarioID,
			"tipo":        tipo,
			"deleted_by":  userEmail, // 🔥 USAR LA VARIABLE LOCAL
			"action":      "consulta_deleted",
		},
	)

	return c.JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S13",
			Data:    []interface{}{fiber.Map{"mensaje": "Consulta eliminada exitosamente"}},
		},
	})
}

// CancelarConsulta cancela una consulta y libera el horario
func CancelarConsulta(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(StandardResponse{
			StatusCode: 400,
			Body: BodyResponse{
				IntCode: "F12",
				Data:    []interface{}{fiber.Map{"error": "ID inválido"}},
			},
		})
	}

	// Verificar permisos
	userRole := c.Locals("user_role").(string)
	userID := c.Locals("user_id").(int)

	// Obtener información de la consulta
	var consulta models.Consulta
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT id_consulta, id_paciente, id_medico, id_horario FROM Consulta WHERE id_consulta = $1", id).Scan(
		&consulta.ID, &consulta.IDPaciente, &consulta.IDMedico, &consulta.IDHorario)

	if err != nil {
		return c.Status(404).JSON(StandardResponse{
			StatusCode: 404,
			Body: BodyResponse{
				IntCode: "F12",
				Data:    []interface{}{fiber.Map{"error": "Consulta no encontrada"}},
			},
		})
	}

	// Verificar permisos específicos
	if userRole == "paciente" && consulta.IDPaciente != userID {
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F12",
				Data:    []interface{}{fiber.Map{"error": "No puedes cancelar esta consulta"}},
			},
		})
	}
	if userRole == "medico" && consulta.IDMedico != userID {
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F12",
				Data:    []interface{}{fiber.Map{"error": "No puedes cancelar esta consulta"}},
			},
		})
	}

	// Como no existe campo estado en la tabla, asumimos que todas las consultas se pueden cancelar

	// Como no existe campo estado en la tabla, solo liberamos el horario
	// La lógica de cancelar consulta se manejará a nivel de aplicación

	// Liberar horario
	_, err = database.GetDB().Exec(context.Background(),
		"UPDATE Horario SET consulta_disponible = true WHERE id_horario = $1", consulta.IDHorario)
	if err != nil {
		// Log error but don't fail the request
	}

	return c.JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S12",
			Data:    []interface{}{fiber.Map{"mensaje": "Consulta cancelada exitosamente"}},
		},
	})
}
