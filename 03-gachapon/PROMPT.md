# Prompt · Gachapon: máquina de cápsulas

Este es el pedido en seis partes que crea este ejemplo. Cópialo, cambia los datos por los tuyos y pégalo en tu agente.

```text
Rol: eres desarrollador web full stack y diseñador de juegos con animación.

Contexto: tengo una academia con varios talleres. Quiero un sorteo de cupones de descuento con forma
de máquina de cápsulas (gachapon) que se comparte por WhatsApp y se juega en el celular.

Tarea: crea la máquina: la persona escribe su nombre y su WhatsApp, gira la manivela, caen tres
cápsulas y abre una con su descuento. Luego ve el catálogo de talleres con su precio con el cupón.

Límites:
- Descuentos 20 %, 30 %, 40 % y 50 % con pesos 50, 30, 15 y 5, y topes de 30 cupones para el 40 %
  y 10 para el 50 %, en config.json.
- Una participación por número de WhatsApp. El cupón vale 7 días y nunca vence en viernes ni sábado.
- El catálogo sale de cursos.json; cada curso tiene una imagen en web/brochures/.
- Panel con buscador, «marcar canjeado» y descarga en CSV.
- El premio lo decide el servidor con crypto/rand. La animación solo lo revela.
- El estado se guarda en un archivo JSON dentro de un volumen de Docker (/datos).
- Panel de administración protegido con una clave que viene de una variable de entorno.
- Sin CDN ni librerías externas en la página; el sonido, con Web Audio.
- No inventes premios, fechas ni precios: usa los que te doy.

Formato: un servidor en Go de un solo binario que sirve la página con embed, un Dockerfile de dos
etapas (compila y corre las pruebas dentro de Docker), un compose.yaml que publica solo en 127.0.0.1
y un .env.example con una clave de ejemplo.

Comprobación:
- docker compose up -d --build y la máquina abre en http://127.0.0.1:8103.
- Una prueba que simula 10 000 participaciones: ningún descuento pasa su tope.
- Una prueba de que el CSV escapa los nombres que empiezan con =, +, - o @.
```

Si algo sale mal, usa el [segundo pedido](../prompts/01-segundo-pedido.md): el lugar, el cambio y «no toques nada más».
