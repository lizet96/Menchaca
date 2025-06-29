package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/lizet96/hospital-backend/database"
	"github.com/lizet96/hospital-backend/routes"
)

func main() {
	// Cargar variables de entorno
	if err := godotenv.Load(); err != nil {
		log.Println("Advertencia: No se pudo cargar el archivo .env")
	}
	// Conectar a la base de datos
	database.ConnectDB()
	defer database.CloseDB()
	log.Println("Conexión a la base de datos establecida")
	// Crear instancia de Fiber con configuración
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error":   true,
				"message": err.Error(),
			})
		},
		AppName: "Hospital Management System API v1.0.0",
	})

	// Configurar rutas
	routes.SetupRoutes(app)

	
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(404).JSON(fiber.Map{
			"error":   "Ruta no encontrada",
			"message": "La ruta solicitada no existe en este servidor",
			"path":    c.Path(),
			"method":  c.Method(),
		})
	})

	// Obtener puerto del entorno o usar 3000 por defecto
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	// Iniciar servidor
	log.Printf(" Servidor Hospital Management System iniciado en puerto %s", port)
	log.Printf(" Documentación de rutas disponible en: http://localhost:%s/routes", port)
	log.Printf(" Estado del sistema: http://localhost:%s/health", port)
	log.Fatal(app.Listen(":" + port))
}
