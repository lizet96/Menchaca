-- Crear tabla de logs para auditoría del sistema
CREATE TABLE IF NOT EXISTS public.logs (
    id_log SERIAL NOT NULL,
    method CHARACTER VARYING(10) NOT NULL,
    path CHARACTER VARYING(500) NOT NULL,
    protocol CHARACTER VARYING(10) NULL DEFAULT 'HTTP/1.1'::CHARACTER VARYING,
    status_code INTEGER NOT NULL,
    response_time INTEGER NULL,
    user_agent TEXT NULL,
    ip CHARACTER VARYING(45) NOT NULL,
    hostname CHARACTER VARYING(255) NULL,
    body TEXT NULL,
    params TEXT NULL,
    query TEXT NULL,
    email CHARACTER VARYING(255) NULL,
    username CHARACTER VARYING(100) NULL,
    role CHARACTER VARYING(50) NULL,
    log_level CHARACTER VARYING(20) NULL DEFAULT 'info'::CHARACTER VARYING,
    environment CHARACTER VARYING(20) NULL DEFAULT 'development'::CHARACTER VARYING,
    node_version CHARACTER VARYING(20) NULL,
    pid INTEGER NULL,
    timestamp TIMESTAMP WITH TIME ZONE NULL DEFAULT CURRENT_TIMESTAMP,
    url CHARACTER VARYING(1000) NULL,
    created_at TIMESTAMP WITH TIME ZONE NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT logs_pkey PRIMARY KEY (id_log)
) TABLESPACE pg_default;

-- Crear índices para optimizar consultas
CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON public.logs USING btree ("timestamp") TABLESPACE pg_default;
CREATE INDEX IF NOT EXISTS idx_logs_status_code ON public.logs USING btree (status_code) TABLESPACE pg_default;
CREATE INDEX IF NOT EXISTS idx_logs_method ON public.logs USING btree (method) TABLESPACE pg_default;
CREATE INDEX IF NOT EXISTS idx_logs_email ON public.logs USING btree (email) TABLESPACE pg_default;
CREATE INDEX IF NOT EXISTS idx_logs_log_level ON public.logs USING btree (log_level) TABLESPACE pg_default;
CREATE INDEX IF NOT EXISTS idx_logs_ip ON public.logs USING btree (ip) TABLESPACE pg_default;
CREATE INDEX IF NOT EXISTS idx_logs_path ON public.logs USING btree (path) TABLESPACE pg_default;
CREATE INDEX IF NOT EXISTS idx_logs_environment ON public.logs USING btree (environment) TABLESPACE pg_default;

-- Comentarios para documentar la tabla
COMMENT ON TABLE public.logs IS 'Tabla de auditoría para registrar todas las operaciones del sistema';
COMMENT ON COLUMN public.logs.id_log IS 'Identificador único del log';
COMMENT ON COLUMN public.logs.method IS 'Método HTTP de la petición';
COMMENT ON COLUMN public.logs.path IS 'Ruta de la petición';
COMMENT ON COLUMN public.logs.protocol IS 'Protocolo utilizado';
COMMENT ON COLUMN public.logs.status_code IS 'Código de estado HTTP de la respuesta';
COMMENT ON COLUMN public.logs.response_time IS 'Tiempo de respuesta en milisegundos';
COMMENT ON COLUMN public.logs.user_agent IS 'User-Agent del cliente';
COMMENT ON COLUMN public.logs.ip IS 'Dirección IP del cliente';
COMMENT ON COLUMN public.logs.hostname IS 'Hostname del servidor';
COMMENT ON COLUMN public.logs.body IS 'Cuerpo de la petición (filtrado para datos sensibles)';
COMMENT ON COLUMN public.logs.params IS 'Parámetros de la ruta';
COMMENT ON COLUMN public.logs.query IS 'Parámetros de consulta';
COMMENT ON COLUMN public.logs.email IS 'Email del usuario autenticado';
COMMENT ON COLUMN public.logs.username IS 'Nombre de usuario autenticado';
COMMENT ON COLUMN public.logs.role IS 'Rol del usuario autenticado';
COMMENT ON COLUMN public.logs.log_level IS 'Nivel del log (info, warning, error, debug, success)';
COMMENT ON COLUMN public.logs.environment IS 'Ambiente de ejecución (development, production, testing)';
COMMENT ON COLUMN public.logs.node_version IS 'Versión de Node.js (si aplica)';
COMMENT ON COLUMN public.logs.pid IS 'ID del proceso';
COMMENT ON COLUMN public.logs.timestamp IS 'Timestamp de cuando ocurrió el evento';
COMMENT ON COLUMN public.logs.url IS 'URL completa de la petición';
COMMENT ON COLUMN public.logs.created_at IS 'Timestamp de cuando se creó el registro';