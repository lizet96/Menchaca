package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lizet96/hospital-backend/database"
	"github.com/lizet96/hospital-backend/models"
)

// GenerarReporteConsultas genera un reporte de consultas
func GenerarReporteConsultas(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(int)

	// Verificar si el usuario es médico para filtrar sus consultas
	var rolNombre string
	err := database.GetDB().QueryRow(context.Background(), `
	    SELECT r.nombre 
	    FROM Usuario u 
	    JOIN Rol r ON u.id_rol = r.id_rol 
	    WHERE u.id_usuario = $1
	`, userID).Scan(&rolNombre)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener información del usuario",
		})
	}

	var whereClause string
	var args []interface{}

	// Si es médico, solo sus consultas
	if rolNombre == "medico" {
		whereClause = "WHERE id_medico = $1"
		args = append(args, userID)
	}

	// Obtener estadísticas generales
	var reporte models.ReporteConsultas
	reporte.FechaGeneracion = time.Now()

	// Total de consultas
	query := "SELECT COUNT(*) FROM Consulta " + whereClause
	err = database.GetDB().QueryRow(context.Background(), query, args...).Scan(&reporte.TotalConsultas)
	if err != nil {
		reporte.TotalConsultas = 0
	}

	// Consultas de hoy
	hoy := time.Now().Format("2006-01-02")
	queryHoy := "SELECT COUNT(*) FROM Consulta WHERE DATE(fecha) = $1"
	argsHoy := []interface{}{hoy}
	if rolNombre == "medico" {
		queryHoy += " AND id_medico = $2"
		argsHoy = append(argsHoy, userID)
	}
	err = database.GetDB().QueryRow(context.Background(), queryHoy, argsHoy...).Scan(&reporte.ConsultasHoy)
	if err != nil {
		reporte.ConsultasHoy = 0
	}

	// Consultas de esta semana
	inicioSemana := time.Now().AddDate(0, 0, -int(time.Now().Weekday())).Format("2006-01-02")
	querySemana := "SELECT COUNT(*) FROM Consulta WHERE DATE(fecha) >= $1"
	argsSemana := []interface{}{inicioSemana}
	if rolNombre == "medico" {
		querySemana += " AND id_medico = $2"
		argsSemana = append(argsSemana, userID)
	}
	err = database.GetDB().QueryRow(context.Background(), querySemana, argsSemana...).Scan(&reporte.ConsultasSemana)
	if err != nil {
		reporte.ConsultasSemana = 0
	}

	// Ingresos totales
	queryIngresos := "SELECT COALESCE(SUM(costo), 0) FROM Consulta WHERE estado = 'completada' " + whereClause
	err = database.GetDB().QueryRow(context.Background(), queryIngresos, args...).Scan(&reporte.IngresosTotales)
	if err != nil {
		reporte.IngresosTotales = 0
	}

	// Promedio de consultas por día (últimos 30 días)
	if reporte.TotalConsultas > 0 {
		reporte.PromedioConsultas = float64(reporte.TotalConsultas) / 30.0
	}

	return c.JSON(fiber.Map{
		"reporte": reporte,
		"mensaje": "Reporte generado exitosamente",
	})
}

