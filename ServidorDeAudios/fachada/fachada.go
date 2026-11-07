// Package fachada centraliza la lógica de negocio entre el controlador y el
// repositorio, siguiendo el mismo patrón de capas que los demás módulos.
package fachada

import (
	"errors"
	"fmt"

	"audios.local/servidor-audios/repositorio"
)

var ErrArchivoVacio = errors.New("el archivo recibido está vacío")
var ErrNoEsMp3 = errors.New("el archivo recibido no es un mp3 válido")
var ErrNoEncontrado = errors.New("no existe un mp3 guardado con ese id")

type FachadaAudios struct {
	repo *repositorio.RepositorioAudios
}

func NuevaFachadaAudios(repo *repositorio.RepositorioAudios) *FachadaAudios {
	return &FachadaAudios{repo: repo}
}

// GuardarAudio recibe el mp3 que sube el administrador y lo persiste con el nombre "{id}.mp3", el mismo id que ya generó el
// ServidorMetadataDeAudios al registrar los metadatos.
func (f *FachadaAudios) GuardarAudio(id int, data []byte) (string, error) {
	if len(data) == 0 {
		return "", ErrArchivoVacio
	}
	if !esMp3(data) {
		return "", ErrNoEsMp3
	}
	nombreMp3 := fmt.Sprintf("%d.mp3", id)
	if err := f.repo.GuardarMp3(nombreMp3, data); err != nil {
		return "", err
	}
	return nombreMp3, nil
}

// ObtenerRutaAudio resuelve la consulta que hace el ServidorDeStreaming
// (por REST) para conseguir el archivo antes de reenviarlo en chunks.
func (f *FachadaAudios) ObtenerRutaAudio(id int) (string, error) {
	nombreMp3 := fmt.Sprintf("%d.mp3", id)
	if !f.repo.Existe(nombreMp3) {
		return "", ErrNoEncontrado
	}
	return f.repo.RutaCompleta(nombreMp3), nil
}

// esMp3 comprueba de forma básica que los bytes correspondan a un mp3: o
// empiezan con una etiqueta ID3, o con la marca de sincronización de un
// frame MPEG (11 bits en uno).
func esMp3(data []byte) bool {
	if len(data) < 3 {
		return false
	}
	if data[0] == 'I' && data[1] == 'D' && data[2] == '3' {
		return true
	}
	return data[0] == 0xFF && data[1]&0xE0 == 0xE0
}
