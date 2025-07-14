-- Script para agregar permisos de horarios faltantes
-- Ejecutar este script en PostgreSQL

-- 1. Insertar permisos de horarios que faltan
INSERT INTO Permiso (nombre, descripcion, recurso, accion) VALUES 
('horarios_read', 'Ver horarios', 'horarios', 'read'),
('horarios_create', 'Crear horarios', 'horarios', 'create'),
('horarios_update', 'Actualizar horarios', 'horarios', 'update'),
('horarios_delete', 'Eliminar horarios', 'horarios', 'delete');

-- 2. Asignar todos los permisos de horarios al rol admin
INSERT INTO RolPermiso (id_rol, id_permiso)
SELECT r.id_rol, p.id_permiso
FROM Rol r, Permiso p
WHERE r.nombre = 'admin'
AND p.nombre IN ('horarios_read', 'horarios_create', 'horarios_update', 'horarios_delete');

-- 3. Asignar permisos de lectura de horarios a médicos
INSERT INTO RolPermiso (id_rol, id_permiso)
SELECT r.id_rol, p.id_permiso
FROM Rol r, Permiso p
WHERE r.nombre = 'medico'
AND p.nombre IN ('horarios_read');

-- 4. Verificar que los permisos se asignaron correctamente
SELECT r.nombre as rol, p.nombre as permiso, p.descripcion
FROM Rol r
JOIN RolPermiso rp ON r.id_rol = rp.id_rol
JOIN Permiso p ON rp.id_permiso = p.id_permiso
WHERE p.nombre LIKE 'horarios_%'
ORDER BY r.nombre, p.nombre;