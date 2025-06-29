package database

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB es la instancia global del pool de conexiones
var DB *pgxpool.Pool

// ConnectDB establece la conexión con la base de datos usando un pool
func ConnectDB() {
	// 📦 Leer la variable de entorno DATABASE_URL (que contiene la cadena de conexión a PostgreSQL)
	config, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("❌ Error al parsear la URL de la base de datos: %v", err)
	}
	config.MaxConns = 30 // Número máximo de conexiones abiertas al mismo tiempo
	config.MinConns = 5  // Número mínimo de conexiones que se mantienen abiertas en espera
	config.MaxConnLifetime = time.Hour // Tiempo máximo que puede vivir una conexión antes de ser cerrada

	// Tiempo máximo que una conexión puede estar inactiva 
	config.MaxConnIdleTime = time.Minute * 30
	// Cambiar el modo de ejecución de queries a "Simple Protocol"
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	// Crear el pool de conexiones usando la configuración anterior
	DB, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("❌ Error al crear el pool de conexiones: %v", err)
	}
	//  Probar si la base de datos está viva haciendo una consulta rápida
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // Nos aseguramos de cancelar el contexto aunque todo salga bien

	var version string
	err = DB.QueryRow(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		log.Fatalf("❌ Error al probar la conexión: %v", err)
	}

	// se imprime la versión del motor de base de datos como confirmación
	log.Println("✅ Conectado exitosamente a la base de datos:", version)
}

// CloseDB cierra el pool de conexiones
func CloseDB() {
	if DB != nil {
		DB.Close()
		log.Println("Pool de conexiones cerrado")
	}
}

// GetDB retorna la instancia del pool de conexiones
func GetDB() *pgxpool.Pool {
	return DB
}
