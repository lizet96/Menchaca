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
	// Solo admin y médicos pueden generar reportes
	userType := c.Locals("user_type").(string)
	if userType != "admin" && userType != "medico" {
		return c.Status(403).JSON(fiber.Map{
			"error": "No tienes permisos para generar reportes",
		})
	}

	userID := c.Locals("user_id").(int)
	var whereClause string
	var args []interface{}

	// Si es médico, solo sus consultas
	if userType == "medico" {
		whereClause = "WHERE id_medico = $1"
		args = append(args, userID)
	}

	// Obtener estadísticas generales
	var reporte models.ReporteConsultas
	reporte.FechaGeneracion = time.Now()

	// Total de consultas
	query := "SELECT COUNT(*) FROM Consulta " + whereClause
	err := database.GetDB().QueryRow(context.Background(), query, args...).Scan(&reporte.TotalConsultas)
	if err != nil {
		reporte.TotalConsultas = 0
	}

	// Consultas de hoy
	hoy := time.Now().Format("2006-01-02")
	queryHoy := "SELECT COUNT(*) FROM Consulta WHERE DATE(fecha) = $1"
	argsHoy := []interface{}{hoy}
	if userType == "medico" {
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
	if userType == "medico" {
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
	// Solo admin puede ver estadísticas generales
	userType := c.Locals("user_type").(string)
	if userType != "admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo administradores pueden ver estadísticas generales",
		})
	}

	type Estadisticas struct {
		TotalUsuarios    int     `json:"total_usuarios"`
		TotalPacientes   int     `json:"total_pacientes"`
		TotalMedicos     int     `json:"total_medicos"`
		TotalEnfermeras  int     `json:"total_enfermeras"`
		TotalConsultas   int     `json:"total_consultas"`
		ConsultasHoy     int     `json:"consultas_hoy"`
		TotalExpedientes int     `json:"total_expedientes"`
		IngresosMes      float64 `json:"ingresos_mes"`
		FechaGeneracion  time.Time `json:"fecha_generacion"`
	}

	var stats Estadisticas
	stats.FechaGeneracion = time.Now()

	// Total de usuarios
	err := database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Usuario").Scan(&stats.TotalUsuarios)
	if err != nil {
		stats.TotalUsuarios = 0
	}

	// Total por tipo de usuario
	database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Usuario WHERE tipo = 'paciente'").Scan(&stats.TotalPacientes)
	database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Usuario WHERE tipo = 'medico'").Scan(&stats.TotalMedicos)
	database.GetDB().QueryRow(context.Background(),
		"SELECT COUNT(*) FROM Usuario WHERE tipo = 'enfermera'").Scan(&stats.TotalEnfermeras)

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
	userType := c.Locals("user_type").(string)
	if userType != "admin" && userType != "medico" {
		return c.Status(403).JSON(fiber.Map{
			"error": "No tienes permisos para ver este reporte",
		})
	}

	userID := c.Locals("user_id").(int)
	var query string
	var args []interface{}

	if userType == "admin" {
		// Admin puede ver todos los médicos y sus pacientes
		query = `SELECT m.nombre as medico_nombre, COUNT(DISTINCT c.id_paciente) as total_pacientes,
				 COUNT(c.id_consulta) as total_consultas
				 FROM Usuario m
				 LEFT JOIN Consulta c ON m.id_usuario = c.id_medico
				 WHERE m.tipo = 'medico'
				 GROUP BY m.id_usuario, m.nombre
				 ORDER BY total_pacientes DESC`
	} else {
		// Médico solo ve sus propios pacientes
		query = `SELECT m.nombre as medico_nombre, COUNT(DISTINCT c.id_paciente) as total_pacientes,
				 COUNT(c.id_consulta) as total_consultas
				 FROM Usuario m
				 LEFT JOIN Consulta c ON m.id_usuario = c.id_medico
				 WHERE m.tipo = 'medico' AND m.id_usuario = $1
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
		MedicoNombre    string `json:"medico_nombre"`
		TotalPacientes  int    `json:"total_pacientes"`
		TotalConsultas  int    `json:"total_consultas"`
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
	// Solo admin puede ver reportes de ingresos
	userType := c.Locals("user_type").(string)
	if userType != "admin" {
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
		Fecha           string  `json:"fecha"`
		TotalConsultas  int     `json:"total_consultas"`
		IngresosDia     float64 `json:"ingresos_dia"`
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