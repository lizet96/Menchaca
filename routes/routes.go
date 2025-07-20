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
	app.Use(middleware.SecurityHeaders())    // Headers de seguridad
	app.Use(middleware.DefaultRateLimiter()) // Rate limiting general
	app.Use(middleware.LoggingMiddleware())  // Logging de auditoría
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
	// Aplicar rate limiting estricto para autenticación
	auth.Use(middleware.AuthRateLimiter())
	auth.Post("/register", middleware.BodySizeLimit(1024*1024), handlers.RegistrarUsuario) // 1MB límite
	auth.Post("/login", middleware.BodySizeLimit(512*1024), handlers.Login)                // 512KB límite
	auth.Post("/refresh", middleware.BodySizeLimit(256*1024), handlers.RefreshToken)       // 256KB límite
	auth.Post("/logout", middleware.JWTMiddlewareOptional(), handlers.Logout)

	// === RUTAS PROTEGIDAS (Requieren autenticación) ===
	protected := api.Group("/", middleware.JWTMiddleware())

	// --- RUTAS DE USUARIOS ---
	usuarios := protected.Group("/usuarios")
	usuarios.Get("/", middleware.RequirePermission("usuarios_read"), handlers.ObtenerUsuarios)
	usuarios.Post("/", middleware.RequirePermission("usuarios_create"), handlers.CrearUsuario)
	usuarios.Get("/perfil", handlers.ObtenerPerfil)
	usuarios.Get("/:id", middleware.RequirePermission("usuarios_read"), handlers.ObtenerUsuarioPorID)
	usuarios.Put("/:id", middleware.RequirePermission("usuarios_update"), handlers.ActualizarUsuario)
	usuarios.Delete("/:id", middleware.RequirePermission("usuarios_delete"), handlers.EliminarUsuario)
	usuarios.Get("/role/:id", middleware.RequirePermission("usuarios_read"), handlers.ObtenerUsuariosPorRol)

	// --- RUTAS ADMINISTRATIVAS ---
	admin := protected.Group("/admin")
	// Aplicar rate limiting estricto para operaciones administrativas
	admin.Use(middleware.StrictRateLimiter())
	admin.Post("/limpiar-sesiones", middleware.RequirePermission("usuarios_delete"), handlers.LimpiarTodasLasSesiones)

	// --- RUTAS DE PACIENTES ---
	pacientes := protected.Group("/pacientes")
	pacientes.Get("/", middleware.RequirePermission("usuarios_read"), handlers.ObtenerPacientes)

	// --- RUTAS DE ROLES Y PERMISOS ---
	roles := protected.Group("/roles")
	roles.Get("/", handlers.ObtenerRoles)
	roles.Get("/:id/permisos", handlers.ObtenerPermisosPorRol)

	// --- RUTAS MFA ---
	mfa := protected.Group("/mfa")
	// Aplicar rate limiting estricto para MFA
	mfa.Use(middleware.StrictRateLimiter())
	mfa.Post("/setup", middleware.BodySizeLimit(256*1024), handlers.SetupMFA)     // 256KB límite
	mfa.Post("/verify", middleware.BodySizeLimit(128*1024), handlers.VerifyMFA)   // 128KB límite
	mfa.Post("/disable", middleware.BodySizeLimit(128*1024), handlers.DisableMFA) // 128KB límite

	// --- RUTAS DE EXPEDIENTES ---
	expedientes := protected.Group("/expedientes")
	expedientes.Post("/", middleware.RequirePermission("expedientes_create"), handlers.CrearExpediente)
	expedientes.Get("/", middleware.RequirePermission("expedientes_read"), handlers.ObtenerExpedientes)
	expedientes.Get("/:id", middleware.RequirePermission("expedientes_read"), handlers.ObtenerExpedientePorID)
	expedientes.Put("/:id", middleware.RequirePermission("expedientes_update"), handlers.ActualizarExpediente)
	expedientes.Delete("/:id", middleware.RequirePermission("expedientes_delete"), handlers.EliminarExpediente)
	expedientes.Get("/paciente/:paciente_id", middleware.RequirePermission("expedientes_read"), handlers.ObtenerExpedientePorPaciente)

	// --- RUTAS DE CONSULTAS ---
	consultas := protected.Group("/consultas")
	consultas.Post("/", middleware.RequirePermission("consultas_create"), handlers.CrearConsulta)
	consultas.Get("/", middleware.RequirePermission("consultas_read"), handlers.ObtenerConsultas)
	consultas.Get("/:id", middleware.RequirePermission("consultas_read"), handlers.ObtenerConsultaPorID)
	consultas.Put("/:id", middleware.RequirePermission("consultas_update"), handlers.ActualizarConsulta)
	consultas.Delete("/:id", middleware.RequirePermission("consultas_delete"), handlers.EliminarConsulta)
	consultas.Put("/:id/cancelar", middleware.RequirePermission("consultas_update"), handlers.CancelarConsulta)
	consultas.Get("/paciente/:paciente_id", middleware.RequirePermission("consultas_read"), handlers.ObtenerConsultasPorPaciente)
	consultas.Get("/medico/:medico_id", middleware.RequirePermission("consultas_read"), handlers.ObtenerConsultasPorMedico)
	consultas.Put("/:id/completar", middleware.RequirePermission("consultas_update"), handlers.CompletarConsulta)

	// --- RUTAS DE RECETAS ---
	recetas := protected.Group("/recetas")
	recetas.Post("/", middleware.RequirePermission("recetas_create"), handlers.CrearReceta)
	recetas.Get("/", middleware.RequirePermission("recetas_read"), handlers.ObtenerRecetas)
	recetas.Get("/:id", middleware.RequirePermission("recetas_read"), handlers.ObtenerRecetaPorID)
	recetas.Put("/:id", middleware.RequirePermission("recetas_update"), handlers.ActualizarReceta)
	recetas.Delete("/:id", middleware.RequirePermission("recetas_delete"), handlers.EliminarReceta)
	recetas.Get("/paciente/:paciente_id", middleware.RequirePermission("recetas_read"), handlers.ObtenerRecetasPorPaciente)

	// --- RUTAS DE CONSULTORIOS ---
	consultorios := protected.Group("/consultorios")
	consultorios.Post("/", middleware.RequirePermission("consultorios_create"), handlers.CrearConsultorio)
	consultorios.Get("/", middleware.RequirePermission("consultorios_read"), handlers.ObtenerConsultorios)
	consultorios.Get("/:id", middleware.RequirePermission("consultorios_read"), handlers.ObtenerConsultorioPorID)
	consultorios.Put("/:id", middleware.RequirePermission("consultorios_update"), handlers.ActualizarConsultorio)
	consultorios.Delete("/:id", middleware.RequirePermission("consultorios_delete"), handlers.EliminarConsultorio)

	// --- RUTAS DE REPORTES ---
	reportes := protected.Group("/reportes")
	reportes.Get("/consultas", middleware.RequirePermission("reportes_read"), handlers.GenerarReporteConsultas)
	reportes.Get("/usuarios", middleware.RequirePermission("reportes_read"), handlers.GenerarReporteUsuarios)
	reportes.Get("/expedientes", middleware.RequirePermission("reportes_read"), handlers.GenerarReporteExpedientes)

	// --- RUTAS DE HORARIOS ---
	horarios := protected.Group("/horarios")
	horarios.Post("/", middleware.RequirePermission("horarios_create"), handlers.CrearHorario)
	horarios.Get("/", middleware.RequirePermission("horarios_read"), handlers.ObtenerHorarios)
	// Rutas específicas ANTES de las rutas con parámetros
	horarios.Get("/disponibles", middleware.RequirePermission("horarios_read"), handlers.ObtenerHorariosDisponibles)
	horarios.Get("/medico/:medico_id", middleware.RequirePermission("horarios_read"), handlers.ObtenerHorariosPorMedico)
	// Rutas con parámetros ID al final
	horarios.Get("/:id", middleware.RequirePermission("horarios_read"), handlers.ObtenerHorarioPorID)
	horarios.Put("/:id", middleware.RequirePermission("horarios_update"), handlers.ActualizarHorario)
	horarios.Delete("/:id", middleware.RequirePermission("horarios_delete"), handlers.EliminarHorario)
	horarios.Put("/:id/disponibilidad", middleware.RequirePermission("horarios_update"), handlers.CambiarDisponibilidadHorario)

	// --- RUTAS DE LOGS (Solo para administradores) ---
	logs := protected.Group("/logs")
	logs.Use(middleware.StrictRateLimiter()) // Rate limiting estricto para logs
	logs.Get("/", middleware.RequirePermission("logs_read"), handlers.ObtenerLogs)
	logs.Get("/estadisticas", middleware.RequirePermission("logs_read"), handlers.ObtenerEstadisticasLogs)
	logs.Delete("/limpiar", middleware.RequirePermission("logs_delete"), handlers.LimpiarLogs)
}
