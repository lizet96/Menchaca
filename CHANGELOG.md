# Changelog

Todos los cambios notables de este proyecto serán documentados en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/es/1.0.0/),
y este proyecto adhiere a [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-01-15

### Agregado

#### Sistema de Autenticación y Autorización
- Implementación de JWT (JSON Web Tokens) para autenticación
- Sistema de roles basado en permisos (RBAC)
- Middleware de autenticación para rutas protegidas
- Roles disponibles: `admin`, `medico`, `enfermera`, `paciente`

#### API de Usuarios
- `POST /api/v1/auth/register` - Registro de usuarios con validaciones
- `POST /api/v1/auth/login` - Inicio de sesión con generación de JWT
- `GET /api/v1/usuarios` - Obtener lista de usuarios (solo admin)
- `GET /api/v1/usuarios/:id` - Obtener usuario por ID
- `PUT /api/v1/usuarios/:id` - Actualizar información de usuario
- `DELETE /api/v1/usuarios/:id` - Eliminar usuario (solo admin)

#### API de Expedientes Médicos
- `GET /api/v1/expedientes` - Listar expedientes
- `POST /api/v1/expedientes` - Crear nuevo expediente
- `GET /api/v1/expedientes/:id` - Obtener expediente específico
- `PUT /api/v1/expedientes/:id` - Actualizar expediente
- `DELETE /api/v1/expedientes/:id` - Eliminar expediente

#### API de Consultas Médicas
- `GET /api/v1/consultas` - Listar consultas
- `POST /api/v1/consultas` - Programar nueva consulta
- `GET /api/v1/consultas/:id` - Obtener consulta específica
- `PUT /api/v1/consultas/:id` - Actualizar consulta
- `DELETE /api/v1/consultas/:id` - Cancelar consulta
- `GET /api/v1/consultas/paciente/:id` - Consultas por paciente
- `GET /api/v1/consultas/medico/:id` - Consultas por médico

#### API de Recetas Médicas
- `GET /api/v1/recetas` - Listar recetas
- `POST /api/v1/recetas` - Crear nueva receta
- `GET /api/v1/recetas/:id` - Obtener receta específica
- `PUT /api/v1/recetas/:id` - Actualizar receta
- `DELETE /api/v1/recetas/:id` - Eliminar receta
- `GET /api/v1/recetas/paciente/:id` - Recetas por paciente

#### API de Consultorios
- `GET /api/v1/consultorios` - Listar consultorios
- `POST /api/v1/consultorios` - Crear nuevo consultorio
- `GET /api/v1/consultorios/:id` - Obtener consultorio específico
- `PUT /api/v1/consultorios/:id` - Actualizar consultorio
- `DELETE /api/v1/consultorios/:id` - Eliminar consultorio
- `GET /api/v1/consultorios/disponibles` - Consultorios disponibles

#### API de Horarios
- `GET /api/v1/horarios` - Listar horarios
- `POST /api/v1/horarios` - Crear nuevo horario
- `GET /api/v1/horarios/:id` - Obtener horario específico
- `PUT /api/v1/horarios/:id` - Actualizar horario
- `DELETE /api/v1/horarios/:id` - Eliminar horario
- `GET /api/v1/horarios/medico/:id` - Horarios por médico

#### API de Reportes
- `GET /api/v1/reportes/consultas` - Reporte de consultas
- `GET /api/v1/reportes/pacientes` - Reporte de pacientes
- `GET /api/v1/reportes/medicos` - Reporte de médicos
- `GET /api/v1/reportes/ingresos` - Reporte de ingresos

#### Funcionalidades de Administración
- `GET /api/v1/admin/usuarios` - Gestión de usuarios
- `GET /api/v1/admin/estadisticas` - Estadísticas del sistema
- `GET /api/v1/admin/configuracion` - Configuración del sistema

#### Base de Datos y Modelos
- Conexión a PostgreSQL mediante Supabase
- Modelo `Usuario` con campos: nombre, apellido, email, tipo, fecha_nacimiento
- Modelo `Consulta` para gestión de citas médicas
- Modelo `Expediente` para historiales médicos
- Modelo `Receta` para prescripciones médicas
- Modelo `Consultorio` para gestión de espacios
- Modelo `Horario` para programación de citas
- Modelo `Reporte` para análisis y estadísticas

#### Seguridad y Validaciones
- Validación de campos obligatorios en registro de usuarios
- Verificación de emails únicos
- Hashing de contraseñas con bcrypt
- Validación de formato de fecha de nacimiento
- Verificación de tipos de usuario válidos
- Manejo de errores HTTP estandarizado

#### Infraestructura
- Framework Fiber para API REST
- Middleware CORS configurado
- Manejo centralizado de errores 404
- Variables de entorno para configuración
- Endpoint `/health` para verificación de estado
- Endpoint `/routes` para listar rutas disponibles

### Tecnologías Utilizadas
- **Backend**: Go (Golang) con Fiber Framework
- **Base de Datos**: PostgreSQL (Supabase)
- **Autenticación**: JWT (JSON Web Tokens)
- **Hashing**: bcrypt para contraseñas
- **ORM**: Consultas SQL nativas
- **Variables de Entorno**: godotenv

### Configuración
- Puerto por defecto: 3000
- Base URL: `http://localhost:3000`
- Prefijo API: `/api/v1`
- Autenticación: Bearer Token en header Authorization

### Notas de Seguridad
- Todas las rutas protegidas requieren autenticación JWT
- Sistema de permisos basado en roles
- Validación de entrada en todos los endpoints
- Manejo seguro de contraseñas

---

## Formato de Versiones

- **MAJOR**: Cambios incompatibles en la API
- **MINOR**: Funcionalidad agregada de manera compatible
- **PATCH**: Correcciones de errores compatibles

## Tipos de Cambios

- `Agregado` para nuevas funcionalidades
- `Cambiado` para cambios en funcionalidades existentes
- `Obsoleto` para funcionalidades que serán removidas
- `Removido` para funcionalidades removidas
- `Corregido` para corrección de errores
- `Seguridad` para vulnerabilidades