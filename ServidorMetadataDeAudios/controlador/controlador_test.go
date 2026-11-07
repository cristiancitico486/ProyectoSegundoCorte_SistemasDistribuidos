package controlador

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"metadatos.local/servidor-metadata/dto"
	"metadatos.local/servidor-metadata/fachada"
	"metadatos.local/servidor-metadata/repositorio"
)

const catalogoSemilla = `{"tipos":[{"id":1,"nombre":"Música"},{"id":2,"nombre":"Podcasts"}],"audios":[{"id":1,"tipoId":1,"titulo":"A","metadatos":{"Artista Principal":"Q"},"nombreMp3":"1.mp3"}],"siguienteId":2}`

func nuevoMux(t *testing.T) *http.ServeMux {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "catalogo.json")
	if err := os.WriteFile(ruta, []byte(catalogoSemilla), 0644); err != nil {
		t.Fatal(err)
	}
	ctrl := NuevoControladorMetadatos(fachada.NuevaFachadaMetadatos(repositorio.NuevoRepositorioEnRuta(ruta)))
	mux := http.NewServeMux()
	ctrl.RegistrarRutas(mux)
	return mux
}

func llamar(mux *http.ServeMux, metodo, ruta, cuerpo string) (int, []byte) {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo)))
	return rec.Code, rec.Body.Bytes()
}

func TestHealthYRutasExistentes(t *testing.T) {
	mux := nuevoMux(t)
	if code, cuerpo := llamar(mux, "GET", "/health", ""); code != 200 || !strings.Contains(string(cuerpo), `"status":"ok"`) {
		t.Fatalf("health: %d %s", code, cuerpo)
	}
	if code, _ := llamar(mux, "GET", "/tipos", ""); code != 200 {
		t.Fatalf("/tipos: %d", code)
	}
	if code, _ := llamar(mux, "GET", "/tipos/1/audios", ""); code != 200 {
		t.Fatalf("/tipos/1/audios: %d", code)
	}
	if code, _ := llamar(mux, "GET", "/audios/1", ""); code != 200 {
		t.Fatalf("/audios/1: %d", code)
	}
}

func TestCampos(t *testing.T) {
	mux := nuevoMux(t)
	code, cuerpo := llamar(mux, "GET", "/tipos/1/campos", "")
	var resp dto.CamposTipoDTO
	if code != 200 || json.Unmarshal(cuerpo, &resp) != nil || resp.Tipo != "Música" || len(resp.Campos) < 3 || !resp.Campos[0].Obligatorio {
		t.Fatalf("campos: %d %s", code, cuerpo)
	}
	t.Logf("campos tipo 1: %s", strings.TrimSpace(string(cuerpo)))
	if code, cuerpo = llamar(mux, "GET", "/tipos/99/campos", ""); code != 404 || !strings.Contains(string(cuerpo), `"error":"tipo_no_encontrado"`) {
		t.Fatalf("tipo inexistente: %d %s", code, cuerpo)
	}
	if code, cuerpo = llamar(mux, "GET", "/tipos/abc/campos", ""); code != 400 || !strings.Contains(string(cuerpo), `"error":"id_invalido"`) {
		t.Fatalf("id inválido: %d %s", code, cuerpo)
	}
}

func TestRegistrarAudioValidacionYErrores(t *testing.T) {
	mux := nuevoMux(t)

	code, cuerpo := llamar(mux, "POST", "/audios", `{"tipoId":2,"titulo":"Ep","metadatos":{"Nombre del Podcast":"P"}}`)
	var e dto.ErrorDTO
	if code != 400 || json.Unmarshal(cuerpo, &e) != nil || e.Error != "metadatos_invalidos" || len(e.Faltantes) != 2 || e.Mensaje == "" {
		t.Fatalf("faltantes: %d %s", code, cuerpo)
	}
	t.Logf("error de validación: %s", strings.TrimSpace(string(cuerpo)))

	code, cuerpo = llamar(mux, "POST", "/audios", `{"tipoId":2,"titulo":"Ep","metadatos":{"Nombre del Podcast":"P","Título del Episodio":"T","Anfitrión":"H"}}`)
	var creado dto.AudioCreadoDTO
	if code != 201 || json.Unmarshal(cuerpo, &creado) != nil || creado.Id != 2 {
		t.Fatalf("registro válido: %d %s", code, cuerpo)
	}
	if code, _ = llamar(mux, "GET", "/audios/2", ""); code != 200 {
		t.Fatalf("el audio recién registrado debe consultarse: %d", code)
	}

	if code, cuerpo = llamar(mux, "POST", "/audios", `{no es json`); code != 400 || !strings.Contains(string(cuerpo), `"error":"cuerpo_invalido"`) {
		t.Fatalf("json inválido: %d %s", code, cuerpo)
	}
	if code, cuerpo = llamar(mux, "POST", "/audios", `{"tipoId":42,"titulo":"x"}`); code != 400 || !strings.Contains(string(cuerpo), `"error":"tipo_invalido"`) {
		t.Fatalf("tipo inválido: %d %s", code, cuerpo)
	}
	if code, cuerpo = llamar(mux, "GET", "/audios/77", ""); code != 404 || !strings.Contains(string(cuerpo), `"error":"audio_no_encontrado"`) {
		t.Fatalf("audio inexistente: %d %s", code, cuerpo)
	}
}
