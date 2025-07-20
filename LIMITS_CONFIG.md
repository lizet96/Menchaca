# Configuración de Límites y Seguridad - Hospital Backend

## 📋 Resumen

Este documento describe la configuración de límites de lectura, rate limiting y medidas de seguridad implementadas en el sistema de gestión hospitalaria.

## 🔧 Configuraciones Implementadas

### 1. Límites del Servidor (main.go)

```go
app := fiber.New(fiber.Config{
    BodyLimit:         4 * 1024 * 1024, // 4MB límite para el cuerpo de la petición
    ReadTimeout:       30 * time.Second, // 30 segundos para leer la petición
    WriteTimeout:      30 * time.Second, // 30 segundos para escribir la respuesta
    IdleTimeout:       120 * time.Second, // 2 minutos de timeout para conexiones inactivas
    ReadBufferSize:    4096,             // 4KB buffer de lectura
    WriteBufferSize:   4096,             // 4KB buffer de escritura
    ServerHeader:      "Hospital-API",   // Header personalizado del servidor
    DisableKeepalive:  false,            // Mantener conexiones keep-alive habilitadas
    ReduceMemoryUsage: true,             // Reducir uso de memoria
})
```

### 2. Rate Limiting por Categoría

#### Rate Limiting General
- **Límite**: 100 requests por 15 minutos
- **Aplicado a**: Todas las rutas
- **Propósito**: Prevenir abuso general del sistema

#### Rate Limiting de Autenticación
- **Límite**: 5 requests por 15 minutos
- **Aplicado a**: `/auth/login`, `/auth/register`, `/auth/refresh`
- **Propósito**: Prevenir ataques de fuerza bruta

#### Rate Limiting Estricto
- **Límite**: 10 requests por 15 minutos
- **Aplicado a**: Rutas administrativas y MFA
- **Propósito**: Proteger operaciones sensibles

### 3. Límites de Tamaño por Endpoint

| Endpoint | Límite | Propósito |
|----------|--------|-----------|
| `/auth/register` | 1MB | Registro de usuarios |
| `/auth/login` | 512KB | Inicio de sesión |
| `/auth/refresh` | 256KB | Renovación de tokens |
| `/mfa/setup` | 256KB | Configuración MFA |
| `/mfa/verify` | 128KB | Verificación MFA |
| `/mfa/disable` | 128KB | Desactivación MFA |

### 4. Headers de Seguridad

Se aplican automáticamente a todas las respuestas:

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Content-Security-Policy: default-src 'self'
Strict-Transport-Security: max-age=31536000; includeSubDomains
```

## 🛡️ Beneficios de Seguridad

### Protección contra Ataques
1. **DDoS**: Rate limiting previene sobrecarga del servidor
2. **Fuerza Bruta**: Límites estrictos en autenticación
3. **Payload Bombing**: Límites de tamaño de petición
4. **Slowloris**: Timeouts de lectura y escritura
5. **XSS/Clickjacking**: Headers de seguridad

### Optimización de Recursos
1. **Memoria**: `ReduceMemoryUsage` y buffers optimizados
2. **Conexiones**: Timeouts apropiados para liberar recursos
3. **CPU**: Rate limiting reduce carga de procesamiento

## 📊 Monitoreo y Logs

### Respuestas de Rate Limiting
```json
{
  "error": true,
  "message": "Demasiadas peticiones, intenta más tarde",
  "retry_after": 900
}
```

### Respuestas de Límite de Tamaño
```json
{
  "error": true,
  "message": "El tamaño de la petición excede el límite permitido",
  "max_size": 1048576
}
```

## ⚙️ Configuración Personalizada

### Modificar Rate Limits

En `middleware/limits.go`, puedes ajustar las configuraciones:

```go
// Ejemplo: Rate limit más estricto
var CustomRateLimit = RateLimitConfig{
    Max:        5,                      // 5 requests
    Expiration: 10 * time.Minute,      // por 10 minutos
    Message:    "Límite personalizado excedido",
}
```

### Aplicar a Rutas Específicas

```go
// En routes.go
api.Use("/endpoint-sensible", middleware.CreateRateLimiter(CustomRateLimit))
```

## 🔍 Troubleshooting

### Problemas Comunes

1. **"Demasiadas peticiones"**
   - Esperar el tiempo indicado en `retry_after`
   - Verificar si la aplicación está haciendo requests excesivos

2. **"Tamaño de petición excedido"**
   - Reducir el tamaño de los datos enviados
   - Verificar si es necesario aumentar el límite para casos específicos

3. **Timeouts de conexión**
   - Verificar la latencia de red
   - Optimizar consultas de base de datos

### Logs Útiles

```bash
# Ver logs del servidor
tail -f logs/server.log

# Filtrar por rate limiting
grep "rate limit" logs/server.log
```

## 📈 Recomendaciones de Producción

1. **Monitoreo**: Implementar alertas para rate limiting frecuente
2. **Métricas**: Trackear patrones de uso para ajustar límites
3. **Whitelist**: Considerar IPs confiables para límites más altos
4. **Balanceador**: Usar load balancer para distribuir carga
5. **CDN**: Implementar CDN para contenido estático

## 🔄 Actualizaciones

- **v1.0**: Implementación inicial de límites básicos
- **v1.1**: Rate limiting por categoría de endpoint
- **v1.2**: Headers de seguridad y optimizaciones de memoria

---

**Nota**: Esta configuración está optimizada para un entorno hospitalario donde la seguridad y estabilidad son prioritarias. Los límites pueden ajustarse según las necesidades específicas del sistema.