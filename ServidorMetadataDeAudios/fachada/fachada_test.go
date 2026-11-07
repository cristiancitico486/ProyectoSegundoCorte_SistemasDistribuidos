package fachada

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"metadatos.local/servidor-metadata/dto"
	"metadatos.local/servidor-metadata/repositorio"
)

const catalogoSemilla = `{"tipos":[{"id":1,"nombre":"Música"},{"id":2,"nombre":"Podcasts"},{"id":3,"nombre":"Audiolibros"},{"id":4,"nombre":"Ruido Blanco"}],"audios":[],"siguienteId":1}`

func nuevaFachada(t *testing.T) *FachadaMetadatos {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "catalogo.json")
	if err := os.WriteFile(ruta, []byte(catalogoSemilla), 0644); err != nil {
		t.Fatal(err)
	}
	return NuevaFachadaMetadatos(repositorio.NuevoRepositorioEnRuta(ruta))
}

func codigo(err error) string {
	var n *ErrorNegocio
	if errors.As(err, &n) {
		return n.Codigo
	}
	return ""
}

func TestRegistrarMusicaValida(t *testing.T) {
	f := nuevaFachada(t)
	creado, err := f.RegistrarAudio(dto.AudioCrearDTO{TipoId: 1, Titulo: "  Mi canción ", Metadatos: map[string]string{
		"Artista Principal": " Queen ", "Álbum": "X", "Género Musical": "Rock", "Extra libre": "ok", "Vacío": "  ",
	}})
	if err != nil || creado.Id != 1 {
		t.Fatalf("debía registrar: %v %+v", err, creado)
	}
	d, _ := f.ObtenerDetalleAudio(1)
	if d.Titulo != "Mi canción" || d.Metadatos["Artista Principal"] != "Queen" || d.Metadatos["Extra libre"] != "ok" {
		t.Fatalf("no recortó espacios o perdió extras: %+v", d)
	}
	if _, hayVacio := d.Metadatos["Vacío"]; hayVacio {
		t.Fatalf("los metadatos vacíos no deben guardarse: %+v", d.Metadatos)
	}
}

func TestRegistrarFaltanObligatorios(t *testing.T) {
	f := nuevaFachada(t)
	_, err := f.RegistrarAudio(dto.AudioCrearDTO{TipoId: 2, Titulo: "Ep", Metadatos: map[string]string{
		"Nombre del Podcast": "Sistemas al Aire", "Anfitrión": "   ",
	}})
	var n *ErrorNegocio
	if !errors.As(err, &n) || n.Codigo != "metadatos_invalidos" {
		t.Fatalf("esperaba metadatos_invalidos: %v", err)
	}
	if len(n.Faltantes) != 2 || n.Faltantes[0] != "Título del Episodio" || n.Faltantes[1] != "Anfitrión" {
		t.Fatalf("faltantes incorrectos (orden del enunciado): %v", n.Faltantes)
	}
	// No debe haberse creado nada.
	if _, errDet := f.ObtenerDetalleAudio(1); codigo(errDet) != "audio_no_encontrado" {
		t.Fatalf("no debe crear el audio si la validación falla")
	}
}

func TestRegistrarOtrosErrores(t *testing.T) {
	f := nuevaFachada(t)
	if _, err := f.RegistrarAudio(dto.AudioCrearDTO{TipoId: 99, Titulo: "x"}); codigo(err) != "tipo_invalido" {
		t.Fatalf("tipo inexistente: %v", err)
	}
	if _, err := f.RegistrarAudio(dto.AudioCrearDTO{TipoId: 1, Titulo: "   "}); codigo(err) != "titulo_obligatorio" {
		t.Fatalf("título vacío: %v", err)
	}
	largo := make([]byte, 600)
	for i := range largo {
		largo[i] = 'a'
	}
	_, err := f.RegistrarAudio(dto.AudioCrearDTO{TipoId: 4, Titulo: "x", Metadatos: map[string]string{"Tipo de Sonido": string(largo)}})
	if codigo(err) != "metadatos_invalidos" {
		t.Fatalf("valor demasiado largo: %v", err)
	}
}

func TestCadaTipoExigeSusTresCampos(t *testing.T) {
	f := nuevaFachada(t)
	esperado := map[int][]string{
		1: {"Artista Principal", "Álbum", "Género Musical"},
		2: {"Nombre del Podcast", "Título del Episodio", "Anfitrión"},
		3: {"Título del Libro", "Autor", "Narrador"},
		4: {"Tipo de Sonido", "Fuente del Audio", "Uso Sugerido"},
	}
	for tipoId, claves := range esperado {
		resp, err := f.ConsultarCampos(tipoId)
		if err != nil {
			t.Fatal(err)
		}
		var obligatorios []string
		for _, c := range resp.Campos {
			if c.Obligatorio {
				obligatorios = append(obligatorios, c.Clave)
			}
		}
		if len(obligatorios) != 3 {
			t.Fatalf("tipo %d: se esperaban 3 obligatorios, hay %v", tipoId, obligatorios)
		}
		for i := range claves {
			if obligatorios[i] != claves[i] {
				t.Fatalf("tipo %d: obligatorios %v, se esperaba %v", tipoId, obligatorios, claves)
			}
		}
		// Con exactamente los obligatorios debe registrarse.
		meta := map[string]string{}
		for _, c := range claves {
			meta[c] = "valor"
		}
		if _, err := f.RegistrarAudio(dto.AudioCrearDTO{TipoId: tipoId, Titulo: "ok", Metadatos: meta}); err != nil {
			t.Fatalf("tipo %d con sus 3 campos debía registrarse: %v", tipoId, err)
		}
	}
	if _, err := f.ConsultarCampos(99); codigo(err) != "tipo_no_encontrado" {
		t.Fatalf("tipo inexistente en campos: %v", err)
	}
}
