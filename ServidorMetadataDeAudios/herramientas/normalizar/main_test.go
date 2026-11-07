package main

import (
	"testing"

	"metadatos.local/servidor-metadata/modelo"
)

func catalogoDePrueba() *modelo.Catalogo {
	return &modelo.Catalogo{Audios: []modelo.Audio{
		{Id: 1, TipoId: 1, Titulo: "Bohemian Rhapsody", Metadatos: map[string]string{
			"Artista Principal": "Queen", "Álbum": "A Night at the Opera", "Género Musical": "Rock"}},
		{Id: 4, TipoId: 2, Titulo: "Episodio 2: gRPC vs REST", Metadatos: map[string]string{
			"Nombre del Podcast": "Sistemas al Aire", "Anfitrión": "Laura Muñoz"}},
		{Id: 9, TipoId: 1, Titulo: "Cancion", Metadatos: map[string]string{
			"Album": "X", "Artista Principal": "Y", "Año Lanzamiento": "2006", "Sello": "Z", "Género Musical": "Pop"}},
		{Id: 10, TipoId: 1, Titulo: "Incompleto", Metadatos: map[string]string{"Año": "2007"}},
	}}
}

func TestNormalizarEsIdempotenteYNoBorraAudios(t *testing.T) {
	c := catalogoDePrueba()
	cambios := normalizar(c)
	if len(cambios) != 4 { // 3 renombres en el audio 9 + título de episodio en el 4
		t.Fatalf("se esperaban 4 cambios, hubo %d: %v", len(cambios), cambios)
	}
	if len(c.Audios) != 4 {
		t.Fatalf("no debe borrar audios")
	}
	if c.Audios[1].Metadatos["Título del Episodio"] != "gRPC vs REST" {
		t.Fatalf("título del episodio incorrecto: %v", c.Audios[1].Metadatos)
	}
	m := c.Audios[2].Metadatos
	if m["Álbum"] != "X" || m["Año de Lanzamiento"] != "2006" || m["Sello Discográfico"] != "Z" || m["Album"] != "" {
		t.Fatalf("renombres incorrectos: %v", m)
	}
	if again := normalizar(c); len(again) != 0 {
		t.Fatalf("segunda pasada debe ser vacía: %v", again)
	}
}

func TestFaltantesNoInventaValores(t *testing.T) {
	c := catalogoDePrueba()
	normalizar(c)
	pend := faltantes(c)
	if len(pend) != 1 { // solo el audio 10
		t.Fatalf("debe quedar solo el audio 10 pendiente: %v", pend)
	}
	if len(c.Audios[3].Metadatos) != 1 {
		t.Fatalf("el audio 10 no debe modificarse: %v", c.Audios[3].Metadatos)
	}
}
