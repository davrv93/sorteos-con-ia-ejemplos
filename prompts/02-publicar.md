# Publicar el sorteo en tu servidor

Cuando el sorteo funciona en tu computadora, este pedido lo deja en un servidor con HTTPS.

```text
Rol: eres administrador de sistemas Linux.

Contexto: tengo un servidor Ubuntu con Docker y nginx, y el dominio [midominio.com] con HTTPS.
Quiero publicar el sorteo de la carpeta [04-puertas-piramide] en https://[midominio.com]/piramide/.

Tarea: publica el sorteo detrás de nginx en esa ruta.

Límites:
- El contenedor escucha solo en 127.0.0.1; nginx es el único que da la cara a internet.
- Contenedor de solo lectura, sin capacidades (cap_drop ALL), con límite de memoria y un volumen
  para los datos.
- Genera la clave del panel en el servidor, guárdala en un .env con permisos 600 y no la imprimas.
- Antes de tocar la configuración de nginx, haz una copia; valida con nginx -t antes de recargar.
- No reinicies ni toques otros contenedores.

Formato: los comandos en orden y el bloque location de nginx.

Comprobación: curl -I https://[midominio.com]/piramide/ responde 200 y /piramide/admin sin clave
responde 401 o 404.
```
