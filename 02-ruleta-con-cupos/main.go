// Ruleta con cupos: cada persona escribe su nombre y su WhatsApp y gira una sola
// vez. El premio lo decide el servidor con crypto/rand. Precio normal S/ 40 y nadie paga menos de S/ 29, salvo la
// única beca del 100 %: 1 beca (gratis), 3 cupones de S/ 29, 6 de S/ 32 y, para todos los demás, S/ 35. Nadie se va
// sin premio. Los premios con stock se reparten como una urna sin reposición del tamaño de los participantes
// esperados y se descuentan bajo un mutex antes de guardar en /datos/estado.json.
package main

import (
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

//go:embed web
var web embed.FS

var lima = time.FixedZone("Lima", -5*3600)

type participacion struct {
	N      int       `json:"n"`
	Nombre string    `json:"nombre"`
	Llave  string    `json:"llave"`
	Tel    string    `json:"tel"`
	Disp   string    `json:"disp"`
	IP     string    `json:"ip"`
	Hora   time.Time `json:"hora"`
	Premio string    `json:"premio"`
	Codigo string    `json:"codigo"`
}

type estado struct {
	Inicial100 int             `json:"inicial100"`
	Inicial29  int             `json:"inicial29"`
	Inicial32  int             `json:"inicial32"`
	Stock100   int             `json:"stock100"`
	Stock29    int             `json:"stock29"`
	Stock32    int             `json:"stock32"`
	Esperados  int             `json:"esperados"`
	Cerrado    bool            `json:"cerrado"`
	Creado     time.Time       `json:"creado"`
	Parts      []participacion `json:"participaciones"`
}

var (
	mu     sync.Mutex
	est    estado
	ruta   string
	clave  = os.Getenv("SORTEO_CLAVE")
	cierre time.Time
	porIP  = 10
)

const cookieNombre = "ruleta_cupos"

func entero(nombre string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(nombre)); err == nil && v >= 0 {
		return v
	}
	return def
}

func azar(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(err)
	}
	return int(v.Int64())
}

func nuevo() estado {
	s100, s29, s32 := entero("SORTEO_STOCK100", 1), entero("SORTEO_STOCK29", 3), entero("SORTEO_STOCK32", 6)
	return estado{
		Inicial100: s100, Inicial29: s29, Inicial32: s32, Stock100: s100, Stock29: s29, Stock32: s32,
		Esperados: entero("SORTEO_ESPERADOS", 20), Creado: time.Now().UTC(), Parts: []participacion{},
	}
}

func guardar() error {
	b, err := json.Marshal(est)
	if err != nil {
		return err
	}
	tmp := ruta + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, ruta)
}

func cerrado() bool { return est.Cerrado || !time.Now().Before(cierre) }

// elegir saca un premio de una urna de max(esperados − participantes, premios con stock) bolillas: las primeras
// Stock100 son la beca, las siguientes Stock29 son cupones de S/ 29, luego Stock32 de S/ 32, y el resto paga S/ 35.
// Si llegan más personas que las esperadas, los premios con stock que queden salen en los siguientes giros.
// Se llama con mu tomado; el stock se descuenta aquí, antes de guardar.
func elegir() string {
	limitados := est.Stock100 + est.Stock29 + est.Stock32
	if limitados == 0 {
		return premio35
	}
	urna := est.Esperados - len(est.Parts)
	if urna < limitados {
		urna = limitados
	}
	r := azar(urna)
	switch {
	case r < est.Stock100:
		est.Stock100--
		return premioGratis
	case r < est.Stock100+est.Stock29:
		est.Stock29--
		return premio29
	case r < limitados:
		est.Stock32--
		return premio32
	}
	return premio35
}

func codigo() string {
	const abc = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	for {
		b := make([]byte, 4)
		for i := range b {
			b[i] = abc[azar(len(abc))]
		}
		c := "CUP-" + string(b)
		libre := true
		for _, p := range est.Parts {
			if p.Codigo == c {
				libre = false
				break
			}
		}
		if libre {
			return c
		}
	}
}

