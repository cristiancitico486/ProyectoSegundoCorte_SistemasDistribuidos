// Package fachada centraliza la orquestación entre el controlador y las
// fuentes externas (el cliente REST de metadatos y el reproductor),
// siguiendo el mismo patrón de capas usado en el ServidorMetadataDeAudios.
package fachada

import (
	"cliente.local/cliente-audio/clientes"
	"cliente.local/cliente-audio/dto"
	"cliente.local/cliente-audio/reproductor"
)

type FachadaCliente struct {
	metadatos   *clientes.MetadatosClient
	reproductor reproductor.Reproductor
}

func NuevaFachadaCliente(metadatos *clientes.MetadatosClient, rep reproductor.Reproductor) *FachadaCliente {
	return &FachadaCliente{metadatos: metadatos, reproductor: rep}
}

// ObtenerTipos devuelve los tipos de audio registrados.
func (f *FachadaCliente) ObtenerTipos() ([]dto.TipoAudioDTO, error) {
	return f.metadatos.ListarTipos()
}

// ObtenerAudiosPorTipo devuelve los audios de un tipo.
func (f *FachadaCliente) ObtenerAudiosPorTipo(tipoId int) ([]dto.AudioResumenDTO, error) {
	return f.metadatos.ListarAudiosPorTipo(tipoId)
}

// ObtenerDetalleAudio devuelve los metadatos de un audio.
func (f *FachadaCliente) ObtenerDetalleAudio(audioId int) (dto.AudioDetalleDTO, error) {
	return f.metadatos.ObtenerDetalleAudio(audioId)
}

// Reproducir reproduce el audio por streaming, delegando en la implementación de
// Reproductor que se le haya inyectado. tipoNombre viaja hasta el
// ServidorDeStreaming para armar el mensaje que se publica en la cola de
// estadísticas.
func (f *FachadaCliente) Reproducir(audioId int, titulo string, tipoNombre string) error {
	return f.reproductor.Reproducir(audioId, titulo, tipoNombre)
}
