package controlador

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"estadisticas.local/servidor-estadisticas/consumidor"
	"estadisticas.local/servidor-estadisticas/dto"
	"estadisticas.local/servidor-estadisticas/fachada"
	"estadisticas.local/servidor-estadisticas/repositorio"
)

func TestFlujoColaYConsultaREST(t *testing.T) {
	repo, err := repositorio.NuevoRepositorioReproducciones(filepath.Join(t.TempDir(), "r.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	ctrl := NuevoControladorEstadisticas(fachada.NuevaFachadaEstadisticas(repo), consumidor.NuevoConsumidorCola())
	mux := http.NewServeMux()
	ctrl.RegistrarRutas(mux)

	// Mensajes tal como los publica el ServidorDeStreaming (uno sin usuario).
	for _, m := range []string{
		`{"audioId":2,"usuarioId":"u9","titulo":"Blinding Lights","tipo":"Música","fecha":"2026-11-02T10:00:00Z"}`,
		`{"audioId":3,"usuarioId":"u9","titulo":"Episodio 1","tipo":"Podcasts","fecha":"2026-11-02T10:01:00Z"}`,
		`{"audioId":2,"usuarioId":"u9","titulo":"Blinding Lights","tipo":"Música","fecha":"2026-11-02T10:02:00Z"}`,
		`{"audioId":2,"titulo":"Blinding Lights","tipo":"Música","fecha":"2026-11-02T10:03:00Z"}`,
	} {
		if err := ctrl.atenderMensaje([]byte(m)); err != nil {
			t.Fatal(err)
		}
	}

	pedir := func(ruta string) (int, []byte) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ruta, nil))
		return rec.Code, rec.Body.Bytes()
	}

	code, cuerpo := pedir("/estadisticas/historial/u9")
	var h dto.HistorialDTO
	if code != 200 || json.Unmarshal(cuerpo, &h) != nil {
		t.Fatalf("respuesta inesperada %d: %s", code, cuerpo)
	}
	if h.UsuarioId != "u9" || h.TotalReproducciones != 3 || len(h.Audios) != 2 {
		t.Fatalf("historial incorrecto: %s", cuerpo)
	}
	// Orden: última reproducción descendente -> audio 2 (10:02) antes que audio 3 (10:01).
	if h.Audios[0].AudioId != 2 || h.Audios[0].Cantidad != 2 || h.Audios[1].AudioId != 3 {
		t.Fatalf("orden o cantidades incorrectos: %s", cuerpo)
	}
	t.Logf("historial u9: %s", strings.TrimSpace(string(cuerpo)))

	code, cuerpo = pedir("/estadisticas/historial/nadie")
	if code != 200 || !strings.Contains(string(cuerpo), `"audios":[]`) {
		t.Fatalf("usuario sin historial debe dar 200 con audios []: %d %s", code, cuerpo)
	}
	if code, cuerpo = pedir("/health"); code != 200 || !strings.Contains(string(cuerpo), `"status":"ok"`) {
		t.Fatalf("health: %d %s", code, cuerpo)
	}
	if code, cuerpo = pedir("/estadisticas/historial/%20"); code != 400 || !strings.Contains(string(cuerpo), `"error":"usuario_invalido"`) {
		t.Fatalf("usuario vacío debe dar 400 usuario_invalido: %d %s", code, cuerpo)
	}
}
