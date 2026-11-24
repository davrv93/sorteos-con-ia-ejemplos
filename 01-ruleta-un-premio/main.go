// Ruleta de un solo premio: cada persona escribe su nombre y gira la ruleta.
// El giro premiado se elige al azar al crear el sorteo (uno entre TOTAL) y nadie lo conoce: quien haga ese
// giro gana. Un giro por persona (por nombre y por navegador). Si al cerrar nadie acertó, el premio se
// sortea entre quienes giraron, así que siempre hay ganador.
package main

import (
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

//go:embed web
var web embed.FS

type giro struct {
	N      int       `json:"n"`
	Nombre string    `json:"nombre"`
	Llave  string    `json:"llave"`
	Disp   string    `json:"disp"`
	Hora   time.Time `json:"hora"`
	Gano   bool      `json:"gano"`
}

type estado struct {
	Total     int    `json:"total"`
	Premiado  int    `json:"premiado"`
	Giros     []giro `json:"giros"`
	Ganador   string `json:"ganador"`
	Codigo    string `json:"codigo"`
	Cerrado   bool   `json:"cerrado"`
	PorCierre bool   `json:"por_cierre"`
}

var (
	mu    sync.Mutex
	est   estado
	ruta  string
	clave = os.Getenv("SORTEO_CLAVE")
)

func azar(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(err)
	}
	return int(v.Int64())
}

func codigo() string {
	const abc = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = abc[azar(len(abc))]
	}
	return "OR-" + string(b)
}

func nuevo(total int) estado {
	if total < 2 {
		total = 12
	}
	return estado{Total: total, Premiado: 1 + azar(total), Giros: []giro{}}
}

func guardar() {
	b, _ := json.Marshal(est)
	tmp := ruta + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		os.Rename(tmp, ruta)
	}
}

