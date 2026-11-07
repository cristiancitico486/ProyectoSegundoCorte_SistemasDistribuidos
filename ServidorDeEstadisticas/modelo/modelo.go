// Package modelo contiene las entidades del dominio del servidor de
// estadísticas (la M de MVC).
package modelo

// Reproduccion representa un evento de reproducción ya validado y listo
// para almacenarse. No lleva tags JSON a propósito: el archivo
// datos/reproducciones.jsonl usa los nombres de campo de Go ("AudioId",
// "Titulo"...) y así las líneas ya guardadas se siguen leyendo (encoding/json
// no distingue mayúsculas al decodificar). Las líneas antiguas no traen
// UsuarioId: cuentan en el resumen global pero no en historiales por usuario.
type Reproduccion struct {
	AudioId   int
	Titulo    string
	Tipo      string
	Fecha     string
	UsuarioId string
}

// Resumen agrupa los contadores acumulados de todas las reproducciones
// almacenadas (se reconstruyen desde el archivo al arrancar).
type Resumen struct {
	TotalReproducciones int
	PorTipo             map[string]int
	PorTitulo           map[string]int
}

// ItemHistorial es la cantidad de veces que un usuario reprodujo un audio.
type ItemHistorial struct {
	AudioId            int
	Titulo             string
	Tipo               string
	Cantidad           int
	UltimaReproduccion string
}

// Historial agrupa los audios reproducidos por un usuario.
type Historial struct {
	UsuarioId           string
	TotalReproducciones int
	Audios              []ItemHistorial
}
