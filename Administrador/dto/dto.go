// Package dto contiene los structs que el administrador envía y recibe por
// REST. Deben coincidir con los DTO expuestos por ServidorMetadataDeAudios
// y ServidorDeAudios.
package dto

// TipoAudioDTO llega en la respuesta de GET /tipos (ServidorMetadataDeAudios).
type TipoAudioDTO struct {
	Id     int    `json:"id"`
	Nombre string `json:"nombre"`
}

// AudioCrearDTO es el body que se envía a POST /audios en el
// ServidorMetadataDeAudios para registrar los metadatos de un audio nuevo.
type AudioCrearDTO struct {
	TipoId    int               `json:"tipoId"`
	Titulo    string            `json:"titulo"`
	Metadatos map[string]string `json:"metadatos"`
}

// AudioCreadoDTO es la respuesta de POST /audios: el id generado, que luego
// se usa para subir el mp3 al ServidorDeAudios.
type AudioCreadoDTO struct {
	Id int `json:"id"`
}

// SubidaOkDTO es la respuesta de POST /audios/{id} en el ServidorDeAudios.
type SubidaOkDTO struct {
	Mensaje   string `json:"mensaje"`
	NombreMp3 string `json:"nombreMp3"`
}

// ErrorDTO es la respuesta estándar de error de ambos servidores.
type ErrorDTO struct {
	Error string `json:"error"`
}
