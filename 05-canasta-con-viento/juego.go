package main

// Reglas del juego, sin red: inscripción, ronda de 25 s, viento, medidor, tiros, ganadores y premios. Todo el azar sale de j.azar
// (crypto/rand en producción) y el reloj de j.ahora, para poder probarlo. Los métodos toman j.mu.

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

type Grupo struct {
	ID   string `json:"id"`
	Dias string `json:"dias"`
	Hora string `json:"hora"`
}

type Premio struct {
	ID       string  `json:"id"`
	Nombre   string  `json:"nombre"`
	Corto    string  `json:"corto,omitempty"`
	Etiqueta string  `json:"etiqueta"`
	Cupos    int     `json:"cupos"`
	Precio   int     `json:"precio,omitempty"` // opcional; lo usan los cursos del descuento especial
	Grupos   []Grupo `json:"grupos,omitempty"`
	Dias     string  `json:"dias,omitempty"`
}

type Medidor struct {
	PeriodosMs   []int   `json:"periodos_ms"`
	MediaZona    float64 `json:"media_zona"`
	CentroMin    float64 `json:"centro_min"`
	CentroMax    float64 `json:"centro_max"`
	ProbPerfecto float64 `json:"prob_perfecto"`
	ProbBorde    float64 `json:"prob_borde"`
	ProbCerca    float64 `json:"prob_cerca"`
	ProbLejos    float64 `json:"prob_lejos"`
}

// Consuelo: para quien tiró al menos una vez y no ganó un cupo al 100 %. Se sortea al cerrar: especial_cupos
// personas (una) con el descuento grande en un curso en vivo y, todos los demás, el premio web (acceso a un
// curso del catálogo por un precio fijo).
type Consuelo struct {
	EspecialPagaPct int       `json:"especial_paga_pct"`
	EspecialCupos   int       `json:"especial_cupos"`
	EspecialTexto   string    `json:"especial_texto"`
	EspecialCursos  []string  `json:"especial_cursos"` // cursos en vivo (con precio) a los que se aplica
	Premio          PremioWeb `json:"premio"`
}

type PremioWeb struct {
	Texto       string   `json:"texto"`
	Precio      int      `json:"precio"`
	Vigencia    string   `json:"vigencia"`
	Incluye     []string `json:"incluye"`
	NoIncluye   string   `json:"no_incluye"`
	URLCatalogo string   `json:"url_catalogo"`
}

// Viento: cada cada_ms el servidor sortea uno nuevo (−max…max; negativo empuja a la izquierda). El tiro se
// desvía lateralmente lo que sopla el viento menos lo que el jugador compensa apuntando. Con un desvío de hasta
// tolerancia la puntería no resta; desde caida ya no entra.
type Viento struct {
	CadaMs     int     `json:"cada_ms"`
	Max        float64 `json:"max"`
	Tolerancia float64 `json:"tolerancia"`
	Caida      float64 `json:"caida"`
}

type Config struct {
	Titulo       string   `json:"titulo"`
	LimiteMs     int      `json:"limite_ms"` // ronda de cada jugador: 25 000 ms desde que pulsa «Empezar»
	Viento       Viento   `json:"viento"`
	Taller       string   `json:"taller"`
	Meta         int      `json:"meta"`
	EsperaMs     int      `json:"espera_ms"`
	PorIP        int      `json:"por_ip"`
	MaxJugadores int      `json:"max_jugadores"`
	AbiertoAlIni bool     `json:"abierto_al_iniciar"`
	RevelarMs    int      `json:"revelar_ms"`
	Premios      []Premio `json:"premios"`
	Medidor      Medidor  `json:"medidor"`
	Consuelo     Consuelo `json:"consuelo"`
	WhatsApp     string   `json:"whatsapp"`
}

func (c Config) premio(id string) *Premio {
	for i := range c.Premios {
		if c.Premios[i].ID == id {
			return &c.Premios[i]
		}
	}
	return nil
}

