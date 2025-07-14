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
	auth.Post("/login", handlers.Login) // ← Cambiar de LoginWithMFA a Login
	auth.Post("/refresh", handlers.RefreshToken)
	auth.Post("/logout", middleware.JWTMiddleware(), handlers.Logout)

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

	// --- RUTAS DE PACIENTES ---
	pacientes := protected.Group("/pacientes")
	pacientes.Get("/", middleware.RequirePermission("usuarios_read"), handlers.ObtenerPacientes)

	// --- RUTAS DE ROLES Y PERMISOS ---
	roles := protected.Group("/roles")
	roles.Get("/:id/permisos", handlers.ObtenerPermisosPorRol)

	// --- RUTAS MFA ---
	mfa := protected.Group("/mfa")
	mfa.Post("/setup", handlers.SetupMFA)
	mfa.Post("/verify", handlers.VerifyMFA)
	mfa.Post("/disable", handlers.DisableMFA)

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
	consultas.Delete("/:id", middleware.RequirePermission("consultas_delete"), handlers.CancelarConsulta)
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
	horarios.Get("/:id", middleware.RequirePermission("horarios_read"), handlers.ObtenerHorarioPorID)
	horarios.Put("/:id", middleware.RequirePermission("horarios_update"), handlers.ActualizarHorario)
	horarios.Delete("/:id", middleware.RequirePermission("horarios_delete"), handlers.EliminarHorario)
	horarios.Get("/medico/:medico_id", middleware.RequirePermission("horarios_read"), handlers.ObtenerHorariosPorMedico)
	horarios.Put("/:id/disponibilidad", middleware.RequirePermission("horarios_update"), handlers.CambiarDisponibilidadHorario)
	horarios.Get("/disponibles", middleware.RequirePermission("horarios_read"), handlers.ObtenerHorariosDisponibles)
}
