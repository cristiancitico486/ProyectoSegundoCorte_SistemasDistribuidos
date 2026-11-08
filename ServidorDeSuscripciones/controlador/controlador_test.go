package controlador

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"suscripciones.local/servidor-suscripciones/dto"
	"suscripciones.local/servidor-suscripciones/fachada"
	"suscripciones.local/servidor-suscripciones/repositorio"
)

func nuevoServidor(t *testing.T) (*httptest.Server, *fachada.FachadaSuscripciones) {
	t.Helper()
	repo, err := repositorio.NuevoRepositorioPagos(filepath.Join(t.TempDir(), "pagos.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	fach := fachada.NuevaFachadaSuscripciones(repo).ConRetrasoSimulado(400 * time.Millisecond)
	mux := http.NewServeMux()
	NuevoControladorSuscripciones(fach).RegistrarRutas(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, fach
}

func enviar(t *testing.T, cliente *http.Client, metodo, url, cuerpo string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(metodo, url, strings.NewReader(cuerpo))
	req.Header.Set("Content-Type", "application/json")
	resp, err := cliente.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	datos, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(datos)
}

func TestPlanesYHealth(t *testing.T) {
	srv, _ := nuevoServidor(t)
	code, cuerpo := enviar(t, http.DefaultClient, "GET", srv.URL+"/planes", "")
	var planes []dto.PlanDTO
	if code != 200 || json.Unmarshal([]byte(cuerpo), &planes) != nil || len(planes) != 3 ||
		planes[0].Id != "basico" || planes[2].Precio != 29900 || planes[0].Moneda != "COP" || planes[0].DuracionDias != 30 {
		t.Fatalf("planes: %d %s", code, cuerpo)
	}
	if code, cuerpo = enviar(t, http.DefaultClient, "GET", srv.URL+"/health", ""); code != 200 || !strings.Contains(cuerpo, `"status":"ok"`) {
		t.Fatalf("health: %d %s", code, cuerpo)
	}
}

func TestCodigosHTTPDelPago(t *testing.T) {
	srv, _ := nuevoServidor(t)
	c := http.DefaultClient
	cuerpoPago := func(tok, usuario, plan, sim string) string {
		return `{"token":"` + tok + `","usuarioId":"` + usuario + `","planId":"` + plan + `","simulacion":"` + sim + `"}`
	}

	code, cuerpo := enviar(t, c, "POST", srv.URL+"/pagos", cuerpoPago("tok-12345678", "u1", "premium", ""))
	var r dto.PagoRespuestaDTO
	if code != 201 || json.Unmarshal([]byte(cuerpo), &r) != nil || r.Repetido || !r.Almacenado || r.Estado != "aprobado" || r.VigenteHasta == "" {
		t.Fatalf("nuevo: %d %s", code, cuerpo)
	}
	t.Logf("pago nuevo: %s", strings.TrimSpace(cuerpo))

	code, cuerpo = enviar(t, c, "POST", srv.URL+"/pagos", cuerpoPago("tok-12345678", "u1", "premium", ""))
	var r2 dto.PagoRespuestaDTO
	json.Unmarshal([]byte(cuerpo), &r2)
	if code != 200 || !r2.Repetido || r2.PagoId != r.PagoId {
		t.Fatalf("repetido: %d %s", code, cuerpo)
	}
	if code, cuerpo = enviar(t, c, "POST", srv.URL+"/pagos", cuerpoPago("tok-12345678", "u2", "premium", "")); code != 409 || !strings.Contains(cuerpo, `"error":"token_en_conflicto"`) {
		t.Fatalf("conflicto: %d %s", code, cuerpo)
	}
	if code, cuerpo = enviar(t, c, "POST", srv.URL+"/pagos", cuerpoPago("tok-87654321", "u1", "basico", "rechazo")); code != 402 || !strings.Contains(cuerpo, `"error":"pago_rechazado"`) || !strings.Contains(cuerpo, `"mensaje"`) {
		t.Fatalf("rechazo: %d %s", code, cuerpo)
	}
	for _, caso := range []struct{ cuerpo, codigo string }{
		{`{no es json`, "cuerpo_invalido"},
		{cuerpoPago("corto", "u1", "premium", ""), "token_invalido"},
		{cuerpoPago("tok-abcdefgh", "", "premium", ""), "usuario_invalido"},
		{cuerpoPago("tok-abcdefgh", "u1", "oro", ""), "plan_invalido"},
	} {
		if code, cuerpo = enviar(t, c, "POST", srv.URL+"/pagos", caso.cuerpo); code != 400 || !strings.Contains(cuerpo, `"error":"`+caso.codigo+`"`) {
			t.Errorf("%s: %d %s", caso.codigo, code, cuerpo)
		}
	}

	code, cuerpo = enviar(t, c, "GET", srv.URL+"/suscripciones/u1", "")
	var s dto.SuscripcionDTO
	if code != 200 || json.Unmarshal([]byte(cuerpo), &s) != nil || !s.Activa || s.PlanId != "premium" || len(s.Pagos) != 1 {
		t.Fatalf("suscripción: %d %s", code, cuerpo)
	}
	code, cuerpo = enviar(t, c, "GET", srv.URL+"/suscripciones/nadie", "")
	if code != 200 || !strings.Contains(cuerpo, `"activa":false`) || !strings.Contains(cuerpo, `"pagos":[]`) {
		t.Fatalf("sin pagos: %d %s", code, cuerpo)
	}
}

// Escenario completo de la demostración: el cliente aborta el primer intento
// a los 150 ms (en producción son 2 s), reenvía con el MISMO token y recibe
// el resultado original.
func TestReintentoDelClienteConDemora(t *testing.T) {
	srv, fach := nuevoServidor(t)
	cuerpo := `{"token":"tok-demora01","usuarioId":"u9","planId":"estandar","simulacion":"demora"}`

	corto := &http.Client{Timeout: 150 * time.Millisecond}
	if _, err := corto.Post(srv.URL+"/pagos", "application/json", strings.NewReader(cuerpo)); err == nil {
		t.Fatalf("el primer intento debía vencer por timeout")
	}
	code, texto := enviar(t, &http.Client{Timeout: 3 * time.Second}, "POST", srv.URL+"/pagos", cuerpo)
	var r dto.PagoRespuestaDTO
	json.Unmarshal([]byte(texto), &r)
	if code != 200 || !r.Repetido || !r.Almacenado || r.PagoId == "" {
		t.Fatalf("el reintento debe recibir el resultado original: %d %s", code, texto)
	}
	fach.EsperarPendientes(time.Second)
	if s, _ := fach.ConsultarSuscripcion("u9"); len(s.Pagos) != 1 {
		t.Fatalf("un solo cobro: %+v", s)
	}
}
