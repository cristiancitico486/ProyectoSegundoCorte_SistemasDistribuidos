package fachada

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"suscripciones.local/servidor-suscripciones/dto"
	"suscripciones.local/servidor-suscripciones/repositorio"
)

func nuevaFachada(t *testing.T) (*FachadaSuscripciones, string) {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "datos", "pagos.jsonl")
	repo, err := repositorio.NuevoRepositorioPagos(ruta)
	if err != nil {
		t.Fatal(err)
	}
	return NuevaFachadaSuscripciones(repo).ConRetrasoSimulado(300 * time.Millisecond), ruta
}

func lineasDelArchivo(t *testing.T, ruta string) int {
	t.Helper()
	archivo, err := os.Open(ruta)
	if err != nil {
		return 0
	}
	defer archivo.Close()
	n := 0
	lector := bufio.NewScanner(archivo)
	for lector.Scan() {
		if len(lector.Bytes()) > 0 {
			n++
		}
	}
	return n
}

func codigo(err error) string {
	var n *ErrorNegocio
	if errors.As(err, &n) {
		return n.Codigo
	}
	return ""
}

func pago(token, usuario, plan, simulacion string) dto.PagoSolicitudDTO {
	return dto.PagoSolicitudDTO{Token: token, UsuarioId: usuario, PlanId: plan, Simulacion: simulacion}
}

func TestPagoNuevoYTokenRepetido(t *testing.T) {
	f, ruta := nuevaFachada(t)
	ctx := context.Background()

	r1, err := f.ProcesarPago(ctx, pago("tok-aaaaaaaa", "u1", "premium", ""))
	if err != nil || r1.Repetido || !r1.Almacenado || r1.Estado != "aprobado" || r1.Monto != 29900 || r1.Plan != "Premium" || r1.PagoId == "" {
		t.Fatalf("pago nuevo incorrecto: %v %+v", err, r1)
	}
	r2, err := f.ProcesarPago(ctx, pago("tok-aaaaaaaa", "u1", "premium", ""))
	if err != nil || !r2.Repetido || r2.PagoId != r1.PagoId || r2.Fecha != r1.Fecha {
		t.Fatalf("el token repetido debe devolver el resultado original: %v %+v", err, r2)
	}
	if n := lineasDelArchivo(t, ruta); n != 1 {
		t.Fatalf("debe haber un solo pago almacenado, hay %d", n)
	}
}

func TestTokenEnConflicto(t *testing.T) {
	f, _ := nuevaFachada(t)
	ctx := context.Background()
	f.ProcesarPago(ctx, pago("tok-bbbbbbbb", "u1", "basico", ""))
	if _, err := f.ProcesarPago(ctx, pago("tok-bbbbbbbb", "u2", "basico", "")); codigo(err) != "token_en_conflicto" {
		t.Fatalf("otro usuario: %v", err)
	}
	if _, err := f.ProcesarPago(ctx, pago("tok-bbbbbbbb", "u1", "premium", "")); codigo(err) != "token_en_conflicto" {
		t.Fatalf("otro plan: %v", err)
	}
}

// Riesgo clave: el cliente aborta su primer intento (timeout) y reenvía con
// el mismo token. El pago debe terminar y almacenarse igual, y el reenvío
// debe recibir el resultado original, sin cobrar dos veces.
func TestClienteAbortaYReenvia(t *testing.T) {
	f, ruta := nuevaFachada(t)

	ctx1, cancelar := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancelar()
	if _, err := f.ProcesarPago(ctx1, pago("tok-cccccccc", "u1", "estandar", "demora")); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("el primer intento debía vencer por timeout: %v", err)
	}

	// Reenvío mientras el primer pago sigue en curso: espera y recibe el mismo resultado.
	r2, err := f.ProcesarPago(context.Background(), pago("tok-cccccccc", "u1", "estandar", "demora"))
	if err != nil || !r2.Repetido || !r2.Almacenado || r2.PagoId == "" {
		t.Fatalf("el reenvío debe recibir el resultado del primer pago: %v %+v", err, r2)
	}
	// Otro reenvío, ya terminado: resultado original inmediato.
	r3, err := f.ProcesarPago(context.Background(), pago("tok-cccccccc", "u1", "estandar", ""))
	if err != nil || !r3.Repetido || r3.PagoId != r2.PagoId {
		t.Fatalf("tercer envío: %v %+v", err, r3)
	}
	if n := lineasDelArchivo(t, ruta); n != 1 {
		t.Fatalf("un solo cobro almacenado, hay %d", n)
	}
}

