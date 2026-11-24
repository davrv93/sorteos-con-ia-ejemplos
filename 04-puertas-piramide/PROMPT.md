# Prompt · Pirámide de puertas (mazo fijo)

Este es el pedido en seis partes que crea este ejemplo. Cópialo, cambia los datos por los tuyos y pégalo en tu agente.

```text
Rol: eres desarrollador web full stack y diseñador de juegos de aventura.

Contexto: tengo un grupo de WhatsApp de unos 30 alumnos. Quiero premiarlos con un sorteo de un solo
intento que se sienta como una aventura.

Tarea: crea una pirámide de 30 puertas en filas de 2, 4, 6, 8 y 10. Detrás de cada puerta hay un
descuento o una trampa. La persona elige una puerta y la abre.

Límites:
- Reparto exacto en config.json: 8 trampas, 7 de 60 %, 5 de 70 %, 5 de 80 %, 3 de 90 % y 2 de 100 %.
- El mazo se baraja una sola vez en el servidor y se guarda; la página nunca lo ve.
- Para entrar hace falta la palabra del grupo (en config.json), comparada sin distinguir mayúsculas.
- Una puerta por número de WhatsApp. Las puertas tomadas muestran solo el primer nombre.
- Si es trampa, la pirámide se derrumba y suena un trombón triste; si gana, una marcha de aventura
  propia (no la de ninguna película).
- El premio lo decide el servidor con crypto/rand. La animación solo lo revela.
- El estado se guarda en un archivo JSON dentro de un volumen de Docker (/datos).
- Panel de administración protegido con una clave que viene de una variable de entorno.
- Sin CDN ni librerías externas en la página; el sonido, con Web Audio.
- No inventes premios, fechas ni precios: usa los que te doy.

Formato: un servidor en Go de un solo binario que sirve la página con embed, un Dockerfile de dos
etapas (compila y corre las pruebas dentro de Docker), un compose.yaml que publica solo en 127.0.0.1
y un .env.example con una clave de ejemplo.

Comprobación:
- docker compose up -d --build y la pirámide abre en http://127.0.0.1:8104.
- Una prueba que abre las 30 puertas: salen exactamente 8 trampas y 22 premios con ese reparto.
- Con una palabra equivocada, el servidor responde 403.
```

Si algo sale mal, usa el [segundo pedido](../prompts/01-segundo-pedido.md): el lugar, el cambio y «no toques nada más».
