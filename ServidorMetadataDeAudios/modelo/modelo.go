// Package modelo contiene las entidades del dominio (la M de MVC).
package modelo

// TipoAudio representa una categoría de audio (Música, Podcasts, Audiolibros, Ruido Blanco).
type TipoAudio struct {
	Id     int    `json:"id"`
	Nombre string `json:"nombre"`
}

// Audio representa un audio concreto perteneciente a un TipoAudio.
// Metadatos es libre por tipo: cada tipo de audio define sus propias
// claves (Artista, Álbum, Narrador, ISBN, etc.) sin necesidad de un struct
// distinto por tipo.
type Audio struct {
	Id        int               `json:"id"`
	TipoId    int               `json:"tipoId"`
	Titulo    string            `json:"titulo"`
	Metadatos map[string]string `json:"metadatos"`
	NombreMp3 string            `json:"nombreMp3"` // nombre del archivo en ServidorDeAudios, ej: "3.mp3"
}

// Catalogo es la raíz del documento JSON persistido en disco.
type Catalogo struct {
	Tipos       []TipoAudio `json:"tipos"`
	Audios      []Audio     `json:"audios"`
	SiguienteId int         `json:"siguienteId"`
}
