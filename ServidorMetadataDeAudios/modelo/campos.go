package modelo

// CampoMetadato describe un metadato propio de un tipo de audio. Los
// obligatorios son los que pide el requerimiento del segundo corte; el
// administrador puede enviar además otros metadatos libres (opcionales).
type CampoMetadato struct {
	Clave       string
	Obligatorio bool
	// Sugerencias son valores de ejemplo para ofrecer en el formulario
	// (no se imponen: el valor es texto libre).
	Sugerencias []string
}

// camposPorTipo define los metadatos de cada tipo, indexados por el id del
// tipo en datos/catalogo.json (1 Música, 2 Podcasts, 3 Audiolibros,
// 4 Ruido Blanco). Los obligatorios van primero, en el orden del enunciado.
// Un tipo nuevo sin entrada aquí no exige ningún metadato.
var camposPorTipo = map[int][]CampoMetadato{
	1: {
		{Clave: "Artista Principal", Obligatorio: true},
		{Clave: "Álbum", Obligatorio: true},
		{Clave: "Género Musical", Obligatorio: true, Sugerencias: []string{"Pop", "Rock", "Jazz"}},
		{Clave: "Año de Lanzamiento"},
		{Clave: "Sello Discográfico"},
	},
	2: {
		{Clave: "Nombre del Podcast", Obligatorio: true},
		{Clave: "Título del Episodio", Obligatorio: true},
		{Clave: "Anfitrión", Obligatorio: true},
		{Clave: "Número de Temporada/Episodio"},
		{Clave: "Clasificación de Contenido"},
		{Clave: "Notas del Show"},
	},
	3: {
		{Clave: "Título del Libro", Obligatorio: true},
		{Clave: "Autor", Obligatorio: true},
		{Clave: "Narrador", Obligatorio: true},
		{Clave: "Capítulo"},
		{Clave: "Editorial"},
		{Clave: "ISBN"},
	},
	4: {
		{Clave: "Tipo de Sonido", Obligatorio: true, Sugerencias: []string{"Ruido Blanco", "Ruido Marrón", "Ruido Rosa"}},
		{Clave: "Fuente del Audio", Obligatorio: true, Sugerencias: []string{"Lluvia", "Ventilador", "Bosque"}},
		{Clave: "Uso Sugerido", Obligatorio: true, Sugerencias: []string{"Dormir", "Concentración", "Meditación"}},
		{Clave: "Duración del Bucle"},
		{Clave: "Frecuencia Dominante"},
		{Clave: "Proveedor de Contenido"},
	},
}

// CamposDeTipo devuelve una copia de los metadatos definidos para un tipo
// (obligatorios primero). Si el tipo no tiene definición devuelve nil.
func CamposDeTipo(tipoId int) []CampoMetadato {
	campos := camposPorTipo[tipoId]
	copia := make([]CampoMetadato, len(campos))
	for i, c := range campos {
		copia[i] = c
		copia[i].Sugerencias = append([]string(nil), c.Sugerencias...)
	}
	return copia
}

// ClavesObligatorias devuelve solo las claves obligatorias de un tipo.
func ClavesObligatorias(tipoId int) []string {
	var claves []string
	for _, c := range camposPorTipo[tipoId] {
		if c.Obligatorio {
			claves = append(claves, c.Clave)
		}
	}
	return claves
}
