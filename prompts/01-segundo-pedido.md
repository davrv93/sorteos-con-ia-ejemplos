# El segundo pedido: corregir sin romper

Cuando algo sale mal, no pidas «arréglalo» a secas. Di **el lugar**, **el cambio** y **qué no tocar**.

```text
En web/index.html, el botón «Girar» se ve detrás de la ruleta en el celular (pantalla de 390 px).
Muévelo debajo de la ruleta y súbele el contraste. No toques nada más: ni el servidor ni los premios.
Comprobación: una captura a 390 × 844 donde el botón se vea completo.
```

Si hay un error, pega el mensaje exacto:

```text
Al correr docker compose up -d --build sale este error:
  failed to solve: process "/bin/sh -c go test ./..." did not complete successfully
  main_test.go:88: tope de 50 % superado: 11 de 10
Corrige la causa en main.go, no la prueba. No cambies los topes de config.json.
```
