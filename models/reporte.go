package models

import (
	"time"
)

// ReporteConsultas representa un reporte de consultas
type ReporteConsultas struct {
	TotalConsultas     int     `json:"total_consultas"`
	ConsultasHoy       int     `json:"consultas_hoy"`
	ConsultasSemana    int     `json:"consultas_semana"`
	IngresosTotales    float64 `json:"ingresos_totales"`
	PromedioConsultas  float64 `json:"promedio_consultas"`
	FechaGeneracion    time.Time `json:"fecha_generacion"`
}