func (c Config) validar() error {
	if c.LimiteMs < 5000 || c.Viento.CadaMs < 500 || c.Viento.Max <= 0 || c.Viento.Tolerancia <= 0 || c.Viento.Caida <= c.Viento.Tolerancia {
		return errors.New("config.json: limite_ms y viento (cada_ms, max, tolerancia < caida) son obligatorios")
	}
	if c.Meta < 1 || c.EsperaMs < 300 || len(c.Premios) == 0 || len(c.Medidor.PeriodosMs) == 0 || c.Medidor.MediaZona <= 0 {
		return errors.New("config.json: meta, espera_ms, premios y medidor son obligatorios")
	}
	for _, p := range c.Premios {
		if p.ID == "" || p.Cupos < 0 || p.Precio < 0 {
			return errors.New("config.json: cada premio necesita id y cupos ≥ 0")
		}
	}
	if c.Consuelo.EspecialCupos < 0 || c.Consuelo.EspecialPagaPct <= 0 || c.Consuelo.EspecialPagaPct > 100 {
		return errors.New("config.json: consuelo.especial_paga_pct va de 1 a 100")
	}
	if c.Consuelo.Premio.Precio <= 0 || c.Consuelo.Premio.Texto == "" {
		return errors.New("config.json: consuelo.premio necesita texto y precio")
	}
	for _, id := range c.Consuelo.EspecialCursos {
		if p := c.premio(id); p == nil || p.Precio <= 0 {
			return errors.New("config.json: consuelo.especial_cursos debe nombrar premios con precio")
		}
	}
	return nil
}

type Jugador struct {
	ID       string    `json:"id"`
	Token    string    `json:"token"`
	Nombre   string    `json:"nombre"`
	Llave    string    `json:"llave"`
	Tel      string    `json:"tel"`
	IP       string    `json:"ip"`
	Semilla  uint32    `json:"semilla"`
	Num      int       `json:"num"`
	Lugar    int       `json:"lugar"`
	Aciertos int       `json:"aciertos"`
	Tiros    int       `json:"tiros"`
	Gano     int       `json:"gano"`
	Creado   time.Time `json:"creado"`
	Inicio   time.Time `json:"inicio"` // cuándo pulsó «Empezar»; cero si aún no juega. Una sola ronda por persona.
}

type Ganador struct {
	Orden   int       `json:"orden"`
	Jugador string    `json:"jugador"`
	Premio  string    `json:"premio"`
	Grupo   string    `json:"grupo,omitempty"`
	Tiros   int       `json:"tiros"`
	Hora    time.Time `json:"hora"`
}

type Estado struct {
	Abierto   bool           `json:"abierto"`
	Cerrado   bool           `json:"cerrado"`
	CerradoEn time.Time      `json:"cerrado_en"`
	Restantes map[string]int `json:"restantes"`
	Jugadores []*Jugador     `json:"jugadores"`
	Ganadores []Ganador      `json:"ganadores"`
	Creado    time.Time      `json:"creado"`
	// Especiales: los del descuento grande, en orden. AccesoWeb: los del premio web.
	Especiales []string        `json:"especiales"`
	AccesoWeb  map[string]bool `json:"acceso_web"`
	// Consuelos: formato anterior (jugador → % que paga). Solo se lee para convertirlo al cargar.
	Consuelos map[string]int `json:"consuelos,omitempty"`
}

// Fase: lo que el servidor manda antes de cada tiro. El medidor vale tri(e/P + X), con e los ms desde que se emitió.
type Fase struct {
	ID      uint32    `json:"f"`
	P       int       `json:"p"`
	C       float64   `json:"c"`
	H       float64   `json:"h"`
	X       float64   `json:"x"`
	V       float64   `json:"v"` // viento de este tiro
	Emitida time.Time `json:"-"`
}

// Tiro: resultado que se difunde a todos. R: limpia | aro (entra) | rebota | corta | larga (fallo).
type Tiro struct {
	Jugador  string
	R        string
	Ok       bool
	D        int // desvío lateral −100…100 para la animación
	M        int // medidor ×1000
	C        int // centro de la franja ×1000
	Aciertos int
	Tiros    int
}

type Victoria struct {
	Ganador
	Corto   string
	Semilla uint32
	Num     int
}

