# Sistema de Gestión Hospitalaria - API Backend

## 📋 Descripción

Sistema completo de gestión hospitalaria desarrollado en Go con Fiber, que incluye gestión de citas médicas, expedientes, recetas, consultorios y reportes. El sistema implementa autenticación JWT y control de acceso basado en roles (RBAC) para garantizar la seguridad y privacidad de la información médica.

## 🏗️ Arquitectura del Sistema

### Roles de Usuario
- **Admin**: Acceso completo al sistema
- **Médico**: Gestión de consultas, expedientes y recetas
- **Enfermera**: Asistencia en gestión de citas y visualización de información
- **Paciente**: Solicitud de citas y acceso a su información médica

### Características de Seguridad
- ✅ Autenticación JWT
- ✅ Control de acceso basado en roles (RBAC)
- ✅ Encriptación de contraseñas con bcrypt
- ✅ Middleware de autorización
- ✅ Validación de datos de entrada
- ✅ Protección CORS
- ✅ Logging de actividades
- ✅ Recuperación de errores

## 🚀 Instalación y Configuración

### Prerrequisitos
- Go 1.21 o superior
- PostgreSQL 12 o superior
- Git

### 1. Clonar el repositorio
```bash
git clone <repository-url>
cd hospital-backend
```

### 2. Instalar dependencias
```bash
go mod download
```

### 3. Configurar base de datos

Ejecutar el siguiente script SQL en PostgreSQL:

```sql
CREATE DATABASE Mechaca;

-- Crear tabla de usuarios
CREATE TABLE Usuario (
    id_usuario SERIAL PRIMARY KEY,
    nombre VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    tipo VARCHAR(20) CHECK (tipo IN ('paciente', 'medico', 'enfermera', 'admin')) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Crear tabla de expedientes
CREATE TABLE Expediente (
    id_expediente SERIAL PRIMARY KEY,
    antecedentes TEXT,
    historial_clinico TEXT,
    seguro VARCHAR(100),
    id_paciente INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (id_paciente) REFERENCES Usuario(id_usuario)
);

-- Crear tabla de consultorios
CREATE TABLE Consultorio (
    id_consultorio SERIAL PRIMARY KEY,
    ubicacion VARCHAR(100),
    nombre_numero VARCHAR(50) UNIQUE NOT NULL
);

-- Crear tabla de horarios
CREATE TABLE Horario (
    id_horario SERIAL PRIMARY KEY,
    turno VARCHAR(50) NOT NULL,
    id_medico INT,
    id_consultorio INT,
    consulta_disponible BOOLEAN DEFAULT TRUE,
    FOREIGN KEY (id_medico) REFERENCES Usuario(id_usuario),
    FOREIGN KEY (id_consultorio) REFERENCES Consultorio(id_consultorio)
);

-- Crear tabla de consultas
CREATE TABLE Consulta (
    id_consulta SERIAL PRIMARY KEY,
    tipo VARCHAR(50),
    diagnostico TEXT,
    costo DECIMAL(10, 2),
    estado VARCHAR(20) DEFAULT 'programada',
    fecha TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    id_paciente INT,
    id_medico INT,
    id_horario INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (id_paciente) REFERENCES Usuario(id_usuario),
    FOREIGN KEY (id_medico) REFERENCES Usuario(id_usuario),
    FOREIGN KEY (id_horario) REFERENCES Horario(id_horario)
);

-- Crear tabla de recetas
CREATE TABLE Receta (
    id_receta SERIAL PRIMARY KEY,
    fecha DATE DEFAULT CURRENT_DATE,
    medicamento VARCHAR(255) NOT NULL,
    dosis VARCHAR(100) NOT NULL,
    id_medico INT,
    id_paciente INT,
    id_consultorio INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (id_medico) REFERENCES Usuario(id_usuario),
    FOREIGN KEY (id_paciente) REFERENCES Usuario(id_usuario),
    FOREIGN KEY (id_consultorio) REFERENCES Consultorio(id_consultorio)
);
```

### 4. Configurar variables de entorno

Crear archivo `.env` en la raíz del proyecto:

