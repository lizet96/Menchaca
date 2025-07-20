package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lizet96/hospital-backend/database"
	"github.com/lizet96/hospital-backend/models"
)

// ObtenerLogs obtiene logs con filtros opcionales
func ObtenerLogs(c *fiber.Ctx) error {
	// Solo admin puede ver logs
	userRole := c.Locals("user_role").(string)
	if userRole != "admin" {
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F70",
				Data:    []interface{}{fiber.Map{"error": "Solo administradores pueden ver logs"}},
			},
		})
	}

	// Parámetros de paginación
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset := (page - 1) * limit

	// Filtros opcionales
	logLevel := c.Query("log_level")
	method := c.Query("method")
	statusCode := c.Query("status_code")
	email := c.Query("email")
	ip := c.Query("ip")
	fechaInicio := c.Query("fecha_inicio")
	fechaFin := c.Query("fecha_fin")
	path := c.Query("path")

	// Construir query dinámicamente
	var conditions []string
	var args []interface{}
	argIndex := 1

	if logLevel != "" {
		conditions = append(conditions, fmt.Sprintf("log_level = $%d", argIndex))
		args = append(args, logLevel)
		argIndex++
	}

	if method != "" {
		conditions = append(conditions, fmt.Sprintf("method = $%d", argIndex))
		args = append(args, method)
		argIndex++
	}

	if statusCode != "" {
		if code, err := strconv.Atoi(statusCode); err == nil {
			conditions = append(conditions, fmt.Sprintf("status_code = $%d", argIndex))
			args = append(args, code)
			argIndex++
		}
	}

	if email != "" {
		conditions = append(conditions, fmt.Sprintf("email ILIKE $%d", argIndex))
		args = append(args, "%"+email+"%")
		argIndex++
	}

	if ip != "" {
		conditions = append(conditions, fmt.Sprintf("ip = $%d", argIndex))
		args = append(args, ip)
		argIndex++
	}

	if path != "" {
		conditions = append(conditions, fmt.Sprintf("path ILIKE $%d", argIndex))
		args = append(args, "%"+path+"%")
		argIndex++
	}

	if fechaInicio != "" {
		if fecha, err := time.Parse("2006-01-02", fechaInicio); err == nil {
			conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIndex))
			args = append(args, fecha)
			argIndex++
		}
	}

	if fechaFin != "" {
		if fecha, err := time.Parse("2006-01-02", fechaFin); err == nil {
			// Agregar 24 horas para incluir todo el día
			fecha = fecha.Add(24 * time.Hour)
			conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIndex))
			args = append(args, fecha)
			argIndex++
		}
	}

	// Construir WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Query para contar total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM logs %s", whereClause)
	var total int
	err := database.GetDB().QueryRow(context.Background(), countQuery, args...).Scan(&total)
	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F70",
				Data:    []interface{}{fiber.Map{"error": "Error al contar logs"}},
			},
		})
	}

	// Query principal con paginación
	query := fmt.Sprintf(`
		SELECT id_log, method, path, protocol, status_code, response_time, user_agent, ip, hostname,
		       body, params, query, email, username, role, log_level, environment, node_version,
		       pid, timestamp, url, created_at
		FROM logs %s
		ORDER BY timestamp DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := database.GetDB().Query(context.Background(), query, args...)
	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F70",
				Data:    []interface{}{fiber.Map{"error": "Error al obtener logs"}},
			},
		})
	}
	defer rows.Close()

	var logs []models.Log
	for rows.Next() {
		var log models.Log
		err := rows.Scan(
			&log.IDLog, &log.Method, &log.Path, &log.Protocol, &log.StatusCode,
			&log.ResponseTime, &log.UserAgent, &log.IP, &log.Hostname,
			&log.Body, &log.Params, &log.Query, &log.Email, &log.Username,
			&log.Role, &log.LogLevel, &log.Environment, &log.NodeVersion,
			&log.PID, &log.Timestamp, &log.URL, &log.CreatedAt,
		)
		if err != nil {
			continue
		}
		logs = append(logs, log)
	}

	return c.Status(200).JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S70",
			Data: []interface{}{fiber.Map{
				"logs":  logs,
				"total": total,
				"page":  page,
				"limit": limit,
			}},
		},
	})
}

// ObtenerEstadisticasLogs obtiene estadísticas de los logs
func ObtenerEstadisticasLogs(c *fiber.Ctx) error {
	// Solo admin puede ver estadísticas
	userRole := c.Locals("user_role").(string)
	if userRole != "admin" {
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F71",
				Data:    []interface{}{fiber.Map{"error": "Solo administradores pueden ver estadísticas"}},
			},
		})
	}

	// Estadísticas por nivel de log
	logLevelStats := make(map[string]int)
	rows, err := database.GetDB().Query(context.Background(), `
		SELECT log_level, COUNT(*) 
		FROM logs 
		WHERE timestamp >= NOW() - INTERVAL '24 hours'
		GROUP BY log_level
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var level string
			var count int
			rows.Scan(&level, &count)
			logLevelStats[level] = count
		}
	}

	// Estadísticas por método HTTP
	methodStats := make(map[string]int)
	rows, err = database.GetDB().Query(context.Background(), `
		SELECT method, COUNT(*) 
		FROM logs 
		WHERE timestamp >= NOW() - INTERVAL '24 hours'
		GROUP BY method
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var method string
			var count int
			rows.Scan(&method, &count)
			methodStats[method] = count
		}
	}

	// Estadísticas por código de estado
	statusStats := make(map[string]int)
	rows, err = database.GetDB().Query(context.Background(), `
		SELECT 
			CASE 
				WHEN status_code >= 200 AND status_code < 300 THEN 'success'
				WHEN status_code >= 300 AND status_code < 400 THEN 'redirect'
				WHEN status_code >= 400 AND status_code < 500 THEN 'client_error'
				WHEN status_code >= 500 THEN 'server_error'
				ELSE 'other'
			END as status_category,
			COUNT(*)
		FROM logs 
		WHERE timestamp >= NOW() - INTERVAL '24 hours'
		GROUP BY status_category
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var category string
			var count int
			rows.Scan(&category, &count)
			statusStats[category] = count
		}
	}

	// Top IPs
	var topIPs []fiber.Map
	rows, err = database.GetDB().Query(context.Background(), `
		SELECT ip, COUNT(*) as requests
		FROM logs 
		WHERE timestamp >= NOW() - INTERVAL '24 hours'
		GROUP BY ip
		ORDER BY requests DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ip string
			var requests int
			rows.Scan(&ip, &requests)
			topIPs = append(topIPs, fiber.Map{
				"ip":       ip,
				"requests": requests,
			})
		}
	}

	// Tiempo de respuesta promedio
	var avgResponseTime float64
	database.GetDB().QueryRow(context.Background(), `
		SELECT COALESCE(AVG(response_time), 0)
		FROM logs 
		WHERE timestamp >= NOW() - INTERVAL '24 hours' AND response_time IS NOT NULL
	`).Scan(&avgResponseTime)

	return c.Status(200).JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S71",
			Data: []interface{}{fiber.Map{
				"log_level_stats":   logLevelStats,
				"method_stats":      methodStats,
				"status_stats":      statusStats,
				"top_ips":           topIPs,
				"avg_response_time": avgResponseTime,
				"period":            "24 hours",
			}},
		},
	})
}

// LimpiarLogs elimina logs antiguos
func LimpiarLogs(c *fiber.Ctx) error {
	// Solo admin puede limpiar logs
	userRole := c.Locals("user_role").(string)
	if userRole != "admin" {
		return c.Status(403).JSON(StandardResponse{
			StatusCode: 403,
			Body: BodyResponse{
				IntCode: "F72",
				Data:    []interface{}{fiber.Map{"error": "Solo administradores pueden limpiar logs"}},
			},
		})
	}

	// Parámetro de días (por defecto 30 días)
	dias, _ := strconv.Atoi(c.Query("dias", "30"))
	if dias < 1 {
		dias = 30
	}

	// Eliminar logs más antiguos que X días
	result, err := database.GetDB().Exec(context.Background(), `
		DELETE FROM logs 
		WHERE timestamp < NOW() - INTERVAL '%d days'
	`, dias)

	if err != nil {
		return c.Status(500).JSON(StandardResponse{
			StatusCode: 500,
			Body: BodyResponse{
				IntCode: "F72",
				Data:    []interface{}{fiber.Map{"error": "Error al limpiar logs"}},
			},
		})
	}

	rowsAffected := result.RowsAffected()

	return c.Status(200).JSON(StandardResponse{
		StatusCode: 200,
		Body: BodyResponse{
			IntCode: "S72",
			Data: []interface{}{fiber.Map{
				"message":      "Logs limpiados exitosamente",
				"rows_deleted": rowsAffected,
				"days_deleted": dias,
			}},
		},
	})
}