var (
	ErrNombre   = errors.New("Escribe tu nombre y tu apellido (solo letras, hasta 40).")
	ErrTel      = errors.New("Escribe tu WhatsApp de 9 dígitos, empezando por 9 (por ejemplo 987 654 321).")
	ErrTelUsado = errors.New("Ese WhatsApp ya está jugando con otro nombre. Escribe el mismo nombre con el que entraste.")
	ErrNomUsado = errors.New("Ese nombre ya está jugando con otro WhatsApp.")
	ErrIP       = errors.New("Desde esta conexión ya entraron muchas personas. Prueba con tus datos móviles.")
	ErrLleno    = errors.New("La cancha está llena.")
	ErrPausa    = errors.New("El juego está en pausa. Empieza cuando el profe lo abra.")
	ErrCerrado  = errors.New("El sorteo ya cerró.")
	ErrYaJugo   = errors.New("Ya jugaste tu ronda de 25 segundos.")
	ErrNoEmpezo = errors.New("Pulsa «Empezar» para que corran tus 25 segundos.")
	ErrTiempo   = errors.New("Se acabó tu tiempo.")
	ErrEspera   = errors.New("espera")
	ErrFase     = errors.New("fase")
)

type vientoVivo struct {
	v     float64
	hasta time.Time
}

type Juego struct {
	mu     sync.Mutex
	cfg    Config
	est    Estado
	porTok map[string]*Jugador
	porID  map[string]*Jugador
	fases  map[string]*Fase
	ultimo map[string]time.Time
	vient  map[string]vientoVivo
	nfase  uint32
	azar   func(n int) int
	ahora  func() time.Time
}

func azarCripto(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(err)
	}
	return int(v.Int64())
}

