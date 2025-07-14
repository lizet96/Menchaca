-- Script SQL para agregar el campo 'hora' a la tabla Consulta
-- Ejecuta este script directamente en tu cliente de PostgreSQL

-- 1. Agregar la columna 'hora' a la tabla Consulta
ALTER TABLE Consulta ADD COLUMN IF NOT EXISTS hora TIMESTAMP;

-- 2. (Opcional) Actualizar registros existentes con una hora por defecto
-- Descomenta la siguiente línea si quieres establecer una hora por defecto para registros existentes
-- UPDATE Consulta SET hora = CURRENT_TIMESTAMP WHERE hora IS NULL;

-- 3. Verificar que la columna se agregó correctamente
SELECT column_name, data_type, is_nullable 
FROM information_schema.columns 
WHERE table_name = 'consulta' AND column_name = 'hora';

-- 4. Ver la estructura actualizada de la tabla
\d Consulta;