// ObtenerEstadisticasGenerales obtiene estadísticas generales del sistema
func ObtenerEstadisticasGenerales(c *fiber.Ctx) error {
	// Verificar si el usuario es admin usando el nuevo sistema de roles
	userID := c.Locals("user_id").(int)
	var rolNombre string
	err := database.GetDB().QueryRow(context.Background(), `
	    SELECT r.nombre 
	    FROM Usuario u 
	    JOIN Rol r ON u.id_rol = r.id_rol 
	    WHERE u.id_usuario = $1
	`, userID).Scan(&rolNombre)

	if err != nil || rolNombre != "admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo administradores pueden ver estadísticas generales",
		})
	}

	type Estadisticas struct {
		TotalUsuarios    int       `json:"total_usuarios"`
		TotalPacientes   int       `json:"total_pacientes"`
		TotalMedicos     int       `json:"total_medicos"`
		TotalEnfermeras  int       `json:"total_enfermeras"`
		TotalConsultas   int       `json:"total_consultas"`
		ConsultasHoy     int       `json:"consultas_hoy"`
		TotalExpedientes int       `json:"total_expedientes"`
		IngresosMes      float64   `json:"ingresos_mes"`
		FechaGeneracion  time.Time `json:"fecha_generacion"`
	}

	var stats Estadisticas
	stats.FechaGeneracion = time.Now()

	// Total de usuarios
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Usuario").Scan(&stats.TotalUsuarios)
	if err != nil {
		stats.TotalUsuarios = 0
	}

	// Total por tipo de usuario usando el nuevo sistema de roles
	database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Usuario u JOIN Rol r ON u.id_rol = r.id_rol WHERE r.nombre = 'paciente'").Scan(&stats.TotalPacientes)
	database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Usuario u JOIN Rol r ON u.id_rol = r.id_rol WHERE r.nombre = 'medico'").Scan(&stats.TotalMedicos)
	database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Usuario u JOIN Rol r ON u.id_rol = r.id_rol WHERE r.nombre = 'enfermera'").Scan(&stats.TotalEnfermeras)

	// Total de consultas
	database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Consulta").Scan(&stats.TotalConsultas)

	// Consultas de hoy
	hoy := time.Now().Format("2006-01-02")
	database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Consulta WHERE DATE(fecha) = $1", hoy).Scan(&stats.ConsultasHoy)

	// Total de expedientes
	database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Expediente").Scan(&stats.TotalExpedientes)

	// Ingresos del mes actual
	inicioMes := time.Now().Format("2006-01-01")
	database.GetDB().QueryRow(context.Background(),
		"SELECT COALESCE(SUM(costo), 0) FROM Consulta WHERE estado = 'completada' AND DATE(fecha) >= $1",
		inicioMes).Scan(&stats.IngresosMes)

	return c.JSON(fiber.Map{
		"estadisticas": stats,
		"mensaje":      "Estadísticas obtenidas exitosamente",
	})
}

// ObtenerReportePacientes obtiene reporte de pacientes por médico
func ObtenerReportePacientes(c *fiber.Ctx) error {
	// Verificar permisos usando el nuevo sistema de roles
	userID := c.Locals("user_id").(int)
	var rolNombre string
	err := database.GetDB().QueryRow(context.Background(), `
	    SELECT r.nombre 
	    FROM Usuario u 
	    JOIN Rol r ON u.id_rol = r.id_rol 
	    WHERE u.id_usuario = $1
	`, userID).Scan(&rolNombre)

	if err != nil || (rolNombre != "admin" && rolNombre != "medico") {
		return c.Status(403).JSON(fiber.Map{
			"error": "No tienes permisos para ver este reporte",
		})
	}

	var query string
	var args []interface{}

	if rolNombre == "admin" {
		// Admin puede ver todos los médicos y sus pacientes
		query = `SELECT m.nombre as medico_nombre, COUNT(DISTINCT c.id_paciente) as total_pacientes,
				 COUNT(c.id_consulta) as total_consultas
				 FROM Usuario m
				 JOIN Rol r ON m.id_rol = r.id_rol
				 LEFT JOIN Consulta c ON m.id_usuario = c.id_medico
				 WHERE r.nombre = 'medico'
				 GROUP BY m.id_usuario, m.nombre
				 ORDER BY total_pacientes DESC`
	} else {
		// Médico solo ve sus propios pacientes
		query = `SELECT m.nombre as medico_nombre, COUNT(DISTINCT c.id_paciente) as total_pacientes,
				 COUNT(c.id_consulta) as total_consultas
				 FROM Usuario m
				 JOIN Rol r ON m.id_rol = r.id_rol
				 LEFT JOIN Consulta c ON m.id_usuario = c.id_medico
				 WHERE r.nombre = 'medico' AND m.id_usuario = $1
				 GROUP BY m.id_usuario, m.nombre`
		args = append(args, userID)
	}

	rows, err := database.GetDB().Query(context.Background(), query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al generar reporte",
		})
	}
	defer rows.Close()

	type ReporteMedico struct {
		MedicoNombre   string `json:"medico_nombre"`
		TotalPacientes int    `json:"total_pacientes"`
		TotalConsultas int    `json:"total_consultas"`
	}

	var reportes []ReporteMedico
	for rows.Next() {
		var reporte ReporteMedico
		err := rows.Scan(&reporte.MedicoNombre, &reporte.TotalPacientes, &reporte.TotalConsultas)
		if err != nil {
			continue
		}
		reportes = append(reportes, reporte)
	}

	return c.JSON(fiber.Map{
		"reporte_pacientes": reportes,
		"total_medicos":     len(reportes),
		"fecha_generacion":  time.Now(),
	})
}

