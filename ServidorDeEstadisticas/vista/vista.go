// Package vista es la capa de presentación por consola del servidor de
// estadísticas (la V de MVC): imprime cada reproducción y los contadores.
package vista

import (
	"fmt"
	"sort"

	"estadisticas.local/servidor-estadisticas/modelo"
)

// MostrarReproduccion imprime una reproducción recién registrada.
func MostrarReproduccion(reproduccion modelo.Reproduccion) {
	fmt.Println()
	fmt.Println("Nueva reproducción registrada")
	fmt.Printf("  Audio: %s (id=%d)\n", reproduccion.Titulo, reproduccion.AudioId)
	fmt.Printf("  Tipo:  %s\n", reproduccion.Tipo)
	usuario := reproduccion.UsuarioId
	if usuario == "" {
		usuario = "(sin usuario)"
	}
	fmt.Printf("  Usuario: %s\n", usuario)
	fmt.Printf("  Fecha: %s\n", reproduccion.Fecha)
}

// MostrarResumen imprime los contadores acumulados, con las claves
// ordenadas para que la salida sea siempre igual.
func MostrarResumen(resumen modelo.Resumen) {
	fmt.Printf("  Total de reproducciones: %d\n", resumen.TotalReproducciones)
	fmt.Println("  Por tipo de audio:")
	imprimirContadores(resumen.PorTipo)
	fmt.Println("  Por audio:")
	imprimirContadores(resumen.PorTitulo)
}

func imprimirContadores(contadores map[string]int) {
	claves := make([]string, 0, len(contadores))
	for clave := range contadores {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	for _, clave := range claves {
		fmt.Printf("    - %s: %d\n", clave, contadores[clave])
	}
}