// llave: el nombre sin tildes, en minúsculas y sin espacios de más, para que «José  Pérez» y «jose perez» sean uno.
func llave(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.Join(strings.Fields(s), " ")) {
		switch r {
		case 'á', 'à', 'ä', 'â':
			r = 'a'
		case 'é', 'è', 'ë', 'ê':
			r = 'e'
		case 'í', 'ì', 'ï', 'î':
			r = 'i'
		case 'ó', 'ò', 'ö', 'ô':
			r = 'o'
		case 'ú', 'ù', 'ü', 'û':
			r = 'u'
		}
		if unicode.IsLetter(r) || r == ' ' {
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// telefono deja solo los dígitos, quita el 51 del país y exige un celular peruano de 9 dígitos que empiece en 9.
func telefono(s string) (string, bool) {
	var d strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			d.WriteRune(r)
		}
	}
	t := d.String()
	if len(t) == 11 && strings.HasPrefix(t, "51") {
		t = t[2:]
	}
	return t, len(t) == 9 && t[0] == '9'
}

func capital(w string) string {
	r, n := utf8.DecodeRuneInString(w)
	return string(unicode.ToUpper(r)) + strings.ToLower(w[n:])
}

// corto: nombre de pila + inicial del apellido («Ana R.»), lo único que se publica.
func corto(nombre string) string {
	f := strings.Fields(nombre)
	if len(f) == 0 {
		return ""
	}
	s := capital(f[0])
	if len(f) > 1 {
		r, _ := utf8.DecodeRuneInString(f[1])
		s += " " + string(unicode.ToUpper(r)) + "."
	}
	return s
}

// Premios (precio normal S/ 40). Se guarda el identificador; lo que se paga sale de la tabla, nunca de una
// cuenta con porcentajes, para que nadie pague menos de S/ 29 por redondeo (salvo la beca).
const (
	precioBase   = 40
	premioGratis = "gratis" // 1 beca del 100 %: paga S/ 0
	premio29     = "s29"    // cupón: paga S/ 29
	premio32     = "s32"    // cupón: paga S/ 32
	premio35     = "s35"    // todos los demás, sin límite: paga S/ 35
)

var tablaPaga = map[string]int{premioGratis: 0, premio29: 29, premio32: 32, premio35: 35}

// paga: lo que paga quien sacó ese premio. Un identificador desconocido paga el precio normal, nunca cero.
func paga(premio string) int {
	if v, ok := tablaPaga[premio]; ok {
		return v
	}
	return precioBase
}

// limitado: la beca o un cupón de S/ 29 o S/ 32 (los de stock limitado).
func limitado(premio string) bool {
	return premio == premioGratis || premio == premio29 || premio == premio32
}

func resultado(p participacion) map[string]any {
	return map[string]any{"n": p.N, "nombre": p.Nombre, "corto": corto(p.Nombre), "premio": p.Premio,
		"paga": paga(p.Premio), "base": precioBase, "codigo": p.Codigo, "tel": "*** *** " + p.Tel[6:]}
}

func escribir(w http.ResponseWriter, cod int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(cod)
	json.NewEncoder(w).Encode(v)
}

func fallo(w http.ResponseWriter, cod int, msg string) {
	escribir(w, cod, map[string]string{"error": msg})
}

func esAdmin(r *http.Request) bool {
	k := r.Header.Get("X-Clave")
	if k == "" {
		k = r.URL.Query().Get("k")
	}
	return clave != "" && subtle.ConstantTimeCompare([]byte(k), []byte(clave)) == 1
}

func ipDe(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return h
}