// ObtenerReporteIngresos obtiene reporte de ingresos por período
func ObtenerReporteIngresos(c *fiber.Ctx) error {
	// Verificar si el usuario es admin usando el nuevo sistema de roles
	userID := c.Locals("user_id").(int)
	var rolNombre string
	err := database.GetDB().QueryRow(context.Background(), `
	    SELECT r.nombre 
	    FROM Usuario u 
	    JOIN Rol r ON u.id_rol = r.id_rol 
	    WHERE u.id_usuario = $1
	`, userID).Scan(&rolNombre)

	if err != nil || rolNombre != "admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo administradores pueden ver reportes de ingresos",
		})
	}

	// Obtener parámetros de fecha (opcional)
	fechaInicio := c.Query("fecha_inicio")
	fechaFin := c.Query("fecha_fin")

	if fechaInicio == "" {
		// Por defecto, último mes
		fechaInicio = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if fechaFin == "" {
		fechaFin = time.Now().Format("2006-01-02")
	}

	type ReporteIngresos struct {
		Fecha          string  `json:"fecha"`
		TotalConsultas int     `json:"total_consultas"`
		IngresosDia    float64 `json:"ingresos_dia"`
	}

	query := `SELECT DATE(fecha) as fecha, COUNT(*) as total_consultas, 
			  COALESCE(SUM(costo), 0) as ingresos_dia
			  FROM Consulta 
			  WHERE estado = 'completada' AND DATE(fecha) BETWEEN $1 AND $2
			  GROUP BY DATE(fecha)
			  ORDER BY fecha DESC`

	rows, err := database.GetDB().Query(context.Background(), query, fechaInicio, fechaFin)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al generar reporte de ingresos",
		})
	}
	defer rows.Close()

	var reportes []ReporteIngresos
	var totalIngresos float64
	var totalConsultas int

	for rows.Next() {
		var reporte ReporteIngresos
		err := rows.Scan(&reporte.Fecha, &reporte.TotalConsultas, &reporte.IngresosDia)
		if err != nil {
			continue
		}
		reportes = append(reportes, reporte)
		totalIngresos += reporte.IngresosDia
		totalConsultas += reporte.TotalConsultas
	}

	return c.JSON(fiber.Map{
		"reporte_ingresos": reportes,
		"resumen": fiber.Map{
			"fecha_inicio":     fechaInicio,
			"fecha_fin":        fechaFin,
			"total_ingresos":   totalIngresos,
			"total_consultas":  totalConsultas,
			"promedio_diario":  totalIngresos / float64(len(reportes)),
			"fecha_generacion": time.Now(),
		},
	})
}

// GenerarReporteUsuarios genera un reporte de usuarios del sistema
func GenerarReporteUsuarios(c *fiber.Ctx) error {
	// Verificar si el usuario es admin usando el nuevo sistema de roles
	userID := c.Locals("user_id").(int)
	var rolNombre string
	err := database.GetDB().QueryRow(context.Background(), `
	    SELECT r.nombre 
	    FROM Usuario u 
	    JOIN Rol r ON u.id_rol = r.id_rol 
	    WHERE u.id_usuario = $1
	`, userID).Scan(&rolNombre)

	if err != nil || rolNombre != "admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo administradores pueden ver reportes de usuarios",
		})
	}

	type ReporteUsuario struct {
		IDUsuario     int       `json:"id_usuario"`
		Nombre        string    `json:"nombre"`
		Apellido      string    `json:"apellido"`
		Email         string    `json:"email"`
		Rol           string    `json:"rol"`
		FechaRegistro time.Time `json:"fecha_registro"`
		MFAEnabled    bool      `json:"mfa_enabled"`
	}

	query := `
		SELECT u.id_usuario, u.nombre, u.apellido, u.email, r.nombre as rol, 
		       u.created_at, u.mfa_enabled
		FROM Usuario u 
		JOIN Rol r ON u.id_rol = r.id_rol 
		ORDER BY u.created_at DESC
	`

	rows, err := database.GetDB().Query(context.Background(), query)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al generar reporte de usuarios",
		})
	}
	defer rows.Close()

	var usuarios []ReporteUsuario
	var totalPorRol = make(map[string]int)

	for rows.Next() {
		var usuario ReporteUsuario
		err := rows.Scan(&usuario.IDUsuario, &usuario.Nombre, &usuario.Apellido,
			&usuario.Email, &usuario.Rol, &usuario.FechaRegistro, &usuario.MFAEnabled)
		if err != nil {
			continue
		}
		usuarios = append(usuarios, usuario)
		totalPorRol[usuario.Rol]++
	}

	return c.JSON(fiber.Map{
		"reporte_usuarios": usuarios,
		"resumen": fiber.Map{
			"total_usuarios":   len(usuarios),
			"total_por_rol":    totalPorRol,
			"fecha_generacion": time.Now(),
		},
	})
}

