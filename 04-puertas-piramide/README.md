# Pirámide de puertas (mazo fijo)

Una pirámide con 30 puertas: 22 sacan a la persona con un descuento de 60 % a 100 % y 8 son trampas. Lo que hay detrás de cada puerta se baraja una sola vez, al primer arranque, y queda guardado. Cada persona abre una puerta; las que ya se abrieron quedan selladas con el nombre de quien las tomó. Para entrar hace falta la palabra del grupo.

![Pirámide de puertas: 30 puertas numeradas y el cubito explorador](../docs/img/04-piramide-inicio.jpg)

## Cómo levantarlo (Windows y Mac)

Con Docker Desktop abierto, en esta carpeta:

```bash
docker compose up -d --build
```

- El sorteo: **http://127.0.0.1:8104**
- El panel: **http://127.0.0.1:8104/admin (escribe la clave en el formulario)**. La clave de ejemplo es `cambia-esta-clave-larga`.
- Para apagarlo: `docker compose down`. Para borrar también los participantes: `docker compose down -v`.

Para usar tu propia clave, copia `.env.example` como `.env` y cámbiala (mínimo 12 caracteres). En
Windows: `copy .env.example .env`; en Mac: `cp .env.example .env`.

## Qué enseña

- **Mazo fijo:** el reparto exacto está en `config.json` (8 trampas, 7 de 60 %, 5 de 70 %, 5 de 80 %, 3 de 90 % y 2 de 100 %). Se baraja una vez y queda en `/datos/puertas.json`. La página nunca lo ve.
- **Elegir rápido o lento no cambia nada:** lo que hay detrás de cada puerta ya estaba decidido.
- **Palabra del grupo:** solo entra quien la conoce (`palabra_grupo`, aquí `PIRAMIDE`).
- **Regalo:** quien gana elige «Es para mí» o «Es un regalo»; los dos abren WhatsApp con el texto escrito.

![Pirámide de puertas: escapó por la puerta 8 con 80 % de descuento](../docs/img/04-piramide-resultado.jpg)

## Hazlo tuyo

Si cambias las cantidades de `config.json` después de que alguien jugó, el servidor no baraja de nuevo: borra el volumen solo si nadie ha jugado (`docker compose down -v`).

El prompt que crea este ejemplo está en [PROMPT.md](PROMPT.md).