// llave: el nombre sin tildes, en minúsculas y sin espacios de más, para que «José  Pérez» y «jose perez» sean uno.
func llave(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.Join(strings.Fields(s), " ")) {
		switch r {
		case 'á', 'à', 'ä':
			r = 'a'
		case 'é', 'è', 'ë':
			r = 'e'
		case 'í', 'ì', 'ï':
			r = 'i'
		case 'ó', 'ò', 'ö':
			r = 'o'
		case 'ú', 'ù', 'ü':
			r = 'u'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func escribir(w http.ResponseWriter, cod int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(cod)
	json.NewEncoder(w).Encode(v)
}

func resultado(g giro) map[string]any {
	out := map[string]any{"gano": g.Gano, "n": g.N, "nombre": g.Nombre, "giros": len(est.Giros)}
	if g.Gano {
		out["codigo"] = est.Codigo
	} else if est.Ganador != "" {
		out["ganador"] = est.Ganador
	}
	return out
}

func esAdmin(r *http.Request) bool {
	k := r.URL.Query().Get("k")
	return clave != "" && subtle.ConstantTimeCompare([]byte(k), []byte(clave)) == 1
}

func main() {
	dir := os.Getenv("SORTEO_DATOS")
	if dir == "" {
		dir = "."
	}
	ruta = filepath.Join(dir, "estado.json")
	total, _ := strconv.Atoi(os.Getenv("SORTEO_TOTAL"))
	if b, err := os.ReadFile(ruta); err != nil || json.Unmarshal(b, &est) != nil || est.Total < 2 {
		est = nuevo(total)
		guardar()
	}
	mux := http.NewServeMux()
	pagina := func(nombre string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			b, _ := web.ReadFile("web/" + nombre)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; img-src 'self' data:; base-uri 'none'; form-action 'none'; frame-ancestors 'self'")
			w.Header().Set("X-Content-Type-Options", "nosniff")
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
	mux.HandleFunc("GET /api/estado", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		out := map[string]any{"giros": len(est.Giros), "ganador": est.Ganador, "cerrado": est.Cerrado, "total": est.Total}
		if c, err := r.Cookie("sor"); err == nil {
			for _, g := range est.Giros {
				if g.Disp == c.Value {
					out["yo"] = resultado(g)
				}
			}
		}
		escribir(w, 200, out)
	})
	mux.HandleFunc("POST /api/girar", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Nombre string `json:"nombre"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&in) != nil {
			escribir(w, 400, map[string]string{"error": "No entendí el pedido."})
			return
		}
		nombre := strings.Join(strings.Fields(in.Nombre), " ")
		ll := llave(nombre)
		if utf8.RuneCountInString(ll) < 3 || utf8.RuneCountInString(nombre) > 40 {
			escribir(w, 400, map[string]string{"error": "Escribe tu nombre y apellido (entre 3 y 40 letras)."})
			return
		}
		disp := ""
		if c, err := r.Cookie("sor"); err == nil && len(c.Value) == 32 {
			disp = c.Value
		} else {
			b := make([]byte, 16)
			rand.Read(b)
			disp = hex.EncodeToString(b)
			http.SetCookie(w, &http.Cookie{Name: "sor", Value: disp, Path: "/", MaxAge: 30 * 86400, HttpOnly: true, Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https", SameSite: http.SameSiteLaxMode})
		}
		mu.Lock()
		defer mu.Unlock()
		for _, g := range est.Giros {
			if g.Disp == disp || g.Llave == ll {
				out := resultado(g)
				out["repetido"] = true
				escribir(w, 200, out)
				return
			}
		}
		if est.Cerrado {
			escribir(w, 409, map[string]string{"error": "El sorteo ya cerró."})
			return
		}
		if len(est.Giros) >= 200 {
			escribir(w, 409, map[string]string{"error": "El sorteo está lleno."})
			return
		}
		g := giro{N: len(est.Giros) + 1, Nombre: nombre, Llave: ll, Disp: disp, Hora: time.Now().UTC()}
		if est.Ganador == "" && g.N == est.Premiado {
			g.Gano = true
			est.Ganador, est.Codigo = nombre, codigo()
		}
		est.Giros = append(est.Giros, g)
		guardar()
		escribir(w, 200, resultado(g))
	})
	mux.HandleFunc("GET /api/admin/estado", func(w http.ResponseWriter, r *http.Request) {
		if !esAdmin(r) {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		escribir(w, 200, est)
	})
	// cerrar: nadie más gira; si el giro premiado no llegó a salir, el premio se sortea entre quienes giraron.
	mux.HandleFunc("POST /api/admin/cerrar", func(w http.ResponseWriter, r *http.Request) {
		if !esAdmin(r) {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if est.Ganador == "" && len(est.Giros) > 0 {
			i := azar(len(est.Giros))
			est.Giros[i].Gano = true
			est.Ganador, est.Codigo, est.PorCierre = est.Giros[i].Nombre, codigo(), true
		}
		est.Cerrado = true
		guardar()
		escribir(w, 200, est)
	})
	mux.HandleFunc("POST /api/admin/reiniciar", func(w http.ResponseWriter, r *http.Request) {
		if !esAdmin(r) {
			http.NotFound(w, r)
			return
		}
		t, _ := strconv.Atoi(r.URL.Query().Get("total"))
		mu.Lock()
		defer mu.Unlock()
		if t < 2 {
			t = est.Total
		}
		est = nuevo(t)
		guardar()
		escribir(w, 200, map[string]any{"ok": true, "total": est.Total})
	})
	puerto := os.Getenv("PORT")
	if puerto == "" {
		puerto = "18715"
	}
	log.Printf("sorteo-openrouter en :%s (total %d, giros %d)", puerto, est.Total, len(est.Giros))
	log.Fatal((&http.Server{Addr: ":" + puerto, Handler: mux, ReadHeaderTimeout: 5 * time.Second}).ListenAndServe())
}
