// Package dto agrupa los structs que viajan serializados como JSON por REST.
package dto

// TipoAudioDTO se usa para listar los tipos.
type TipoAudioDTO struct {
	Id     int    `json:"id"`
	Nombre string `json:"nombre"`
}

// AudioResumenDTO se usa para listar los audios de un tipo: solo lo
// necesario para mostrar el menú, sin cargar todos los metadatos.
type AudioResumenDTO struct {
	Id     int    `json:"id"`
	Titulo string `json:"titulo"`
}

// AudioDetalleDTO se usa para el detalle de un audio.
type AudioDetalleDTO struct {
	Id        int               `json:"id"`
	TipoId    int               `json:"tipoId"`
	Titulo    string            `json:"titulo"`
	Metadatos map[string]string `json:"metadatos"`
	NombreMp3 string            `json:"nombreMp3"`
}

// AudioCrearDTO es el body que envía el administrador para registrar los
// metadatos de un audio nuevo, antes de subir el mp3.
type AudioCrearDTO struct {
	TipoId    int               `json:"tipoId"`
	Titulo    string            `json:"titulo"`
	Metadatos map[string]string `json:"metadatos"`
}

// AudioCreadoDTO es la respuesta al registrar un audio: el id generado,
// que el cliente usará para subir el mp3 al ServidorDeAudios.
type AudioCreadoDTO struct {
	Id int `json:"id"`
}

// CampoDTO describe un metadato de un tipo de audio.
type CampoDTO struct {
	Clave       string   `json:"clave"`
	Obligatorio bool     `json:"obligatorio"`
	Sugerencias []string `json:"sugerencias,omitempty"`
}

// CamposTipoDTO es la respuesta de GET /tipos/{id}/campos: los metadatos
// que el administrador debe (o puede) enviar al registrar un audio del tipo.
type CamposTipoDTO struct {
	TipoId int        `json:"tipoId"`
	Tipo   string     `json:"tipo"`
	Campos []CampoDTO `json:"campos"`
}

// ErrorDTO es la respuesta estándar de error de los servicios REST:
// un código estable para que el cliente decida qué hacer y un mensaje
// legible. Faltantes solo se llena en "metadatos_invalidos".
type ErrorDTO struct {
	Error     string   `json:"error"`
	Mensaje   string   `json:"mensaje"`
	Faltantes []string `json:"faltantes,omitempty"`
}

// SaludDTO es la respuesta de GET /health (health check para Consul).
type SaludDTO struct {
	Status string `json:"status"`
}
