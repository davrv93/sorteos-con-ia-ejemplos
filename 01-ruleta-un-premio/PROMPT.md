# Prompt · Ruleta de un premio

Este es el pedido en seis partes que crea este ejemplo. Cópialo, cambia los datos por los tuyos y pégalo en tu agente.

```text
Rol: eres desarrollador web full stack y diseñador de juegos para clases en vivo.

Contexto: doy una clase por Zoom a unas 15 personas. Al final quiero sortear un premio. Abren el
enlace desde el celular; yo administro desde mi laptop.

Tarea: crea una ruleta de 12 casillas con un solo premio. El giro ganador (uno entre 12) se elige al
azar al crear el sorteo y quien haga ese giro gana. Si nadie acierta, al cerrar se sortea el premio
entre quienes giraron.

Límites:
- Cada persona escribe su nombre y gira una sola vez (por nombre y por navegador).
- El premio lo decide el servidor con crypto/rand. La animación solo lo revela.
- El estado se guarda en un archivo JSON dentro de un volumen de Docker (/datos).
- Panel de administración protegido con una clave que viene de una variable de entorno.
- Sin CDN ni librerías externas en la página; el sonido, con Web Audio.
- No inventes premios, fechas ni precios: usa los que te doy.
- Las casillas sin premio muestran plátanos; la del premio, una bolsa de dinero.

Formato: un servidor en Go de un solo binario que sirve la página con embed, un Dockerfile de dos
etapas (compila y corre las pruebas dentro de Docker), un compose.yaml que publica solo en 127.0.0.1
y un .env.example con una clave de ejemplo.

Comprobación:
- docker compose up -d --build y la ruleta abre en http://127.0.0.1:8101.
- Giro dos veces con el mismo nombre: la segunda me dice que ya giré.
- En /admin?k=<clave> veo la lista de giros y el botón «Cerrar el sorteo», que siempre deja un ganador.
```

Si algo sale mal, usa el [segundo pedido](../prompts/01-segundo-pedido.md): el lugar, el cambio y «no toques nada más».