func publico(disp string) map[string]any {
	type fila struct {
		Nombre string `json:"nombre"`
		Premio string `json:"premio"`
		Paga   int    `json:"paga"`
	}
	grandes, todos := []fila{}, []fila{}
	for _, p := range est.Parts {
		f := fila{corto(p.Nombre), p.Premio, paga(p.Premio)}
		todos = append(todos, f)
		if limitado(p.Premio) {
			grandes = append(grandes, f)
		}
	}
	out := map[string]any{
		"cerrado": cerrado(), "cierre": cierre.Format(time.RFC3339), "ahora": time.Now().UTC().Format(time.RFC3339Nano),
		"participantes": len(est.Parts), "quedan100": est.Stock100, "quedan29": est.Stock29, "quedan32": est.Stock32,
		"inicial100": est.Inicial100, "inicial29": est.Inicial29, "inicial32": est.Inicial32, "grandes": grandes,
		"precioBase": precioBase,
	}
	if cerrado() {
		sort.SliceStable(todos, func(i, j int) bool { return todos[i].Paga < todos[j].Paga })
		out["entregados"] = todos
	}
	if disp != "" {
		for _, p := range est.Parts {
			if p.Disp == disp {
				out["yo"] = resultado(p)
			}
		}
	}
	return out
}

