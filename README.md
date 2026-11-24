# Sorteos y juegos interactivos con IA · ejemplos

Seis sorteos listos para levantar con Docker en Windows o Mac. Son los ejemplos del curso
**«Crea sorteos y juegos interactivos con IA»** (JMD Learning, modalidad web). Cada carpeta trae el
código, un `compose.yaml`, un README con capturas y el **prompt en seis partes** que lo crea.

![Gachapon: la máquina de cápsulas y el formulario para participar](docs/img/03-gachapon-inicio.jpg)

| # | Ejemplo | Qué enseña | Puerto |
|---|---|---|---|
| 01 | [Ruleta de un premio](01-ruleta-un-premio/) | Un solo premio escondido en un giro; siempre hay ganador | 8101 |
| 02 | [Ruleta con cupos](02-ruleta-con-cupos/) | Premios con stock, cierre por fecha, QR y WhatsApp | 8102 |
| 03 | [Gachapon](03-gachapon/) | Pesos y topes, cupón con vencimiento, catálogo y panel de canje | 8103 |
| 04 | [Pirámide de puertas](04-puertas-piramide/) | Mazo fijo barajado una vez y palabra del grupo | 8104 |
| 05 | [Canasta con viento](05-canasta-con-viento/) | Juego de habilidad en vivo: el servidor es el árbitro | 8105 |
| 06 | [Colibrí](06-colibri-aleteo/) | Un toque que revela un premio ya decidido, con video | 8106 |

## Lo que tienen en común

1. **El premio lo decide el servidor**, con `crypto/rand` de Go. La animación solo lo revela: tocar
   antes o después no cambia nada.
2. **Topes que no se pasan.** Los cupos están en `config.json` (o en el código, en los dos primeros)
   y las pruebas en Go simulan miles de participaciones para comprobarlo.
3. **Una participación por persona**, por número de WhatsApp o por navegador.
4. **Un panel con clave** que viene de una variable de entorno, nunca del código.
5. **Docker endurecido:** contenedor de solo lectura, sin capacidades, publicado solo en `127.0.0.1`.

## Antes de empezar

- **Docker Desktop** instalado y abierto. En Windows, con WSL 2 (el instalador lo ofrece).
- **Git** (opcional). Sin Git: botón **Code › Download ZIP** y descomprime la carpeta.

No necesitas instalar Go: el código se compila y se prueba dentro de Docker.

## Levantar un ejemplo

Los mismos comandos en Windows (PowerShell) y en Mac (Terminal):

```bash
git clone https://github.com/davrv93/sorteos-con-ia-ejemplos.git
cd sorteos-con-ia-ejemplos/03-gachapon
docker compose up -d --build
```

Abre **http://127.0.0.1:8103**. La primera vez tarda uno o dos minutos: descarga Go, compila y
corre las pruebas. Cada ejemplo usa su propio puerto, así que puedes tener los seis encendidos.

- **Panel:** cada README dice dónde está. La clave de ejemplo es `cambia-esta-clave-larga`.
- **Tu propia clave:** copia `.env.example` como `.env` y cámbiala. En Windows,
  `copy .env.example .env`; en Mac, `cp .env.example .env`. Luego `docker compose up -d`.
- **Apagar:** `docker compose down`. **Empezar de cero** (borra participantes): `docker compose down -v`.

## Los prompts

- [Plantilla del pedido en seis partes](prompts/00-plantilla-seis-partes.md): rol, contexto, tarea,
  límites, formato y comprobación.
- [El segundo pedido](prompts/01-segundo-pedido.md): cómo corregir sin romper lo que ya funciona.
- [Publicar en tu servidor](prompts/02-publicar.md): detrás de nginx, con HTTPS.
- Cada ejemplo tiene su `PROMPT.md` con el pedido exacto que lo crea.

## Antes de usarlo con gente real

- Cambia el número de WhatsApp de ejemplo (`51900000000`) por el tuyo, y la clave del panel.
- Los precios, talleres y fechas son de ejemplo.
- Si lo publicas en internet, ponlo detrás de HTTPS. En los ejemplos 01 y 02, la cookie de
  participación se marca como segura cuando la conexión llega por HTTPS.
- Pide solo los datos que necesitas para entregar el premio y di para qué los usas.

## Problemas frecuentes

| Síntoma | Qué hacer |
|---|---|
| `port is already allocated` | Otro programa usa ese puerto. En el `.env`, cambia `PUERTO=8110` y vuelve a levantar. |
| `all predefined address pools have been fully subnetted` | Docker se quedó sin redes. Borra las que no usas con `docker network prune`. |
| `GACHAPON_CLAVE debe tener al menos 12 caracteres` | Tu clave del `.env` es muy corta. |
| El juego de basket no deja tirar | Empieza en pausa: entra al panel y pulsa «Abrir». |

## Licencia

Código bajo licencia MIT. Las imágenes y los videos de los ejemplos se incluyen para practicar; los
datos (nombres, números y talleres) son ficticios.
