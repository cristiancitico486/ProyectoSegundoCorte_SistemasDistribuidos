// Package repositorio es la capa de acceso a datos del servidor de
// estadísticas: guarda cada reproducción en un archivo de texto (una línea
// JSON por evento) y mantiene en memoria los contadores y el historial por
// usuario. Al arrancar reconstruye ambos leyendo el archivo, así los datos
// no se pierden al reiniciar el servidor.
package repositorio

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"estadisticas.local/servidor-estadisticas/modelo"
)

type RepositorioReproducciones struct {
	mu          sync.Mutex
	rutaArchivo string
	resumen     modelo.Resumen
	// porUsuario: usuarioId -> audioId -> acumulado de ese usuario.
	porUsuario map[string]map[int]*modelo.ItemHistorial
}

// NuevoRepositorioReproducciones crea la carpeta del archivo si no existe y
// reconstruye contadores e historial a partir de las líneas ya guardadas.
func NuevoRepositorioReproducciones(rutaArchivo string) (*RepositorioReproducciones, error) {
	if err := os.MkdirAll(filepath.Dir(rutaArchivo), os.ModePerm); err != nil {
		return nil, fmt.Errorf("no se pudo crear la carpeta de datos: %w", err)
	}
	r := &RepositorioReproducciones{
		rutaArchivo: rutaArchivo,
		resumen: modelo.Resumen{
			PorTipo:   map[string]int{},
			PorTitulo: map[string]int{},
		},
		porUsuario: map[string]map[int]*modelo.ItemHistorial{},
	}
	if err := r.cargarExistentes(); err != nil {
		return nil, err
	}
	return r, nil
}

// cargarExistentes lee el archivo línea por línea y acumula cada
// reproducción. Las líneas ilegibles se saltan con un aviso.
func (r *RepositorioReproducciones) cargarExistentes() error {
	archivo, err := os.Open(r.rutaArchivo)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error abriendo %s: %w", r.rutaArchivo, err)
	}
	defer archivo.Close()

	lector := bufio.NewScanner(archivo)
	lector.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	cargadas, saltadas := 0, 0
	for lector.Scan() {
		linea := lector.Bytes()
		if len(linea) == 0 {
			continue
		}
		var reproduccion modelo.Reproduccion
		if err := json.Unmarshal(linea, &reproduccion); err != nil {
			saltadas++
			fmt.Printf("[Datos] Línea inválida en %s, se omite: %v\n", r.rutaArchivo, err)
			continue
		}
		r.acumular(reproduccion)
		cargadas++
	}
	if err := lector.Err(); err != nil {
		return fmt.Errorf("error leyendo %s: %w", r.rutaArchivo, err)
	}
	fmt.Printf("[Datos] %d reproducciones cargadas desde %s (%d líneas omitidas)\n", cargadas, r.rutaArchivo, saltadas)
	return nil
}

// acumular actualiza contadores globales e historial del usuario. Debe
// llamarse con el mutex tomado (o antes de compartir el repositorio). Las
// reproducciones sin usuario (líneas antiguas) solo cuentan en el resumen.
func (r *RepositorioReproducciones) acumular(reproduccion modelo.Reproduccion) {
	r.resumen.TotalReproducciones++
	r.resumen.PorTipo[reproduccion.Tipo]++
	r.resumen.PorTitulo[reproduccion.Titulo]++

	if reproduccion.UsuarioId == "" {
		return
	}

	audios, ok := r.porUsuario[reproduccion.UsuarioId]
	if !ok {
		audios = map[int]*modelo.ItemHistorial{}
		r.porUsuario[reproduccion.UsuarioId] = audios
	}
	item, ok := audios[reproduccion.AudioId]
	if !ok {
		item = &modelo.ItemHistorial{
			AudioId: reproduccion.AudioId,
			Titulo:  reproduccion.Titulo,
			Tipo:    reproduccion.Tipo,
		}
		audios[reproduccion.AudioId] = item
	}
	item.Cantidad++
	// Se conserva la fecha más reciente (las líneas del archivo están en
	// orden de llegada, pero no se depende de eso).
	if item.UltimaReproduccion == "" || posterior(reproduccion.Fecha, item.UltimaReproduccion) {
		item.UltimaReproduccion = reproduccion.Fecha
	}
}

// Guardar agrega la reproducción al archivo y actualiza los contadores.
// Devuelve una copia del resumen acumulado.
func (r *RepositorioReproducciones) Guardar(reproduccion modelo.Reproduccion) (modelo.Resumen, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	linea, err := json.Marshal(reproduccion)
	if err != nil {
		return modelo.Resumen{}, fmt.Errorf("error convirtiendo la reproducción a JSON: %w", err)
	}

	archivo, err := os.OpenFile(r.rutaArchivo, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return modelo.Resumen{}, fmt.Errorf("error abriendo %s: %w", r.rutaArchivo, err)
	}
	defer archivo.Close()

	if _, err := archivo.Write(append(linea, '\n')); err != nil {
		return modelo.Resumen{}, fmt.Errorf("error escribiendo en %s: %w", r.rutaArchivo, err)
	}

	r.acumular(reproduccion)
	return r.copiarResumen(), nil
}

// ObtenerResumen devuelve una copia de los contadores acumulados.
func (r *RepositorioReproducciones) ObtenerResumen() modelo.Resumen {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.copiarResumen()
}

// ObtenerHistorial devuelve los audios que reprodujo un usuario con la
// cantidad de veces de cada uno, el de reproducción más reciente primero. Si
// el usuario no tiene reproducciones devuelve un historial vacío, no un error.
func (r *RepositorioReproducciones) ObtenerHistorial(usuarioId string) modelo.Historial {
	r.mu.Lock()
	defer r.mu.Unlock()

	historial := modelo.Historial{UsuarioId: usuarioId, Audios: []modelo.ItemHistorial{}}
	for _, item := range r.porUsuario[usuarioId] {
		historial.Audios = append(historial.Audios, *item)
		historial.TotalReproducciones += item.Cantidad
	}
	sort.Slice(historial.Audios, func(i, j int) bool {
		a, b := historial.Audios[i], historial.Audios[j]
		if a.UltimaReproduccion != b.UltimaReproduccion {
			return posterior(a.UltimaReproduccion, b.UltimaReproduccion)
		}
		return a.Titulo < b.Titulo
	})
	return historial
}

// posterior indica si la fecha a es posterior a la fecha b (RFC 3339). Si
// alguna no se puede interpretar, compara como texto.
func posterior(a, b string) bool {
	ta, errA := time.Parse(time.RFC3339, a)
	tb, errB := time.Parse(time.RFC3339, b)
	if errA != nil || errB != nil {
		return a > b
	}
	return ta.After(tb)
}

func (r *RepositorioReproducciones) copiarResumen() modelo.Resumen {
	copia := modelo.Resumen{
		TotalReproducciones: r.resumen.TotalReproducciones,
		PorTipo:             make(map[string]int, len(r.resumen.PorTipo)),
		PorTitulo:           make(map[string]int, len(r.resumen.PorTitulo)),
	}
	for clave, valor := range r.resumen.PorTipo {
		copia.PorTipo[clave] = valor
	}
	for clave, valor := range r.resumen.PorTitulo {
		copia.PorTitulo[clave] = valor
	}
	return copia
}