func main() {
	dir := os.Getenv("SORTEO_DATOS")
	if dir == "" {
		dir = "."
	}
	ruta = filepath.Join(dir, "estado.json")
	porIP = entero("SORTEO_POR_IP", 10)
	c := os.Getenv("SORTEO_CIERRE")
	if c == "" {
		c = "2027-06-30T23:59:00-05:00" // cámbialo con SORTEO_CIERRE en el .env
	}
	var err error
	if cierre, err = time.Parse(time.RFC3339, c); err != nil {
		log.Fatalf("SORTEO_CIERRE inválido: %v", err)
	}
	if b, err := os.ReadFile(ruta); err == nil {
		if err := json.Unmarshal(b, &est); err != nil {
			log.Fatalf("estado.json ilegible, no lo piso: %v", err)
		}
	} else {
		est = nuevo()
		if err := guardar(); err != nil {
			log.Fatalf("no puedo escribir %s: %v", ruta, err)
		}
	}
	if est.Parts == nil {
		est.Parts = []participacion{}
	}

	mux := http.NewServeMux()
	pagina := func(nombre string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			b, _ := web.ReadFile("web/" + nombre)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; img-src 'self' data:; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Write(b)
		}
	}
	mux.HandleFunc("GET /{$}", pagina("index.html"))
	mux.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) {
		if !esAdmin(r) {
			http.NotFound(w, r)
			return
		}
		pagina("admin.html")(w, r)
	})
	mux.HandleFunc("GET /api/salud", func(w http.ResponseWriter, r *http.Request) { escribir(w, 200, map[string]bool{"ok": true}) })
	mux.HandleFunc("GET /api/estado", func(w http.ResponseWriter, r *http.Request) {
		disp := ""
		if c, err := r.Cookie(cookieNombre); err == nil {
			disp = c.Value
		}
		mu.Lock()
		defer mu.Unlock()
		escribir(w, 200, publico(disp))
	})
	mux.HandleFunc("POST /api/girar", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Nombre   string `json:"nombre"`
			WhatsApp string `json:"whatsapp"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&in) != nil {
			fallo(w, 400, "No entendí el pedido.")
			return
		}
		nombre := strings.Join(strings.Fields(in.Nombre), " ")
		ll := llave(nombre)
		if strings.ContainsAny(nombre, "0123456789<>") || len(strings.Fields(ll)) < 2 || utf8.RuneCountInString(ll) < 5 || utf8.RuneCountInString(nombre) > 40 {
			fallo(w, 400, "Escribe tu nombre y tu apellido (solo letras, hasta 40).")
			return
		}
		tel, ok := telefono(in.WhatsApp)
		if !ok {
			fallo(w, 400, "Escribe tu WhatsApp de 9 dígitos, empezando por 9 (por ejemplo 987 654 321).")
			return
		}
		disp := ""
		if c, err := r.Cookie(cookieNombre); err == nil && len(c.Value) == 32 {
			disp = c.Value
		} else {
			b := make([]byte, 16)
			rand.Read(b)
			disp = hex.EncodeToString(b)
			http.SetCookie(w, &http.Cookie{Name: cookieNombre, Value: disp, Path: "/", MaxAge: 60 * 86400, HttpOnly: true, Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https", SameSite: http.SameSiteLaxMode})
		}
		ip := ipDe(r)
		mu.Lock()
		defer mu.Unlock()
		deIP := 0
		for _, p := range est.Parts {
			if p.Disp == disp {
				out := resultado(p)
				out["repetido"] = true
				escribir(w, 200, out)
				return
			}
			if p.Tel == tel {
				fallo(w, 409, "Ese número de WhatsApp ya participó. Cada número gira una sola vez.")
				return
			}
			if p.Llave == ll {
				fallo(w, 409, "Ese nombre ya participó. Cada persona gira una sola vez.")
				return
			}
			if p.IP == ip {
				deIP++
			}
		}
		if cerrado() {
			fallo(w, 409, "El sorteo ya cerró.")
			return
		}
		if porIP > 0 && deIP >= porIP {
			fallo(w, 429, "Desde esta conexión ya giraron muchas personas. Prueba desde tus datos móviles.")
			return
		}
		if len(est.Parts) >= 5000 {
			fallo(w, 409, "El sorteo está lleno.")
			return
		}
		antes100, antes29, antes32 := est.Stock100, est.Stock29, est.Stock32
		p := participacion{N: len(est.Parts) + 1, Nombre: nombre, Llave: ll, Tel: tel, Disp: disp, IP: ip, Hora: time.Now().UTC()}
		p.Premio = elegir()
		p.Codigo = codigo()
		est.Parts = append(est.Parts, p)
		if err := guardar(); err != nil {
			est.Parts = est.Parts[:len(est.Parts)-1]
			est.Stock100, est.Stock29, est.Stock32 = antes100, antes29, antes32
			log.Printf("guardar: %v", err)
			fallo(w, 503, "No pudimos guardar tu giro. Intenta otra vez en un momento.")
			return
		}
		escribir(w, 200, resultado(p))
	})

	admin := func(f func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !esAdmin(r) {
				http.NotFound(w, r)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			f(w, r)
		}
	}
	vistaAdmin := func(w http.ResponseWriter) {
		escribir(w, 200, map[string]any{"estado": est, "cerrado": cerrado(), "cierre": cierre.In(lima).Format("02/01/2006 15:04")})
	}
	mux.HandleFunc("GET /api/admin/estado", admin(func(w http.ResponseWriter, r *http.Request) { vistaAdmin(w) }))
	mux.HandleFunc("POST /api/admin/cerrar", admin(func(w http.ResponseWriter, r *http.Request) {
		est.Cerrado = true
		guardar()
		vistaAdmin(w)
	}))
	mux.HandleFunc("POST /api/admin/reabrir", admin(func(w http.ResponseWriter, r *http.Request) {
		est.Cerrado = false
		guardar()
		vistaAdmin(w)
	}))
	mux.HandleFunc("POST /api/admin/reiniciar", admin(func(w http.ResponseWriter, r *http.Request) {
		if b, err := json.Marshal(est); err == nil {
			os.WriteFile(filepath.Join(filepath.Dir(ruta), fmt.Sprintf("estado-%s.json", time.Now().In(lima).Format("20060102-150405"))), b, 0o600)
		}
		est = nuevo()
		guardar()
		vistaAdmin(w)
	}))

	puerto := os.Getenv("PORT")
	if puerto == "" {
		puerto = "18745"
	}
	log.Printf("ruleta con cupos en :%s (participantes %d, quedan gratis: %d, S/ 29: %d, S/ 32: %d, esperados %d, cierre %s)",
		puerto, len(est.Parts), est.Stock100, est.Stock29, est.Stock32, est.Esperados, cierre.In(lima).Format(time.RFC3339))
	log.Fatal((&http.Server{Addr: ":" + puerto, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}).ListenAndServe())
}
