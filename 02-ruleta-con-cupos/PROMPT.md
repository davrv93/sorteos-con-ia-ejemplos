# Prompt · Ruleta con cupos limitados

Este es el pedido en seis partes que crea este ejemplo. Cópialo, cambia los datos por los tuyos y pégalo en tu agente.

```text
Rol: eres desarrollador web full stack y diseñador de juegos de promoción.

Contexto: vendo un taller de S/ 40 y quiero una ruleta de descuentos que se comparte por WhatsApp.
La abren desde el celular.

Tarea: crea una ruleta de 16 casillas con estos premios: 1 beca del 100 %, 3 cupones de S/ 29,
6 cupones de S/ 32 y, para todos los demás, S/ 35.

Límites:
- Una participación por número de WhatsApp (9 dígitos, empieza con 9). Si repite, se rechaza.
- Cuando un premio llega a su tope, sale de la ruleta. Lo que se paga sale de una tabla, no de un
  cálculo con porcentajes.
- El sorteo cierra en la fecha de la variable SORTEO_CIERRE y la página muestra una cuenta regresiva.
- Al ganar: código del cupón, botón de WhatsApp con el mensaje ya escrito y un QR generado en la
  página (sin servicios externos).
- El premio lo decide el servidor con crypto/rand. La animación solo lo revela.
- El estado se guarda en un archivo JSON dentro de un volumen de Docker (/datos).
- Panel de administración protegido con una clave que viene de una variable de entorno.
- Sin CDN ni librerías externas en la página; el sonido, con Web Audio.
- No inventes premios, fechas ni precios: usa los que te doy.

Formato: un servidor en Go de un solo binario que sirve la página con embed, un Dockerfile de dos
etapas (compila y corre las pruebas dentro de Docker), un compose.yaml que publica solo en 127.0.0.1
y un .env.example con una clave de ejemplo.

Comprobación:
- docker compose up -d --build y la ruleta abre en http://127.0.0.1:8102.
- Una prueba que simula 1000 giros: sale 1 beca, 3 de S/ 29, 6 de S/ 32 y el resto S/ 35.
- En /admin?k=<clave> veo cuántos cupones quedan de cada premio.
```

Si algo sale mal, usa el [segundo pedido](../prompts/01-segundo-pedido.md): el lugar, el cambio y «no toques nada más».
