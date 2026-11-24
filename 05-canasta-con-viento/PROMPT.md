# Prompt · Canasta con viento (juego de habilidad en vivo)

Este es el pedido en seis partes que crea este ejemplo. Cópialo, cambia los datos por los tuyos y pégalo en tu agente.

```text
Rol: eres desarrollador de juegos web en tiempo real.

Contexto: al final de mi clase en vivo quiero un minijuego de habilidad para sortear 2 cupos. Juegan
desde el celular mientras yo proyecto la tabla en vivo.

Tarea: crea un juego de basket: cada persona tiene 25 segundos para meter 6 canastas con un medidor
de tiro, mientras el viento cambia cada 3 segundos.

Límites:
- El servidor manda el viento, guarda el tiempo de cada tiro y decide si entró; la página solo dibuja
  y avisa en qué milisegundo se tocó el botón.
- WebSocket para la tabla en vivo; un límite de conexiones por IP.
- El juego empieza en pausa: yo lo abro, pauso y cierro desde el panel.
- Al cerrar, entre quienes tiraron y no ganaron, se sortea un premio de consuelo con crypto/rand.
- Dificultad en config.json (viento máximo, tolerancia, periodos del medidor).
- El premio lo decide el servidor con crypto/rand. La animación solo lo revela.
- El estado se guarda en un archivo JSON dentro de un volumen de Docker (/datos).
- Panel de administración protegido con una clave que viene de una variable de entorno.
- Sin CDN ni librerías externas en la página; el sonido, con Web Audio.
- No inventes premios, fechas ni precios: usa los que te doy.

Formato: un servidor en Go de un solo binario que sirve la página con embed, un Dockerfile de dos
etapas (compila y corre las pruebas dentro de Docker), un compose.yaml que publica solo en 127.0.0.1
y un .env.example con una clave de ejemplo.

Comprobación:
- docker compose up -d --build y el juego abre en http://127.0.0.1:8105.
- Pruebas que simulan tiros perfectos y tiros tarde: el servidor acepta los primeros y rechaza los
  segundos, sin importar lo que diga la página.
- Con el juego en pausa, el servidor rechaza los tiros.
```

Si algo sale mal, usa el [segundo pedido](../prompts/01-segundo-pedido.md): el lugar, el cambio y «no toques nada más».
