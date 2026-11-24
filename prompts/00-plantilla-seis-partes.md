# Plantilla: el pedido en seis partes

Todos los prompts de este repositorio siguen la misma estructura. Cópiala, cambia lo que está entre
corchetes y pégala en tu agente (Claude Code, OpenCode, DeepSeek o el que uses).

| Parte | Responde a |
|---|---|
| 1 · Rol | ¿Quién debe ser el agente? |
| 2 · Contexto | ¿Para quién es y en qué situación se usa? |
| 3 · Tarea | ¿Qué debe hacer? Una sola tarea, con un verbo. |
| 4 · Límites | ¿Qué no debe hacer ni inventar? |
| 5 · Formato | ¿Cómo quieres recibir el resultado? |
| 6 · Comprobación | ¿Cómo sabremos que está bien? Algo que se ve o se ejecuta. |

```text
Rol: eres desarrollador web full stack y diseñador de juegos para eventos en vivo.

Contexto: hago un sorteo para [mi clase / mi tienda / mi evento] de [número] personas. Lo abren
desde el celular, con un código QR o un enlace de WhatsApp. Yo lo administro desde mi laptop.

Tarea: crea el sorteo «[nombre]»: [la mecánica en una o dos frases].

Límites:
- El premio lo decide el servidor con crypto/rand. La animación solo lo revela.
- Una participación por persona (por número de WhatsApp o por navegador).
- Topes de premios en un archivo config.json, no en el código.
- El estado se guarda en un archivo JSON dentro de un volumen de Docker.
- Panel de administración protegido con una clave que viene de una variable de entorno.
- Sin librerías externas en la página ni CDN; todo el sonido con Web Audio.
- No inventes premios, fechas ni precios: usa los que te doy.

Formato: un servidor en Go de un solo binario que sirve la página (embed), un Dockerfile de dos
etapas, un compose.yaml que publica en 127.0.0.1 y un .env.example con la clave de ejemplo.

Comprobación:
- docker compose up -d --build levanta el sorteo en http://127.0.0.1:[puerto].
- Juego dos veces con el mismo número: la segunda vez me muestra el mismo resultado.
- Pruebas en Go que simulan 10 000 participaciones y confirman que nadie pasa los topes.
```

Reglas que conviene repetir en cada pedido:

1. Una tarea por pedido. Si son tres cosas, son tres pedidos.
2. La comprobación es algo que se ve o se ejecuta: un archivo que existe, un comando que responde,
   una prueba que pasa.
3. Nunca le pegues al agente una clave real. Usa la clave de ejemplo del `.env.example`.
4. Usa `127.0.0.1` y no `localhost`: evita que el navegador pruebe IPv6 primero.
5. Si algo no se puede comprobar, que el agente lo diga en lugar de darlo por hecho.
