-- Agregar columna 'hora' a la tabla Consulta
ALTER TABLE Consulta ADD COLUMN hora TIMESTAMP;

-- Actualizar registros existentes con una hora por defecto (opcional)
-- UPDATE Consulta SET hora = CURRENT_TIMESTAMP WHERE hora IS NULL;