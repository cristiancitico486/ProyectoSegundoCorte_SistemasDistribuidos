package controlador

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"audios.local/servidor-audios/fachada"
	"audios.local/servidor-audios/repositorio"
)

// nuevoMux trabaja dentro de una carpeta temporal: el repositorio guarda los
// mp3 en "audios/" relativa al directorio actual.
func nuevoMux(t *testing.T) *http.ServeMux {
	t.Helper()
	anterior, _ := os.Getwd()
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(anterior) })

	mux := http.NewServeMux()
	NuevoControladorAudios(fachada.NuevaFachadaAudios(repositorio.NuevoRepositorioAudios())).RegistrarRutas(mux)
	return mux
}

func subir(mux *http.ServeMux, ruta, campo string, contenido []byte) (int, string) {
	var cuerpo bytes.Buffer
	mp := multipart.NewWriter(&cuerpo)
	if campo != "" {
		parte, _ := mp.CreateFormFile(campo, "x.mp3")
		parte.Write(contenido)
	}
	mp.Close()
	req := httptest.NewRequest("POST", ruta, &cuerpo)
	req.Header.Set("Content-Type", mp.FormDataContentType())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func pedir(mux *http.ServeMux, ruta string) (int, string, string) {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", ruta, nil))
	return rec.Code, rec.Body.String(), rec.Header().Get("Content-Type")
}

func TestHealth(t *testing.T) {
	mux := nuevoMux(t)
	if code, cuerpo, _ := pedir(mux, "/health"); code != 200 || !strings.Contains(cuerpo, `"status":"ok"`) {
		t.Fatalf("health: %d %s", code, cuerpo)
	}
}

func TestSubirYDescargarMp3(t *testing.T) {
	mux := nuevoMux(t)
	mp3 := append([]byte("ID3"), bytes.Repeat([]byte{0x01}, 100)...)

	code, cuerpo := subir(mux, "/audios/5", "archivo", mp3)
	if code != 201 || !strings.Contains(cuerpo, `"nombreMp3":"5.mp3"`) {
		t.Fatalf("subida válida: %d %s", code, cuerpo)
	}
	code, descargado, tipo := pedir(mux, "/audios/5/archivo")
	if code != 200 || descargado != string(mp3) || tipo != "audio/mpeg" {
		t.Fatalf("descarga: %d tipo=%s", code, tipo)
	}
}

func TestErroresDeSubida(t *testing.T) {
	mux := nuevoMux(t)
	casos := []struct {
		nombre, ruta, campo string
		contenido           []byte
		status              int
		codigo              string
	}{
		{"no es mp3", "/audios/1", "archivo", []byte("esto es texto"), 400, "archivo_no_mp3"},
		{"vacío", "/audios/1", "archivo", nil, 400, "archivo_vacio"},
		{"sin campo", "/audios/1", "", nil, 400, "archivo_requerido"},
		{"id no numérico", "/audios/abc", "archivo", []byte("ID3xx"), 400, "id_invalido"},
		{"id cero", "/audios/0", "archivo", []byte("ID3xx"), 400, "id_invalido"},
		{"id negativo", "/audios/-5", "archivo", []byte("ID3xx"), 400, "id_invalido"},
	}
	for _, c := range casos {
		code, cuerpo := subir(mux, c.ruta, c.campo, c.contenido)
		if code != c.status || !strings.Contains(cuerpo, `"error":"`+c.codigo+`"`) || !strings.Contains(cuerpo, `"mensaje"`) {
			t.Errorf("%s: %d %s", c.nombre, code, cuerpo)
		}
	}
	if code, cuerpo, _ := pedir(mux, "/audios/99/archivo"); code != 404 || !strings.Contains(cuerpo, `"error":"audio_no_encontrado"`) {
		t.Errorf("archivo inexistente: %d %s", code, cuerpo)
	}
	if _, err := os.Stat("audios/-5.mp3"); err == nil {
		t.Errorf("no debe guardar archivos con id inválido")
	}
}
