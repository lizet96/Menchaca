package handlers

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/lizet96/hospital-backend/database"
	"github.com/lizet96/hospital-backend/models"
)

// CrearConsultorio crea un nuevo consultorio
func CrearConsultorio(c *fiber.Ctx) error {
	// Solo admin puede crear consultorios
	userRole := c.Locals("user_role").(string)
	if userRole != "admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo administradores pueden crear consultorios",
		})
	}

	var consultorio models.Consultorio
	if err := c.BodyParser(&consultorio); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Validaciones
	if consultorio.NombreNumero == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "El nombre/número del consultorio es requerido",
		})
	}

	// Verificar que no exista un consultorio con el mismo nombre/número
	var existe bool
	err := database.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM Consultorio WHERE nombre_numero = $1)", consultorio.NombreNumero).Scan(&existe)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al verificar consultorio",
		})
	}

	if existe {
		return c.Status(409).JSON(fiber.Map{
			"error": "Ya existe un consultorio con ese nombre/número",
		})
	}

	// Insertar consultorio
	query := `INSERT INTO Consultorio (ubicacion, nombre_numero) VALUES ($1, $2) RETURNING id_consultorio`
	err = database.GetDB().QueryRow(context.Background(), query,
		consultorio.Ubicacion, consultorio.NombreNumero).Scan(&consultorio.IDConsultorio)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al crear el consultorio",
		})
	}

	return c.Status(201).JSON(fiber.Map{
		"consultorio": consultorio,
		"mensaje":     "Consultorio creado exitosamente",
	})
}

// ObtenerConsultorios obtiene todos los consultorios
func ObtenerConsultorios(c *fiber.Ctx) error {
	// Todos los usuarios autenticados pueden ver consultorios
	query := `SELECT id_consultorio, ubicacion, nombre_numero FROM Consultorio ORDER BY nombre_numero`

	rows, err := database.GetDB().Query(context.Background(), query)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener consultorios",
		})
	}
	defer rows.Close()

	var consultorios []models.Consultorio
	for rows.Next() {
		var consultorio models.Consultorio
		err := rows.Scan(&consultorio.IDConsultorio, &consultorio.Ubicacion, &consultorio.NombreNumero)
		if err != nil {
			continue
		}
		consultorios = append(consultorios, consultorio)
	}

	return c.JSON(fiber.Map{
		"consultorios": consultorios,
		"total":        len(consultorios),
	})
}

// ObtenerConsultorioPorID obtiene un consultorio específico por ID
func ObtenerConsultorioPorID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	var consultorio models.Consultorio
	query := `SELECT id_consultorio, ubicacion, nombre_numero FROM Consultorio WHERE id_consultorio = $1`

	err = database.GetDB().QueryRow(context.Background(), query, id).Scan(
		&consultorio.IDConsultorio, &consultorio.Ubicacion, &consultorio.NombreNumero)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Consultorio no encontrado",
		})
	}

	return c.JSON(fiber.Map{
		"consultorio": consultorio,
	})
}

// ActualizarConsultorio actualiza un consultorio existente
func ActualizarConsultorio(c *fiber.Ctx) error {
	// Solo admin puede actualizar consultorios
	userRole := c.Locals("user_role").(string)
	if userRole != "admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo administradores pueden actualizar consultorios",
		})
	}

	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Verificar que el consultorio existe
	var existe bool
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM Consultorio WHERE id_consultorio = $1)", id).Scan(&existe)
	if err != nil || !existe {
		return c.Status(404).JSON(fiber.Map{
			"error": "Consultorio no encontrado",
		})
	}

	var consultorioActualizado models.Consultorio
	if err := c.BodyParser(&consultorioActualizado); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Datos inválidos",
		})
	}

	// Validaciones
	if consultorioActualizado.NombreNumero == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "El nombre/número del consultorio es requerido",
		})
	}

	// Verificar que no exista otro consultorio con el mismo nombre/número
	var existeOtro bool
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM Consultorio WHERE nombre_numero = $1 AND id_consultorio != $2)",
		consultorioActualizado.NombreNumero, id).Scan(&existeOtro)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al verificar consultorio",
		})
	}

	if existeOtro {
		return c.Status(409).JSON(fiber.Map{
			"error": "Ya existe otro consultorio con ese nombre/número",
		})
	}

	// Actualizar consultorio
	query := `UPDATE Consultorio SET ubicacion = $1, nombre_numero = $2 WHERE id_consultorio = $3`
	_, err = database.GetDB().Exec(context.Background(), query,
		consultorioActualizado.Ubicacion, consultorioActualizado.NombreNumero, id)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al actualizar el consultorio",
		})
	}

	return c.JSON(fiber.Map{
		"mensaje": "Consultorio actualizado exitosamente",
	})
}

