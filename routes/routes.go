package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/lizet96/hospital-backend/handlers"
	"github.com/lizet96/hospital-backend/middleware"
)

// SetupRoutes configura todas las rutas de la aplicación
func SetupRoutes(app *fiber.App) {
	// Middleware global
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Ruta de salud del sistema
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Hospital Management System API",
			"version": "1.0.0",
		})
	})

	// Grupo de API
	api := app.Group("/api/v1")

	// === RUTAS PÚBLICAS (Sin autenticación) ===
	auth := api.Group("/auth")
	auth.Post("/register", handlers.RegistrarUsuario)
	auth.Post("/login", handlers.Login)

	// === RUTAS PROTEGIDAS (Requieren autenticación) ===
	protected := api.Group("/", middleware.JWTMiddleware())

	// --- RUTAS DE USUARIOS ---
	usuarios := protected.Group("/usuarios")
	usuarios.Get("/", middleware.RequireRole("admin"), handlers.ObtenerUsuarios)
	usuarios.Get("/perfil", handlers.ObtenerPerfil)
	usuarios.Get("/:id", middleware.RequireRole("admin", "medico", "enfermera"), handlers.ObtenerUsuarioPorID)
	usuarios.Put("/:id", middleware.RequireRole("admin"), handlers.ActualizarUsuario)
	usuarios.Delete("/:id", middleware.RequireRole("admin"), handlers.EliminarUsuario)

	// --- RUTAS DE EXPEDIENTES ---
	expedientes := protected.Group("/expedientes")
	expedientes.Post("/", middleware.RequireRole("admin", "medico", "enfermera"), handlers.CrearExpediente)
	expedientes.Get("/", middleware.RequireRole("admin", "medico", "enfermera"), handlers.ObtenerExpedientes)
	expedientes.Get("/:id", middleware.RequireRole("admin", "medico", "enfermera", "paciente"), handlers.ObtenerExpedientePorID)
	expedientes.Put("/:id", middleware.RequireRole("admin", "medico", "enfermera"), handlers.ActualizarExpediente)
	expedientes.Delete("/:id", middleware.RequireRole("admin"), handlers.EliminarExpediente)
	expedientes.Get("/paciente/:paciente_id", middleware.RequireRole("admin", "medico", "enfermera", "paciente"), handlers.ObtenerExpedientePorPaciente)

	// --- RUTAS DE CONSULTAS ---
	consultas := protected.Group("/consultas")
	consultas.Post("/", middleware.RequireRole("admin", "medico", "enfermera", "paciente"), handlers.CrearConsulta)
	consultas.Get("/", handlers.ObtenerConsultas) // Filtrado por rol en el handler
	consultas.Get("/:id", handlers.ObtenerConsultaPorID) // Filtrado por rol en el handler
	consultas.Put("/:id", middleware.RequireRole("admin", "medico", "enfermera"), handlers.ActualizarConsulta)
	consultas.Delete("/:id", middleware.RequireRole("admin", "medico"), handlers.CancelarConsulta)
	consultas.Get("/paciente/:paciente_id", handlers.ObtenerConsultasPorPaciente) // Filtrado por rol en el handler
	consultas.Get("/medico/:medico_id", middleware.RequireRole("admin", "medico", "enfermera"), handlers.ObtenerConsultasPorMedico)
	consultas.Put("/:id/completar", middleware.RequireRole("medico"), handlers.CompletarConsulta)

	// --- RUTAS DE RECETAS ---
	recetas := protected.Group("/recetas")
	recetas.Post("/", middleware.RequireRole("medico"), handlers.CrearReceta)
	recetas.Get("/", handlers.ObtenerRecetas) // Filtrado por rol en el handler
	recetas.Get("/:id", handlers.ObtenerRecetaPorID) // Filtrado por rol en el handler
	recetas.Put("/:id", middleware.RequireRole("medico"), handlers.ActualizarReceta)
	recetas.Delete("/:id", middleware.RequireRole("admin", "medico"), handlers.EliminarReceta)
	recetas.Get("/paciente/:paciente_id", handlers.ObtenerRecetasPorPaciente) // Filtrado por rol en el handler

	// --- RUTAS DE CONSULTORIOS ---
	consultorios := protected.Group("/consultorios")
	consultorios.Post("/", middleware.RequireRole("admin"), handlers.CrearConsultorio)
	consultorios.Get("/", handlers.ObtenerConsultorios) // Todos pueden ver
	consultorios.Get("/disponibles", handlers.ObtenerConsultoriosDisponibles)
	consultorios.Get("/:id", handlers.ObtenerConsultorioPorID)
	consultorios.Put("/:id", middleware.RequireRole("admin"), handlers.ActualizarConsultorio)
	consultorios.Delete("/:id", middleware.RequireRole("admin"), handlers.EliminarConsultorio)
	consultorios.Get("/:id/horarios", handlers.ObtenerHorariosPorConsultorio)

	// --- RUTAS DE HORARIOS ---
	horarios := protected.Group("/horarios")
	horarios.Post("/", middleware.RequireRole("admin"), handlers.CrearHorario)
	horarios.Get("/", handlers.ObtenerHorarios) // Filtrado por rol en el handler
	horarios.Get("/disponibles", handlers.ObtenerHorariosDisponibles)
	horarios.Get("/:id", handlers.ObtenerHorarioPorID) // Filtrado por rol en el handler
	horarios.Put("/:id", middleware.RequireRole("admin"), handlers.ActualizarHorario)
	horarios.Delete("/:id", middleware.RequireRole("admin"), handlers.EliminarHorario)
	horarios.Put("/:id/disponibilidad", middleware.RequireRole("admin", "medico"), handlers.CambiarDisponibilidadHorario)
	horarios.Get("/medico/:medico_id", handlers.ObtenerHorariosPorMedico) // Filtrado por rol en el handler

	// --- RUTAS DE REPORTES ---
	reportes := protected.Group("/reportes")
	reportes.Get("/consultas", middleware.RequireRole("admin", "medico"), handlers.GenerarReporteConsultas)
	reportes.Get("/estadisticas", middleware.RequireRole("admin"), handlers.ObtenerEstadisticasGenerales)
	reportes.Get("/pacientes", middleware.RequireRole("admin", "medico"), handlers.ObtenerReportePacientes)
	reportes.Get("/ingresos", middleware.RequireRole("admin"), handlers.ObtenerReporteIngresos)

	// === RUTAS DE ADMINISTRACIÓN ===
	admin := protected.Group("/admin", middleware.RequireRole("admin"))

	// Gestión avanzada de usuarios
	admin.Get("/usuarios/estadisticas", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"mensaje": "Estadísticas de usuarios - Funcionalidad por implementar",
		})
	})

	// Configuración del sistema
	admin.Get("/configuracion", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"mensaje": "Configuración del sistema - Funcionalidad por implementar",
		})
	})

	// Logs del sistema
	admin.Get("/logs", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"mensaje": "Logs del sistema - Funcionalidad por implementar",
		})
	})
}