```env
# Base de datos
DATABASE_URL=postgres://usuario:password@localhost:5432/Mechaca

# JWT
JWT_SECRET=tu_clave_secreta_muy_segura_aqui

# Servidor
PORT=3000

# Entorno
ENVIRONMENT=development
```

### 5. Ejecutar el servidor
```bash
go run main.go
```

El servidor estará disponible en: `http://localhost:3000`

## 📚 Documentación de la API

### Rutas Públicas

#### Autenticación
- `POST /api/v1/auth/register` - Registrar nuevo usuario
- `POST /api/v1/auth/login` - Iniciar sesión

#### Sistema
- `GET /health` - Estado del sistema
- `GET /routes` - Documentación de rutas

### Rutas Protegidas (Requieren JWT)

#### Usuarios
- `GET /api/v1/usuarios` - Obtener todos los usuarios (admin)
- `GET /api/v1/usuarios/perfil` - Obtener perfil propio
- `GET /api/v1/usuarios/:id` - Obtener usuario por ID
- `PUT /api/v1/usuarios/:id` - Actualizar usuario (admin)
- `DELETE /api/v1/usuarios/:id` - Eliminar usuario (admin)

#### Expedientes
- `POST /api/v1/expedientes` - Crear expediente
- `GET /api/v1/expedientes` - Obtener expedientes
- `GET /api/v1/expedientes/:id` - Obtener expediente por ID
- `PUT /api/v1/expedientes/:id` - Actualizar expediente
- `DELETE /api/v1/expedientes/:id` - Eliminar expediente (admin)
- `GET /api/v1/expedientes/paciente/:paciente_id` - Expedientes por paciente

#### Consultas
- `POST /api/v1/consultas` - Crear consulta
- `GET /api/v1/consultas` - Obtener consultas
- `GET /api/v1/consultas/:id` - Obtener consulta por ID
- `PUT /api/v1/consultas/:id` - Actualizar consulta
- `DELETE /api/v1/consultas/:id` - Cancelar consulta
- `GET /api/v1/consultas/paciente/:paciente_id` - Consultas por paciente
- `GET /api/v1/consultas/medico/:medico_id` - Consultas por médico
- `PUT /api/v1/consultas/:id/completar` - Completar consulta (médico)

#### Recetas
- `POST /api/v1/recetas` - Crear receta (médico)
- `GET /api/v1/recetas` - Obtener recetas
- `GET /api/v1/recetas/:id` - Obtener receta por ID
- `PUT /api/v1/recetas/:id` - Actualizar receta (médico)
- `DELETE /api/v1/recetas/:id` - Eliminar receta
- `GET /api/v1/recetas/paciente/:paciente_id` - Recetas por paciente

#### Consultorios
- `POST /api/v1/consultorios` - Crear consultorio (admin)
- `GET /api/v1/consultorios` - Obtener consultorios
- `GET /api/v1/consultorios/disponibles` - Consultorios disponibles
- `GET /api/v1/consultorios/:id` - Obtener consultorio por ID
- `PUT /api/v1/consultorios/:id` - Actualizar consultorio (admin)
- `DELETE /api/v1/consultorios/:id` - Eliminar consultorio (admin)
- `GET /api/v1/consultorios/:id/horarios` - Horarios por consultorio

#### Horarios
- `POST /api/v1/horarios` - Crear horario (admin)
- `GET /api/v1/horarios` - Obtener horarios
- `GET /api/v1/horarios/disponibles` - Horarios disponibles
- `GET /api/v1/horarios/:id` - Obtener horario por ID
- `PUT /api/v1/horarios/:id` - Actualizar horario (admin)
- `DELETE /api/v1/horarios/:id` - Eliminar horario (admin)
- `PUT /api/v1/horarios/:id/disponibilidad` - Cambiar disponibilidad
- `GET /api/v1/horarios/medico/:medico_id` - Horarios por médico

#### Reportes
- `GET /api/v1/reportes/consultas` - Reporte de consultas
- `GET /api/v1/reportes/estadisticas` - Estadísticas generales (admin)
- `GET /api/v1/reportes/pacientes` - Reporte de pacientes
- `GET /api/v1/reportes/ingresos` - Reporte de ingresos (admin)

#### Administración
- `GET /api/v1/admin/usuarios/estadisticas` - Estadísticas de usuarios
- `GET /api/v1/admin/configuracion` - Configuración del sistema
- `GET /api/v1/admin/logs` - Logs del sistema

