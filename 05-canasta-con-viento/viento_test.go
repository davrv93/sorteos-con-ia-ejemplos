package main

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
	"time"
)

// Reglas nuevas con config.json tal cual: 6 encestes en una sola ronda de 25 s, con viento.
func juegoViento(t *testing.T) (*Juego, *reloj) {
	t.Helper()
	var c Config
	if err := json.Unmarshal(configBase, &c); err != nil {
		t.Fatal(err)
	}
	if err := c.validar(); err != nil {
		t.Fatal(err)
	}
	c.AbiertoAlIni = true
	c.PorIP = 0
	r := &reloj{t: time.Date(2026, 11, 23, 21, 0, 0, 0, lima)}
	j := NuevoJuego(c)
	j.ahora = r.ahora
	return j, r
}

func jugadorSinEmpezar(t *testing.T, j *Juego, i int) *Jugador {
	t.Helper()
	p, err := j.Entrar("Viento Prueba"+string(rune('a'+i)), "9"+pad8(i), "1.1.1.1", uint32(i))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// eCentro: el instante (ms desde la fase) en que el medidor pasa por el centro de la franja.
func eCentro(f *Fase) int {
	u := f.C/2 - f.X
	u -= math.Floor(u)
	return int(math.Round(u * float64(f.P)))
}

// proximoCentro: primer instante ≥ desde (ms) en que el medidor cruza el centro (subiendo o bajando).
func proximoCentro(f *Fase, desde int) int {
	P := float64(f.P)
	mejor := math.Inf(1)
	for _, u0 := range []float64{f.C/2 - f.X, 1 - f.C/2 - f.X} {
		u := u0 - math.Floor(u0)
		e := u * P
		for e < float64(desde) {
			e += P
		}
		mejor = math.Min(mejor, e)
	}
	return int(math.Round(mejor))
}

func TestUnaSolaRonda(t *testing.T) {
	j, _ := juegoViento(t)
	p := jugadorSinEmpezar(t, j, 0)
	f, _ := j.NuevaFase(p.ID)
	if _, _, err := j.Tirar(p.ID, f.ID, 0, 0, 300); !errors.Is(err, ErrNoEmpezo) {
		t.Fatalf("sin empezar: %v", err)
	}
	if err := j.Empezar(p.ID); err != nil {
		t.Fatal(err)
	}
	if err := j.Empezar(p.ID); !errors.Is(err, ErrYaJugo) {
		t.Fatalf("segunda ronda: %v", err)
	}
	j.Abrir(false)
	q := jugadorSinEmpezar(t, j, 1)
	if err := j.Empezar(q.ID); !errors.Is(err, ErrPausa) {
		t.Fatalf("empezar en pausa: %v", err)
	}
}

func TestReloj25s(t *testing.T) {
	j, r := juegoViento(t)
	p := jugadorSinEmpezar(t, j, 0)
	if err := j.Empezar(p.ID); err != nil {
		t.Fatal(err)
	}
	r.pasar(24900 * time.Millisecond)
	f, _ := j.NuevaFase(p.ID)
	if _, _, err := j.Tirar(p.ID, f.ID, 0, -f.V, 300); err != nil {
		t.Fatalf("a los 24,9 s: %v", err)
	}
	r.pasar(1500 * time.Millisecond) // 26,4 s: fuera aun con 300 ms de holgura
	f, _ = j.NuevaFase(p.ID)
	if _, _, err := j.Tirar(p.ID, f.ID, 0, -f.V, 300); !errors.Is(err, ErrTiempo) {
		t.Fatalf("a los 26,4 s: %v", err)
	}
	j.mu.Lock()
	resta := j.resta(p)
	j.mu.Unlock()
	if resta != 0 {
		t.Fatalf("resta %d tras el tiempo", resta)
	}
}

// El viento sopla entre el 25 % y el 100 % del máximo, a veces a cada lado, y cambia cada cada_ms.
func TestVientoCambia(t *testing.T) {
	j, r := juegoViento(t)
	p := jugadorSinEmpezar(t, j, 0)
	izq, der, cambios := 0, 0, 0
	f, _ := j.NuevaFase(p.ID)
	prev := f.V
	for i := 0; i < 400; i++ {
		r.pasar(time.Duration(j.cfg.Viento.CadaMs/2) * time.Millisecond)
		f, _ = j.NuevaFase(p.ID)
		if a := math.Abs(f.V); a < 0.25*j.cfg.Viento.Max-1e-9 || a > j.cfg.Viento.Max+1e-9 {
			t.Fatalf("viento %v fuera de rango", f.V)
		}
		if f.V < 0 {
			izq++
		} else {
			der++
		}
		if f.V != prev {
			cambios++
		}
		prev = f.V
	}
	if izq < 120 || der < 120 {
		t.Fatalf("viento sesgado: %d izq, %d der", izq, der)
	}
	// con medio periodo por fase, cambia como mucho cada dos fases
	if cambios < 150 || cambios > 205 {
		t.Fatalf("cambios = %d de 400 (cada %d ms)", cambios, j.cfg.Viento.CadaMs)
	}
}

// Compensar el viento apuntando funciona; ignorarlo con viento fuerte no.
func TestCompensarViento(t *testing.T) {
	j, r := juegoViento(t)
	j.cfg.Meta = 1 << 30
	j.cfg.LimiteMs = 1 << 30
	p := jugadorSinEmpezar(t, j, 0)
	if err := j.Empezar(p.ID); err != nil {
		t.Fatal(err)
	}
	tasa := func(apuntar func(v float64) float64, soloFuerte bool) float64 {
		ok, n := 0, 0
		for n < 3000 {
			r.pasar(2 * time.Second)
			f, _ := j.NuevaFase(p.ID)
			if soloFuerte && math.Abs(f.V) < j.cfg.Viento.Caida {
				continue
			}
			e := eCentro(f)
			r.pasar(time.Duration(e) * time.Millisecond)
			tr, _, err := j.Tirar(p.ID, f.ID, e, apuntar(f.V), 300)
			if err != nil {
				t.Fatal(err)
			}
			n++
			if tr.Ok {
				ok++
			}
			if !tr.Ok && math.Abs(f.V+apuntar(f.V)) >= j.cfg.Viento.Caida && tr.R != "lado" {
				t.Fatalf("fallo lateral marcado %q", tr.R)
			}
		}
		return float64(ok) / float64(n)
	}
	comp := tasa(func(v float64) float64 { return -v }, false)
	want := j.cfg.Medidor.ProbPerfecto
	if math.Abs(comp-want) > 0.05 {
		t.Fatalf("compensando entra %.2f, quería ≈ %.2f (prob_perfecto)", comp, want)
	}
	sin := tasa(func(v float64) float64 { return 0 }, true)
	if sin != 0 {
		t.Fatalf("sin compensar un viento fuerte entra %.2f, quería 0", sin)
	}
	medio := tasa(func(v float64) float64 { return -v * 0.6 }, false)
	if medio >= comp {
		t.Fatalf("compensar a medias (%.2f) no debería rendir como compensar bien (%.2f)", medio, comp)
	}
}

// Con buen pulso y compensando el viento, 6 en 25 s se logra casi siempre; a ciegas, casi nunca.
func TestSeisEn25(t *testing.T) {
	jugar := func(bueno bool) (ganas int) {
		const n = 400
		for k := 0; k < n; k++ {
			j, r := juegoViento(t)
			p := jugadorSinEmpezar(t, j, 0)
			if err := j.Empezar(p.ID); err != nil {
				t.Fatal(err)
			}
			for {
				f, w := j.NuevaFase(p.ID)
				desde := w + 150 // espera entre tiros + reacción humana
				e, a := proximoCentro(f, desde), -f.V
				if !bueno {
					e, a = desde+j.azar(f.P), 0
				}
				r.pasar(time.Duration(e) * time.Millisecond)
				_, v, err := j.Tirar(p.ID, f.ID, e, a, 300)
				if errors.Is(err, ErrTiempo) {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				if v != nil || p.Aciertos >= j.cfg.Meta {
					ganas++
					break
				}
			}
		}
		return ganas
	}
	if g := jugar(true); g < 300 {
		t.Fatalf("un buen jugador gana %d de 400", g)
	}
	if g := jugar(false); g > 8 {
		t.Fatalf("a ciegas se gana %d de 400", g)
	}
}

func TestPunteriaInvalida(t *testing.T) {
	j, r := juegoViento(t)
	p := jugadorSinEmpezar(t, j, 0)
	if err := j.Empezar(p.ID); err != nil {
		t.Fatal(err)
	}
	f, _ := j.NuevaFase(p.ID)
	r.pasar(200 * time.Millisecond)
	tr, _, err := j.Tirar(p.ID, f.ID, 200, math.NaN(), 300)
	if err != nil {
		t.Fatal(err)
	}
	if tr.D < -100 || tr.D > 100 {
		t.Fatalf("desvío %d fuera de rango", tr.D)
	}
}