// EliminarConsultorio elimina un consultorio
func EliminarConsultorio(c *fiber.Ctx) error {
	// Solo admin puede eliminar consultorios
	userRole := c.Locals("user_role").(string)
	if userRole != "admin" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Solo administradores pueden eliminar consultorios",
		})
	}

	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Verificar si el consultorio tiene horarios asociados
	var tieneHorarios bool
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM Horario WHERE id_consultorio = $1)", id).Scan(&tieneHorarios)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al verificar horarios asociados",
		})
	}

	if tieneHorarios {
		return c.Status(409).JSON(fiber.Map{
			"error": "No se puede eliminar el consultorio porque tiene horarios asociados",
		})
	}

	// Verificar si el consultorio tiene recetas asociadas
	var tieneRecetas bool
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM Receta WHERE id_consultorio = $1)", id).Scan(&tieneRecetas)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al verificar recetas asociadas",
		})
	}

	if tieneRecetas {
		return c.Status(409).JSON(fiber.Map{
			"error": "No se puede eliminar el consultorio porque tiene recetas asociadas",
		})
	}

	// Eliminar consultorio
	result, err := database.GetDB().Exec(context.Background(),
		"DELETE FROM Consultorio WHERE id_consultorio = $1", id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al eliminar el consultorio",
		})
	}

	if result.RowsAffected() == 0 {
		return c.Status(404).JSON(fiber.Map{
			"error": "Consultorio no encontrado",
		})
	}

	return c.JSON(fiber.Map{
		"mensaje": "Consultorio eliminado exitosamente",
	})
}

// ObtenerConsultoriosDisponibles obtiene consultorios con horarios disponibles
func ObtenerConsultoriosDisponibles(c *fiber.Ctx) error {
	// Obtener consultorios que tienen al menos un horario disponible
	query := `SELECT DISTINCT c.id_consultorio, c.ubicacion, c.nombre_numero
			  FROM Consultorio c
			  INNER JOIN Horario h ON c.id_consultorio = h.id_consultorio
			  WHERE h.consulta_disponible = true
			  ORDER BY c.nombre_numero`

	rows, err := database.GetDB().Query(context.Background(), query)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener consultorios disponibles",
		})
	}
	defer rows.Close()

	var consultorios []models.Consultorio
	for rows.Next() {
		var consultorio models.Consultorio
		err := rows.Scan(&consultorio.IDConsultorio, &consultorio.Ubicacion, &consultorio.NombreNumero)
		if err != nil {
			continue
		}
		consultorios = append(consultorios, consultorio)
	}

	return c.JSON(fiber.Map{
		"consultorios_disponibles": consultorios,
		"total":                    len(consultorios),
	})
}

// ObtenerHorariosPorConsultorio obtiene todos los horarios de un consultorio específico
func ObtenerHorariosPorConsultorio(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "ID inválido",
		})
	}

	// Verificar que el consultorio existe
	var existe bool
	err = database.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM Consultorio WHERE id_consultorio = $1)", id).Scan(&existe)
	if err != nil || !existe {
		return c.Status(404).JSON(fiber.Map{
			"error": "Consultorio no encontrado",
		})
	}

	// Obtener horarios del consultorio
	query := `SELECT h.id_horario, h.turno, h.id_medico, h.id_consultorio, h.consulta_disponible,
			  u.nombre as medico_nombre
			  FROM Horario h
			  JOIN Usuario u ON h.id_medico = u.id_usuario
			  WHERE h.id_consultorio = $1
			  ORDER BY h.turno`

	rows, err := database.GetDB().Query(context.Background(), query, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Error al obtener horarios del consultorio",
		})
	}
	defer rows.Close()

	type HorarioDetalle struct {
		models.Horario
		MedicoNombre string `json:"medico_nombre"`
	}

	var horarios []HorarioDetalle
	for rows.Next() {
		var horario HorarioDetalle
		err := rows.Scan(
			&horario.IDHorario, &horario.Turno, &horario.IDMedico,
			&horario.IDConsultorio, &horario.ConsultaDisponible, &horario.MedicoNombre,
		)
		if err != nil {
			continue
		}
		horarios = append(horarios, horario)
	}

	return c.JSON(fiber.Map{
		"horarios":       horarios,
		"total":          len(horarios),
		"consultorio_id": id,
	})
}
