package repositorio

import (
	"os"
	"path/filepath"
	"testing"

	"estadisticas.local/servidor-estadisticas/modelo"
)

// Formato antiguo (claves PascalCase, sin usuario) + líneas con usuario.
const lineasIniciales = `{"AudioId":1,"Titulo":"Bohemian Rhapsody","Tipo":"Música","Fecha":"2026-09-28T21:40:08-05:00"}
{"AudioId":1,"Titulo":"Bohemian Rhapsody","Tipo":"Música","Fecha":"2026-11-01T10:00:00-05:00","UsuarioId":"u1"}
{"AudioId":1,"Titulo":"Bohemian Rhapsody","Tipo":"Música","Fecha":"2026-11-01T10:05:00-05:00","UsuarioId":"u1"}
{"AudioId":3,"Titulo":"Episodio 1","Tipo":"Podcasts","Fecha":"2026-11-01T10:10:00-05:00","UsuarioId":"u1"}
esto no es json
{"AudioId":3,"Titulo":"Episodio 1","Tipo":"Podcasts","Fecha":"2026-11-01T11:00:00-05:00","UsuarioId":"u2"}
`

func nuevoRepoConDatos(t *testing.T) (*RepositorioReproducciones, string) {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "datos", "reproducciones.jsonl")
	if err := os.MkdirAll(filepath.Dir(ruta), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(lineasIniciales), 0644); err != nil {
		t.Fatal(err)
	}
	repo, err := NuevoRepositorioReproducciones(ruta)
	if err != nil {
		t.Fatal(err)
	}
	return repo, ruta
}

func TestReconstruyeDesdeArchivo(t *testing.T) {
	repo, _ := nuevoRepoConDatos(t)

	resumen := repo.ObtenerResumen()
	if resumen.TotalReproducciones != 5 { // la línea inválida se omite
		t.Fatalf("total esperado 5, obtenido %d", resumen.TotalReproducciones)
	}
	if resumen.PorTipo["Música"] != 3 || resumen.PorTipo["Podcasts"] != 2 {
		t.Fatalf("contadores por tipo incorrectos: %v", resumen.PorTipo)
	}

	// u1: audio 3 (10:10) es más reciente que audio 1 (10:05) -> va primero.
	h := repo.ObtenerHistorial("u1")
	if h.TotalReproducciones != 3 || len(h.Audios) != 2 {
		t.Fatalf("historial de u1 incorrecto: %+v", h)
	}
	if h.Audios[0].AudioId != 3 || h.Audios[0].Cantidad != 1 {
		t.Fatalf("debe ir primero el de reproducción más reciente: %+v", h.Audios)
	}
	if h.Audios[1].AudioId != 1 || h.Audios[1].Cantidad != 2 || h.Audios[1].UltimaReproduccion != "2026-11-01T10:05:00-05:00" {
		t.Fatalf("segundo elemento incorrecto: %+v", h.Audios[1])
	}
}

func TestLineasAntiguasNoCuentanEnHistoriales(t *testing.T) {
	repo, _ := nuevoRepoConDatos(t)
	// La línea sin UsuarioId cuenta en el resumen (total 5) pero en ningún historial.
	suma := repo.ObtenerHistorial("u1").TotalReproducciones + repo.ObtenerHistorial("u2").TotalReproducciones
	if suma != 4 {
		t.Fatalf("u1+u2 esperado 4, obtenido %d", suma)
	}
	if h := repo.ObtenerHistorial(""); h.TotalReproducciones != 0 {
		t.Fatalf("usuario vacío no debe tener historial: %+v", h)
	}
}

func TestGuardarActualizaHistorialYPersiste(t *testing.T) {
	repo, ruta := nuevoRepoConDatos(t)

	_, err := repo.Guardar(modelo.Reproduccion{
		AudioId: 1, Titulo: "Bohemian Rhapsody", Tipo: "Música",
		Fecha: "2026-11-02T09:00:00-05:00", UsuarioId: "u1",
	})
	if err != nil {
		t.Fatal(err)
	}
	h := repo.ObtenerHistorial("u1")
	if h.TotalReproducciones != 4 || h.Audios[0].AudioId != 1 {
		t.Fatalf("u1 esperado 4 con audio 1 primero, obtenido %+v", h)
	}

	// Un repositorio nuevo sobre el mismo archivo ve lo mismo (reinicio).
	repo2, err := NuevoRepositorioReproducciones(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if h := repo2.ObtenerHistorial("u1"); h.TotalReproducciones != 4 {
		t.Fatalf("tras reiniciar, u1 esperado 4, obtenido %+v", h)
	}
}

func TestUsuarioSinReproducciones(t *testing.T) {
	repo, _ := nuevoRepoConDatos(t)
	h := repo.ObtenerHistorial("nadie")
	if h.TotalReproducciones != 0 || h.Audios == nil || len(h.Audios) != 0 {
		t.Fatalf("debe ser lista vacía no nula: %+v", h)
	}
}