// GenerarReporteExpedientes genera un reporte de expedientes médicos
func GenerarReporteExpedientes(c *fiber.Ctx) error {
	// Verificar permisos usando el nuevo sistema de roles
	userID := c.Locals("user_id").(int)
	var rolNombre string
	err := database.GetDB().QueryRow(context.Background(), `
	    SELECT r.nombre 
	    FROM Usuario u 
	    JOIN Rol r ON u.id_rol = r.id_rol 
	    WHERE u.id_usuario = $1
	`, userID).Scan(&rolNombre)

	if err != nil || (rolNombre != "admin" && rolNombre != "medico" && rolNombre != "enfermera") {
		return c.Status(403).JSON(fiber.Map{
			"error": "No tienes permisos para ver este reporte",
		})
	}

	type ReporteExpediente struct {
		IDExpediente   int        `json:"id_expediente"`
		PacienteNombre string     `json:"paciente_nombre"`
		PacienteEmail  string     `json:"paciente_email"`
		MedicoNombre   string     `json:"medico_nombre"`
		FechaCreacion  time.Time  `json:"fecha_creacion"`
		TotalConsultas int        `json:"total_consultas"`
		UltimaConsulta *time.Time `json:"ultima_consulta,omitempty"`
	}

	var query string
	var args []interface{}

	if rolNombre == "admin" || rolNombre == "enfermera" {
		// Admin y enfermeras pueden ver todos los expedientes
		query = `
			SELECT e.id_expediente, 
			       p.nombre || ' ' || p.apellido as paciente_nombre,
			       p.email as paciente_email,
			       m.nombre || ' ' || m.apellido as medico_nombre,
			       e.fecha_creacion,
			       COUNT(c.id_consulta) as total_consultas,
			       MAX(c.fecha) as ultima_consulta
			FROM Expediente e
			JOIN Usuario p ON e.id_paciente = p.id_usuario
			JOIN Usuario m ON e.id_medico = m.id_usuario
			LEFT JOIN Consulta c ON e.id_expediente = c.id_expediente
			GROUP BY e.id_expediente, p.nombre, p.apellido, p.email, m.nombre, m.apellido, e.fecha_creacion
			ORDER BY e.fecha_creacion DESC
		`
	} else {
		// Médicos solo ven sus propios expedientes
		query = `
			SELECT e.id_expediente, 
			       p.nombre || ' ' || p.apellido as paciente_nombre,
			       p.email as paciente_email,
			       m.nombre || ' ' || m.apellido as medico_nombre,
			       e.fecha_creacion,
			       COUNT(c.id_consulta) as total_consultas,
			       MAX(c.fecha) as ultima_consulta
			FROM Expediente e
			JOIN Usuario p ON e.id_paciente = p.id_usuario
			JOIN Usuario m ON e.id_medico = m.id_usuario
			LEFT JOIN Consulta c ON e.id_expediente = c.id_expediente
			WHERE e.id_medico = $1
			GROUP BY e.id_expediente, p.nombre, p.apellido, p.email, m.nombre, m.apellido, e.fecha_creacion
			ORDER BY e.fecha_creacion DESC
		`
		args = append(args, userID)
	}

	rows, err := database.GetDB().Query(context.Background(), query, args...)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al generar reporte de expedientes",
		})
	}
	defer rows.Close()

	var expedientes []ReporteExpediente
	var totalConsultas int

	for rows.Next() {
		var expediente ReporteExpediente
		var ultimaConsulta *time.Time
		err := rows.Scan(&expediente.IDExpediente, &expediente.PacienteNombre,
			&expediente.PacienteEmail, &expediente.MedicoNombre,
			&expediente.FechaCreacion, &expediente.TotalConsultas, &ultimaConsulta)
		if err != nil {
			continue
		}
		expediente.UltimaConsulta = ultimaConsulta
		expedientes = append(expedientes, expediente)
		totalConsultas += expediente.TotalConsultas
	}

	return c.JSON(fiber.Map{
		"reporte_expedientes": expedientes,
		"resumen": fiber.Map{
			"total_expedientes": len(expedientes),
			"total_consultas":   totalConsultas,
			"promedio_consultas_por_expediente": func() float64 {
				if len(expedientes) > 0 {
					return float64(totalConsultas) / float64(len(expedientes))
				}
				return 0
			}(),
			"fecha_generacion": time.Now(),
		},
	})
}