func TestClienteAbortaYElPagoQuedaAlmacenadoSinReenvio(t *testing.T) {
	f, ruta := nuevaFachada(t)
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelar()
	f.ProcesarPago(ctx, pago("tok-dddddddd", "u1", "basico", "demora"))
	f.EsperarPendientes(2 * time.Second)
	if n := lineasDelArchivo(t, ruta); n != 1 {
		t.Fatalf("el pago debía guardarse aunque el cliente abortó: %d", n)
	}
}

func TestEnviosConcurrentesCobranUnaSolaVez(t *testing.T) {
	f, ruta := nuevaFachada(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	nuevos, repetidos := 0, 0
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := f.ProcesarPago(context.Background(), pago("tok-eeeeeeee", "u1", "premium", "demora"))
			if err != nil {
				t.Errorf("error inesperado: %v", err)
				return
			}
			mu.Lock()
			if r.Repetido {
				repetidos++
			} else {
				nuevos++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	if nuevos != 1 || repetidos != 29 {
		t.Fatalf("debe haber 1 pago nuevo y 29 repetidos: %d y %d", nuevos, repetidos)
	}
	if n := lineasDelArchivo(t, ruta); n != 1 {
		t.Fatalf("un solo cobro almacenado, hay %d", n)
	}
}

func TestRechazoNoConsumeElToken(t *testing.T) {
	f, ruta := nuevaFachada(t)
	ctx := context.Background()
	if _, err := f.ProcesarPago(ctx, pago("tok-ffffffff", "u1", "premium", "rechazo")); codigo(err) != "pago_rechazado" {
		t.Fatalf("debía rechazarse: %v", err)
	}
	if n := lineasDelArchivo(t, ruta); n != 0 {
		t.Fatalf("un pago rechazado no se almacena: %d", n)
	}
	r, err := f.ProcesarPago(ctx, pago("tok-ffffffff", "u1", "premium", ""))
	if err != nil || r.Repetido {
		t.Fatalf("tras el rechazo el mismo token debe procesarse como nuevo: %v %+v", err, r)
	}
}

func TestFalloAlAlmacenarNoConsumeElToken(t *testing.T) {
	f, ruta := nuevaFachada(t)
	// Rompe el almacenamiento: en lugar de la carpeta de datos hay un archivo.
	carpeta := filepath.Dir(ruta)
	os.RemoveAll(carpeta)
	if err := os.WriteFile(carpeta, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := f.ProcesarPago(context.Background(), pago("tok-gggggggg", "u1", "basico", "")); codigo(err) != "error_interno" {
		t.Fatalf("debía dar error_interno: %v", err)
	}
	// Se repara el almacenamiento y el mismo token se procesa como nuevo.
	os.Remove(carpeta)
	os.MkdirAll(carpeta, 0755)
	r, err := f.ProcesarPago(context.Background(), pago("tok-gggggggg", "u1", "basico", ""))
	if err != nil || r.Repetido {
		t.Fatalf("el token no debía consumirse: %v %+v", err, r)
	}
}

func TestPersistenciaTrasReiniciar(t *testing.T) {
	f, ruta := nuevaFachada(t)
	r1, _ := f.ProcesarPago(context.Background(), pago("tok-hhhhhhhh", "u1", "premium", ""))

	repo2, err := repositorio.NuevoRepositorioPagos(ruta)
	if err != nil {
		t.Fatal(err)
	}
	f2 := NuevaFachadaSuscripciones(repo2)
	r2, err := f2.ProcesarPago(context.Background(), pago("tok-hhhhhhhh", "u1", "premium", ""))
	if err != nil || !r2.Repetido || r2.PagoId != r1.PagoId {
		t.Fatalf("tras reiniciar el token sigue usado: %v %+v", err, r2)
	}
}

func TestValidaciones(t *testing.T) {
	f, _ := nuevaFachada(t)
	ctx := context.Background()
	casos := []struct {
		nombre string
		e      dto.PagoSolicitudDTO
		codigo string
	}{
		{"token corto", pago("abc", "u1", "premium", ""), "token_invalido"},
		{"token con espacios", pago("tok con espacio", "u1", "premium", ""), "token_invalido"},
		{"token largo", pago("tok-"+string(make([]byte, 70)), "u1", "premium", ""), "token_invalido"},
		{"usuario vacío", pago("tok-iiiiiiii", "  ", "premium", ""), "usuario_invalido"},
		{"plan inexistente", pago("tok-iiiiiiii", "u1", "oro", ""), "plan_invalido"},
		{"simulación desconocida", pago("tok-iiiiiiii", "u1", "premium", "lluvia"), "cuerpo_invalido"},
	}
	for _, c := range casos {
		if _, err := f.ProcesarPago(ctx, c.e); codigo(err) != c.codigo {
			t.Errorf("%s: esperaba %s, obtuve %v", c.nombre, c.codigo, err)
		}
	}
	if _, err := f.ProcesarPago(ctx, pago("tok-iiiiiiii", "u1", "premium", "ninguna")); err != nil {
		t.Errorf(`simulacion "ninguna" debe aceptarse: %v`, err)
	}
}

func TestConsultarSuscripcion(t *testing.T) {
	f, _ := nuevaFachada(t)
	ctx := context.Background()

	vacia, err := f.ConsultarSuscripcion("nadie")
	if err != nil || vacia.Activa || vacia.PlanId != "" || vacia.Pagos == nil || len(vacia.Pagos) != 0 {
		t.Fatalf("usuario sin pagos: %v %+v", err, vacia)
	}

	reloj := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	f.ahora = func() time.Time { return reloj }
	f.ProcesarPago(ctx, pago("tok-jjjjjjjj", "u1", "basico", ""))
	reloj = reloj.Add(48 * time.Hour)
	f.ProcesarPago(ctx, pago("tok-kkkkkkkk", "u1", "premium", ""))

	s, err := f.ConsultarSuscripcion("u1")
	if err != nil || !s.Activa || s.PlanId != "premium" || s.Plan != "Premium" || len(s.Pagos) != 2 {
		t.Fatalf("suscripción: %v %+v", err, s)
	}
	if s.Pagos[0].PlanId != "premium" || s.Pagos[1].PlanId != "basico" {
		t.Fatalf("pagos del más reciente al más antiguo: %+v", s.Pagos)
	}
	// El último pago fue el 3 nov a las 10:00 UTC: vigente hasta el 3 dic.
	if s.VigenteHasta != "2026-12-03T10:00:00Z" {
		t.Fatalf("vigenteHasta debe ser 30 días tras el último pago: %s", s.VigenteHasta)
	}

	// 31 días después ya no está activa.
	reloj = reloj.Add(31 * 24 * time.Hour)
	if s, _ := f.ConsultarSuscripcion("u1"); s.Activa {
		t.Fatalf("pasados 30 días no debe estar activa: %+v", s)
	}
	if _, err := f.ConsultarSuscripcion("   "); codigo(err) != "usuario_invalido" {
		t.Fatalf("usuario vacío: %v", err)
	}
}
