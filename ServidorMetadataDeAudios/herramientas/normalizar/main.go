// Herramienta de una sola vez: ajusta datos/catalogo.json a los metadatos
// obligatorios del segundo corte (ver modelo/campos.go) SIN borrar nada.
//
//	go run ./herramientas/normalizar              # solo muestra qué haría
//	go run ./herramientas/normalizar -aplicar     # guarda copia .bak y aplica
//
// Qué hace:
//   - Música: renombra claves antiguas a las del enunciado cuando el destino
//     no existe ("Album" -> "Álbum", "Año Lanzamiento" -> "Año de
//     Lanzamiento", "Sello" -> "Sello Discográfico").
//   - Podcasts: si falta "Título del Episodio", lo toma del título del audio
//     quitando el prefijo "Episodio N: " (ej. "Episodio 2: gRPC vs REST" ->
//     "gRPC vs REST").
//   - Lista los audios que siguen sin algún metadato obligatorio. No inventa
//     valores ni borra audios: esos casos los completa o decide una persona.
//
// IMPORTANTE: detén el ServidorMetadataDeAudios antes de aplicar y reinícialo
// después; si no, su copia en memoria sobrescribiría el archivo.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"metadatos.local/servidor-metadata/modelo"
)

// renombresMusica: clave antigua -> clave del enunciado (tipo 1, Música).
var renombresMusica = map[string]string{
	"Album":           "Álbum",
	"Año Lanzamiento": "Año de Lanzamiento",
	"Sello":           "Sello Discográfico",
}

var prefijoEpisodio = regexp.MustCompile(`^Episodio\s+\d+\s*:\s*(.+)$`)

func main() {
	ruta := flag.String("catalogo", "datos/catalogo.json", "ruta del catalogo.json")
	aplicar := flag.Bool("aplicar", false, "guardar los cambios (hace copia .bak antes)")
	flag.Parse()

	datos, err := os.ReadFile(*ruta)
	if err != nil {
		fmt.Println("No se pudo leer el catálogo:", err)
		os.Exit(1)
	}
	var catalogo modelo.Catalogo
	if err := json.Unmarshal(datos, &catalogo); err != nil {
		fmt.Println("El catálogo no es un JSON válido:", err)
		os.Exit(1)
	}

	cambios := normalizar(&catalogo)
	if len(cambios) == 0 {
		fmt.Println("Cambios: ninguno (el catálogo ya está normalizado).")
	} else {
		fmt.Printf("Cambios (%d):\n", len(cambios))
		for _, c := range cambios {
			fmt.Println("  -", c)
		}
	}

	if pendientes := faltantes(&catalogo); len(pendientes) > 0 {
		fmt.Println("\nAudios que siguen sin metadatos obligatorios (hay que completarlos o decidir si se borran):")
		for _, p := range pendientes {
			fmt.Println("  -", p)
		}
	} else {
		fmt.Println("\nTodos los audios tienen sus metadatos obligatorios.")
	}

	if !*aplicar {
		fmt.Println("\n(Simulación: no se guardó nada. Usa -aplicar para guardar.)")
		return
	}
	if len(cambios) == 0 {
		return
	}
	copia := fmt.Sprintf("%s.bak-%s", *ruta, time.Now().Format("20060102-150405"))
	if err := os.WriteFile(copia, datos, 0644); err != nil {
		fmt.Println("No se pudo crear la copia de seguridad:", err)
		os.Exit(1)
	}
	salida, err := json.MarshalIndent(catalogo, "", "  ")
	if err != nil {
		fmt.Println("Error serializando el catálogo:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*ruta, salida, 0644); err != nil {
		fmt.Println("Error guardando el catálogo:", err)
		os.Exit(1)
	}
	fmt.Printf("\nGuardado. Copia de seguridad en %s\n", copia)
}

// normalizar aplica los ajustes automáticos y devuelve la lista de cambios.
// Es idempotente: una segunda ejecución no cambia nada.
func normalizar(c *modelo.Catalogo) []string {
	var cambios []string
	for i := range c.Audios {
		a := &c.Audios[i]
		if a.Metadatos == nil {
			continue
		}
		if a.TipoId == 1 {
			for antigua, nueva := range renombresMusica {
				valor, existe := a.Metadatos[antigua]
				if !existe {
					continue
				}
				if _, yaTiene := a.Metadatos[nueva]; yaTiene {
					continue
				}
				a.Metadatos[nueva] = valor
				delete(a.Metadatos, antigua)
				cambios = append(cambios, fmt.Sprintf("audio %d %q: clave %q -> %q", a.Id, a.Titulo, antigua, nueva))
			}
		}
		if a.TipoId == 2 && strings.TrimSpace(a.Metadatos["Título del Episodio"]) == "" {
			if m := prefijoEpisodio.FindStringSubmatch(a.Titulo); m != nil {
				a.Metadatos["Título del Episodio"] = strings.TrimSpace(m[1])
				cambios = append(cambios, fmt.Sprintf("audio %d %q: se agrega \"Título del Episodio\" = %q", a.Id, a.Titulo, strings.TrimSpace(m[1])))
			}
		}
	}
	return cambios
}

// faltantes lista, por audio, los metadatos obligatorios que no tiene.
func faltantes(c *modelo.Catalogo) []string {
	var lista []string
	for _, a := range c.Audios {
		var sin []string
		for _, clave := range modelo.ClavesObligatorias(a.TipoId) {
			if strings.TrimSpace(a.Metadatos[clave]) == "" {
				sin = append(sin, clave)
			}
		}
		if len(sin) > 0 {
			lista = append(lista, fmt.Sprintf("audio %d %q (tipo %d): falta %s", a.Id, a.Titulo, a.TipoId, strings.Join(sin, ", ")))
		}
	}
	return lista
}
