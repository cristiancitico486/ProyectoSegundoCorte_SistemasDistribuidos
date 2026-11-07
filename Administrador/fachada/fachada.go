// Package fachada centraliza la orquestación entre los dos clientes REST
// (metadatos y audios), siguiendo el mismo patrón de capas del resto del
// proyecto.
package fachada

import (
	"fmt"

	"administrador.local/administrador-cliente/clientes"
	"administrador.local/administrador-cliente/dto"
)

type FachadaAdministrador struct {
	metadatos *clientes.MetadatosClient
	audios    *clientes.AudiosClient
}

func NuevaFachadaAdministrador(metadatos *clientes.MetadatosClient, audios *clientes.AudiosClient) *FachadaAdministrador {
	return &FachadaAdministrador{metadatos: metadatos, audios: audios}
}

// ObtenerTipos lista los tipos de audio disponibles, para que el
// administrador elija a cuál pertenece el audio nuevo.
func (f *FachadaAdministrador) ObtenerTipos() ([]dto.TipoAudioDTO, error) {
	return f.metadatos.ListarTipos()
}

// RegistrarYSubirAudio sube un audio de punta a punta:
//  1. Registra título, tipo y metadatos en el ServidorMetadataDeAudios,
//     que devuelve un id nuevo.
//  2. Sube el mp3 local al ServidorDeAudios usando ese mismo id.
//
// Si el paso 2 falla, el registro de metadatos ya quedó creado (sin
// archivo asociado); se informa igual al administrador para que pueda
// reintentar la subida con el id ya conocido.
func (f *FachadaAdministrador) RegistrarYSubirAudio(tipoId int, titulo string, metadatos map[string]string, rutaMp3 string) (int, error) {
	creado, err := f.metadatos.RegistrarAudio(dto.AudioCrearDTO{
		TipoId:    tipoId,
		Titulo:    titulo,
		Metadatos: metadatos,
	})
	if err != nil {
		return 0, fmt.Errorf("no se pudo registrar los metadatos: %w", err)
	}

	if _, err := f.audios.SubirMp3(creado.Id, rutaMp3); err != nil {
		return creado.Id, fmt.Errorf(
			"los metadatos quedaron registrados con id=%d, pero falló la subida del mp3: %w",
			creado.Id, err,
		)
	}
	return creado.Id, nil
}