## 🔐 Autenticación y Autorización

### Registro de Usuario
```json
POST /api/v1/auth/register
{
  "nombre": "Dr. Juan Pérez",
  "email": "juan.perez@hospital.com",
  "password": "password123",
  "tipo": "medico"
}
```

### Inicio de Sesión
```json
POST /api/v1/auth/login
{
  "email": "juan.perez@hospital.com",
  "password": "password123"
}
```

### Uso del Token
Incluir en el header de las peticiones:
```
Authorization: Bearer <jwt_token>
```

## 📊 Ejemplos de Uso

### Crear una Consulta
```json
POST /api/v1/consultas
Authorization: Bearer <token>
{
  "tipo": "Consulta General",
  "costo": 500.00,
  "id_paciente": 1,
  "id_horario": 1
}
```

### Crear una Receta
```json
POST /api/v1/recetas
Authorization: Bearer <token>
{
  "medicamento": "Paracetamol 500mg",
  "dosis": "1 tableta cada 8 horas",
  "id_paciente": 1,
  "id_consultorio": 1
}
```

### Obtener Reportes
```json
GET /api/v1/reportes/consultas
Authorization: Bearer <token>
```

## 🗂️ Estructura del Proyecto

```
hospital-backend/
├── database/
│   └── connection.go          # Configuración de base de datos
├── handlers/
│   ├── usuarios.go           # Handlers de usuarios
│   ├── expedientes.go        # Handlers de expedientes
│   ├── consultas.go          # Handlers de consultas
│   ├── recetas.go            # Handlers de recetas
│   ├── consultorios.go       # Handlers de consultorios
│   ├── horarios.go           # Handlers de horarios
│   └── reportes.go           # Handlers de reportes
├── middleware/
│   └── auth.go               # Middleware de autenticación
├── models/
│   └── usuario.go            # Modelos de datos
├── routes/
│   └── routes.go             # Configuración de rutas
├── .env                      # Variables de entorno
├── go.mod                    # Dependencias
├── go.sum                    # Checksums de dependencias
├── main.go                   # Punto de entrada
└── README.md                 # Documentación
```

## 🛡️ Seguridad y Privacidad

### Medidas Implementadas
1. **Autenticación JWT**: Tokens seguros para autenticación
2. **Control de Acceso**: Roles específicos para cada tipo de usuario
3. **Encriptación**: Contraseñas hasheadas con bcrypt
4. **Validación**: Validación estricta de datos de entrada
5. **CORS**: Configuración de CORS para seguridad web
6. **Logging**: Registro de actividades para auditoría

### Privacidad de Datos
- Los pacientes solo pueden acceder a su propia información
- Los médicos solo ven sus propias consultas y pacientes asignados
- Las enfermeras tienen acceso limitado para asistencia
- Los administradores tienen acceso completo para gestión

## 🚨 Manejo de Errores

El sistema incluye manejo robusto de errores:
- Validación de datos de entrada
- Respuestas de error estructuradas
- Logging de errores para debugging
- Recuperación automática de errores

## 📈 Monitoreo y Logs

- Endpoint de salud: `GET /health`
- Logs automáticos de todas las peticiones
- Manejo de errores con stack traces
- Métricas de rendimiento

## 🔧 Desarrollo

### Ejecutar en modo desarrollo
```bash
go run main.go
```

### Ejecutar tests
```bash
go test ./...
```

### Compilar para producción
```bash
go build -o hospital-backend main.go
```

## 📝 Licencia

Este proyecto está desarrollado para fines educativos y de demostración.

## 👥 Contribución

1. Fork el proyecto
2. Crear una rama para tu feature (`git checkout -b feature/AmazingFeature`)
3. Commit tus cambios (`git commit -m 'Add some AmazingFeature'`)
4. Push a la rama (`git push origin feature/AmazingFeature`)
5. Abrir un Pull Request

## 📞 Soporte

Para soporte técnico o preguntas sobre el sistema, contactar al equipo de desarrollo.

---

**Sistema de Gestión Hospitalaria v1.0.0**  
*Desarrollado con Go, Fiber, PostgreSQL y mucho ❤️*