func hexAzar(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func SemillaNueva() uint32 {
	var b [4]byte
	rand.Read(b[:])
	return binary.BigEndian.Uint32(b[:])
}

func NuevoJuego(cfg Config) *Juego {
	j := &Juego{cfg: cfg, azar: azarCripto, ahora: time.Now}
	j.est = j.estadoNuevo()
	j.indexar()
	return j
}

func (j *Juego) estadoNuevo() Estado {
	r := map[string]int{}
	for _, p := range j.cfg.Premios {
		r[p.ID] = p.Cupos
	}
	return Estado{Abierto: j.cfg.AbiertoAlIni, Restantes: r, Jugadores: []*Jugador{}, Ganadores: []Ganador{}, Creado: j.ahora().UTC(),
		Especiales: []string{}, AccesoWeb: map[string]bool{}}
}

func (j *Juego) indexar() {
	j.porTok, j.porID = map[string]*Jugador{}, map[string]*Jugador{}
	j.fases, j.ultimo, j.vient = map[string]*Fase{}, map[string]time.Time{}, map[string]vientoVivo{}
	for _, p := range j.est.Jugadores {
		j.porTok[p.Token] = p
		j.porID[p.ID] = p
	}
	// los cupos que quedan salen de config.json: cupos − ganadores que ya tienen ese premio (nunca negativo).
	// Así se pueden agregar premios o cerrar uno (cupos = sus ganadores) sin tocar estado.json.
	j.est.Restantes = map[string]int{}
	usados := map[string]int{}
	for _, g := range j.est.Ganadores {
		usados[g.Premio]++
	}
	for _, p := range j.cfg.Premios {
		j.est.Restantes[p.ID] = max(0, p.Cupos-usados[p.ID])
	}
	if j.est.Ganadores == nil {
		j.est.Ganadores = []Ganador{}
	}
	if j.est.Especiales == nil {
		j.est.Especiales = []string{}
	}
	if j.est.AccesoWeb == nil {
		j.est.AccesoWeb = map[string]bool{}
	}
	// migración: los «pagas el X %» de antes pasan al premio web; el especial sigue siendo especial
	for id := range j.est.Consuelos {
		if !j.esEspecial(id) {
			j.est.AccesoWeb[id] = true
		}
	}
	j.est.Consuelos = nil
}

func (j *Juego) esEspecial(id string) bool {
	for _, e := range j.est.Especiales {
		if e == id {
			return true
		}
	}
	return false
}

// Cargar reemplaza el estado por uno leído del disco.
func (j *Juego) Cargar(b []byte) error {
	var e Estado
	if err := json.Unmarshal(b, &e); err != nil {
		return err
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	j.est = e
	if j.est.Jugadores == nil {
		j.est.Jugadores = []*Jugador{}
	}
	j.indexar()
	return nil
}

func (j *Juego) Volcar() ([]byte, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return json.Marshal(j.est)
}

// ---------- inscripción ----------

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

// Entrar inscribe a alguien o, si nombre y WhatsApp coinciden con alguien ya inscrito, le devuelve su jugador.
func (j *Juego) Entrar(nombreIn, telIn, ip string, semilla uint32) (*Jugador, error) {
	nombre := strings.Join(strings.Fields(nombreIn), " ")
	ll := llave(nombre)
	if strings.ContainsAny(nombre, "0123456789<>") || len(strings.Fields(ll)) < 2 || utf8.RuneCountInString(ll) < 5 || utf8.RuneCountInString(nombre) > 40 {
		return nil, ErrNombre
	}
	tel, ok := telefono(telIn)
	if !ok {
		return nil, ErrTel
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	deIP := 0
	for _, p := range j.est.Jugadores {
		if p.Tel == tel {
			if p.Llave == ll {
				return p, nil
			}
			return nil, ErrTelUsado
		}
		if p.Llave == ll {
			return nil, ErrNomUsado
		}
		if p.IP == ip {
			deIP++
		}
	}
	if j.cfg.PorIP > 0 && deIP >= j.cfg.PorIP {
		return nil, ErrIP
	}
	if j.cfg.MaxJugadores > 0 && len(j.est.Jugadores) >= j.cfg.MaxJugadores {
		return nil, ErrLleno
	}
	p := &Jugador{ID: hexAzar(4), Token: hexAzar(16), Nombre: nombre, Llave: ll, Tel: tel, IP: ip,
		Semilla: semilla, Num: 1 + j.azar(99), Lugar: len(j.est.Jugadores), Creado: j.ahora().UTC()}
	for j.porID[p.ID] != nil {
		p.ID = hexAzar(4)
	}
	j.est.Jugadores = append(j.est.Jugadores, p)
	j.porTok[p.Token] = p
	j.porID[p.ID] = p
	return p, nil
}

func (j *Juego) PorToken(tok string) *Jugador {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.porTok[tok]
}

// ---------- medidor ----------

func tri(x float64) float64 {
	f := x - math.Floor(x)
	return 1 - math.Abs(2*f-1)
}

func (j *Juego) periodo(aciertos int) int {
	ps := j.cfg.Medidor.PeriodosMs
	if aciertos >= len(ps) {
		return ps[len(ps)-1]
	}
	return ps[aciertos]
}

// NuevaFase emite la fase del próximo tiro de un jugador. espera: ms que faltan para poder tirar.
func (j *Juego) NuevaFase(id string) (*Fase, int) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.nuevaFase(id)
}

func (j *Juego) nuevaFase(id string) (*Fase, int) {
	p := j.porID[id]
	if p == nil {
		return nil, 0
	}
	m := j.cfg.Medidor
	j.nfase++
	f := &Fase{
		ID: j.nfase, P: j.periodo(p.Aciertos), H: m.MediaZona,
		C:       m.CentroMin + (m.CentroMax-m.CentroMin)*float64(j.azar(1001))/1000,
		X:       float64(j.azar(1000)) / 1000,
		V:       j.viento(id),
		Emitida: j.ahora(),
	}
	j.fases[id] = f
	return f, j.espera(id)
}

// viento: el que sopla ahora para ese jugador. Cambia cada cada_ms; lo sortea el servidor (crypto/rand) y
// nunca baja de un 25 % del máximo, para que siempre haya algo que compensar.
func (j *Juego) viento(id string) float64 {
	ahora := j.ahora()
	if w, ok := j.vient[id]; ok && ahora.Before(w.hasta) {
		return w.v
	}
	vc := j.cfg.Viento
	mag := vc.Max * (0.25 + 0.75*float64(j.azar(1001))/1000)
	if j.azar(2) == 0 {
		mag = -mag
	}
	mag = math.Round(mag*100) / 100
	j.vient[id] = vientoVivo{v: mag, hasta: ahora.Add(time.Duration(vc.CadaMs) * time.Millisecond)}
	return mag
}

// lateral: cuánto conserva la probabilidad según el desvío lateral (viento + puntería), de 1 a 0.
func (j *Juego) lateral(desvio float64) float64 {
	v := j.cfg.Viento
	d := math.Abs(desvio)
	switch {
	case d <= v.Tolerancia:
		return 1
	case d >= v.Caida:
		return 0
	}
	return 1 - (d-v.Tolerancia)/(v.Caida-v.Tolerancia)
}

// Empezar arranca la ronda de 25 s de un jugador. Una sola vez por persona.
func (j *Juego) Empezar(id string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	p := j.porID[id]
	if p == nil {
		return ErrFase
	}
	if j.est.Cerrado {
		return ErrCerrado
	}
	if !j.est.Abierto {
		return ErrPausa
	}
	if !p.Inicio.IsZero() {
		return ErrYaJugo
	}
	p.Inicio = j.ahora().UTC()
	delete(j.ultimo, id)
	delete(j.vient, id)
	return nil
}

// resta: ms que le quedan a la ronda del jugador (0 si no empezó o ya acabó). Se llama con j.mu tomado.
func (j *Juego) resta(p *Jugador) int {
	if p.Inicio.IsZero() {
		return 0
	}
	r := j.cfg.LimiteMs - int(j.ahora().Sub(p.Inicio)/time.Millisecond)
	if r < 0 {
		return 0
	}
	return r
}

func (j *Juego) espera(id string) int {
	u, ok := j.ultimo[id]
	if !ok {
		return 0
	}
	w := j.cfg.EsperaMs - int(j.ahora().Sub(u)/time.Millisecond)
	if w < 0 {
		return 0
	}
	return w
}

// probabilidad de encestar según la distancia al centro de la franja, en medias-franjas (z).
func (j *Juego) probabilidad(z float64) float64 {
	m := j.cfg.Medidor
	lerp := func(a, b, t float64) float64 { return a + (b-a)*t }
	switch {
	case z <= 0.3:
		return m.ProbPerfecto
	case z <= 1:
		return lerp(m.ProbPerfecto, m.ProbBorde, (z-0.3)/0.7)
	case z <= 2.5:
		return lerp(m.ProbBorde, m.ProbCerca, (z-1)/1.5)
	case z <= 4:
		return lerp(m.ProbCerca, m.ProbLejos, (z-2.5)/1.5)
	}
	return m.ProbLejos
}

// Tirar resuelve un tiro. e: ms que el cliente dice que pasaron desde que recibió la fase. holgura: ms de
// latencia que se le aceptan (se mide con ping). El valor del medidor sale del reloj del servidor: e se
// recorta a [transcurrido − holgura, transcurrido], así que nadie puede elegir un instante fuera de esa ventana.
// a: puntería del jugador (−1…1, negativo hacia la izquierda). El desvío del tiro es viento + a.
// Fuera de los 25 s (más la holgura de latencia) no se acepta ningún tiro.
func (j *Juego) Tirar(id string, faseID uint32, e int, a float64, holgura int) (Tiro, *Victoria, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	p := j.porID[id]
	if p == nil {
		return Tiro{}, nil, ErrFase
	}
	if j.est.Cerrado {
		return Tiro{}, nil, ErrCerrado
	}
	if !j.est.Abierto {
		return Tiro{}, nil, ErrPausa
	}
	if p.Inicio.IsZero() {
		return Tiro{}, nil, ErrNoEmpezo
	}
	if j.ahora().Sub(p.Inicio) > time.Duration(j.cfg.LimiteMs+holgura)*time.Millisecond {
		return Tiro{}, nil, ErrTiempo
	}
	f := j.fases[id]
	if f == nil || f.ID != faseID {
		return Tiro{}, nil, ErrFase
	}
	if math.IsNaN(a) || math.IsInf(a, 0) {
		a = 0
	}
	a = math.Max(-1.5, math.Min(1.5, a))
	if j.espera(id) > 0 {
		return Tiro{}, nil, ErrEspera
	}
	ahora := j.ahora()
	trans := int(ahora.Sub(f.Emitida) / time.Millisecond)
	if e > trans {
		e = trans
	}
	if e < trans-holgura {
		e = trans - holgura
	}
	if e < 0 {
		e = 0
	}
	m := tri(float64(e)/float64(f.P) + f.X)
	z := math.Abs(m-f.C) / f.H
	desvio := f.V + a
	lat := j.lateral(desvio)
	ok := j.azar(10000) < int(j.probabilidad(z)*lat*10000)
	d := int(math.Round(desvio / j.cfg.Viento.Caida * 100))
	d = max(-100, min(100, d))
	t := Tiro{Jugador: id, Ok: ok, D: d, M: int(math.Round(m * 1000)), C: int(math.Round(f.C * 1000))}
	switch {
	case !ok && lat < 0.5:
		t.R = "lado"
	case ok && z <= 0.6 && math.Abs(desvio) <= j.cfg.Viento.Tolerancia/2 && j.azar(10) < 6:
		t.R = "limpia"
	case ok:
		t.R = "aro"
	case z <= 1 || (z <= 2.5 && j.azar(2) == 0):
		t.R = "rebota"
	case m < f.C:
		t.R = "corta"
	default:
		t.R = "larga"
	}
	p.Tiros++
	if ok {
		p.Aciertos++
	}
	t.Aciertos, t.Tiros = p.Aciertos, p.Tiros
	j.ultimo[id] = ahora
	delete(j.fases, id)
	var v *Victoria
	if ok && p.Aciertos == j.cfg.Meta && p.Gano == 0 {
		v = j.premiar(p)
	}
	return t, v, nil
}

// premiar sortea un premio entre los cupos que quedan: cada cupo restante pesa lo mismo. Si el premio tiene
// grupos, también sortea el grupo. Devuelve nil si ya no quedan cupos.
func (j *Juego) premiar(p *Jugador) *Victoria {
	total := 0
	for _, pr := range j.cfg.Premios {
		if j.est.Restantes[pr.ID] > 0 {
			total += j.est.Restantes[pr.ID]
		}
	}
	if total == 0 {
		return nil
	}
	r := j.azar(total)
	var elegido *Premio
	for i := range j.cfg.Premios {
		pr := &j.cfg.Premios[i]
		n := j.est.Restantes[pr.ID]
		if n <= 0 {
			continue
		}
		if r < n {
			elegido = pr
			break
		}
		r -= n
	}
	j.est.Restantes[elegido.ID]--
	g := Ganador{Orden: len(j.est.Ganadores) + 1, Jugador: p.ID, Premio: elegido.ID, Tiros: p.Tiros, Hora: j.ahora().UTC()}
	if len(elegido.Grupos) > 0 {
		g.Grupo = elegido.Grupos[j.azar(len(elegido.Grupos))].ID
	}
	p.Gano = g.Orden
	j.est.Ganadores = append(j.est.Ganadores, g)
	// quien gana un cupo al 100 % (por ejemplo, tras reabrir) deja su descuento; el especial vuelve a sortearse al cerrar
	delete(j.est.AccesoWeb, p.ID)
	for i, id := range j.est.Especiales {
		if id == p.ID {
			j.est.Especiales = append(j.est.Especiales[:i], j.est.Especiales[i+1:]...)
			break
		}
	}
	return &Victoria{Ganador: g, Corto: corto(p.Nombre), Semilla: p.Semilla, Num: p.Num}
}

// ---------- administración ----------

func (j *Juego) Abrir(abierto bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.est.Abierto = abierto
}

func (j *Juego) Cerrar(cerrado bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.est.Cerrado = cerrado
	if cerrado {
		j.est.CerradoEn = j.ahora().UTC()
		j.consolar()
	}
}

// consolar reparte los premios de consuelo entre quienes tiraron al menos una vez, no ganaron un cupo y todavía
// no tienen premio. Cerrar dos veces no vuelve a sortear: solo entran los que aparecieron después. El descuento
// grande (especial_cupos) sale una sola vez en todo el sorteo; los demás reciben el premio web.
func (j *Juego) consolar() {
	var nuevos []*Jugador
	for _, p := range j.est.Jugadores {
		if p.Tiros >= 1 && p.Gano == 0 && !j.est.AccesoWeb[p.ID] && !j.esEspecial(p.ID) {
			nuevos = append(nuevos, p)
		}
	}
	// orden al azar (Fisher-Yates con crypto/rand)
	for i := len(nuevos) - 1; i > 0; i-- {
		k := j.azar(i + 1)
		nuevos[i], nuevos[k] = nuevos[k], nuevos[i]
	}
	c := j.cfg.Consuelo
	for _, p := range nuevos {
		if len(j.est.Especiales) < c.EspecialCupos {
			j.est.Especiales = append(j.est.Especiales, p.ID)
			continue
		}
		j.est.AccesoWeb[p.ID] = true
	}
}

// consuelo: «esp» (descuento grande, con el % que paga), «web» (premio web) o «» si no tiene.
func (j *Juego) consuelo(id string) (string, int) {
	if j.esEspecial(id) {
		return "esp", j.cfg.Consuelo.EspecialPagaPct
	}
	if j.est.AccesoWeb[id] {
		return "web", 0
	}
	return "", 0
}

func (j *Juego) Reiniciar() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.est = j.estadoNuevo()
	j.indexar()
}

func (j *Juego) Quedan() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	n := 0
	for _, v := range j.est.Restantes {
		n += v
	}
	return n
}
