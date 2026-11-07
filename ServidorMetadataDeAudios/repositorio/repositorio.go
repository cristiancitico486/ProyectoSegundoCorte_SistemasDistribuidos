// Package repositorio es la capa de acceso a datos: lee y escribe el
// catálogo (tipos + audios) en un archivo JSON, protegido con un mutex
// porque el servidor HTTP atiende peticiones concurrentes.
package repositorio

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"metadatos.local/servidor-metadata/modelo"
)

type RepositorioCatalogo struct {
	mu       sync.Mutex
	rutaJSON string
	catalogo modelo.Catalogo
}

var (
	instancia *RepositorioCatalogo
	once      sync.Once
)

// ObtenerRepositorio aplica el patrón Singleton: una sola instancia del
// repositorio para todo el servidor.
func ObtenerRepositorio(rutaJSON string) *RepositorioCatalogo {
	once.Do(func() {
		instancia = NuevoRepositorioEnRuta(rutaJSON)
	})
	return instancia
}

// NuevoRepositorioEnRuta crea un repositorio independiente sobre el archivo
// indicado (lo usan las pruebas; el servidor usa ObtenerRepositorio).
func NuevoRepositorioEnRuta(rutaJSON string) *RepositorioCatalogo {
	r := &RepositorioCatalogo{rutaJSON: rutaJSON}
	r.cargar()
	return r
}

// cargar lee el catalogo.json de disco. Si no existe, arranca con un
// catálogo vacío (no debería pasar si se entrega el seed junto al código).
func (r *RepositorioCatalogo) cargar() {
	data, err := os.ReadFile(r.rutaJSON)
	if err != nil {
		fmt.Println("No se encontró catalogo.json, se inicia vacío:", err)
		r.catalogo = modelo.Catalogo{SiguienteId: 1}
		return
	}
	if err := json.Unmarshal(data, &r.catalogo); err != nil {
		fmt.Println("Error leyendo catalogo.json, se inicia vacío:", err)
		r.catalogo = modelo.Catalogo{SiguienteId: 1}
	}
}

// guardar persiste el catálogo completo en disco. Debe llamarse siempre con
// el mutex tomado.
func (r *RepositorioCatalogo) guardar() error {
	data, err := json.MarshalIndent(r.catalogo, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.rutaJSON, data, 0644)
}

// ListarTipos devuelve una copia de los tipos de audio.
func (r *RepositorioCatalogo) ListarTipos() []modelo.TipoAudio {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]modelo.TipoAudio{}, r.catalogo.Tipos...)
}

// ObtenerTipo busca un tipo por id. El segundo valor indica si se encontró.
func (r *RepositorioCatalogo) ObtenerTipo(tipoId int) (modelo.TipoAudio, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.catalogo.Tipos {
		if t.Id == tipoId {
			return t, true
		}
	}
	return modelo.TipoAudio{}, false
}

// ExisteTipo confirma si un tipoId es válido.
func (r *RepositorioCatalogo) ExisteTipo(tipoId int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.catalogo.Tipos {
		if t.Id == tipoId {
			return true
		}
	}
	return false
}

// ListarAudiosPorTipo devuelve todos los audios de un tipo dado.
func (r *RepositorioCatalogo) ListarAudiosPorTipo(tipoId int) []modelo.Audio {
	r.mu.Lock()
	defer r.mu.Unlock()
	var resultado []modelo.Audio
	for _, a := range r.catalogo.Audios {
		if a.TipoId == tipoId {
			resultado = append(resultado, a)
		}
	}
	return resultado
}

// ObtenerAudioPorId busca un audio por su id. El segundo valor indica si se encontró.
func (r *RepositorioCatalogo) ObtenerAudioPorId(id int) (modelo.Audio, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.catalogo.Audios {
		if a.Id == id {
			return a, true
		}
	}
	return modelo.Audio{}, false
}

// CrearAudio asigna un id nuevo, agrega el audio al catálogo y persiste en disco.
func (r *RepositorioCatalogo) CrearAudio(tipoId int, titulo string, metadatos map[string]string) (modelo.Audio, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	nuevo := modelo.Audio{
		Id:        r.catalogo.SiguienteId,
		TipoId:    tipoId,
		Titulo:    titulo,
		Metadatos: metadatos,
		NombreMp3: fmt.Sprintf("%d.mp3", r.catalogo.SiguienteId),
	}
	r.catalogo.Audios = append(r.catalogo.Audios, nuevo)
	r.catalogo.SiguienteId++

	if err := r.guardar(); err != nil {
		return modelo.Audio{}, fmt.Errorf("error guardando catalogo.json: %w", err)
	}
	return nuevo, nil
}
