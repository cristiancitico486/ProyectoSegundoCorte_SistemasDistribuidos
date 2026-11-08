// Package repositorio es la capa de acceso a datos del servidor de
// suscripciones: guarda cada pago aprobado en datos/pagos.jsonl (una línea
// JSON por pago) y lo mantiene indexado en memoria. Al arrancar relee el
// archivo, así los tokens ya usados no se olvidan al reiniciar.
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

	"suscripciones.local/servidor-suscripciones/modelo"
)

// ErrTokenDuplicado se devuelve si se intenta guardar un token ya guardado.
var ErrTokenDuplicado = errors.New("el token ya está almacenado")

type RepositorioPagos struct {
	mu       sync.Mutex
	ruta     string
	pagos    []modelo.Pago
	porToken map[string]int // token -> posición en pagos
}

// NuevoRepositorioPagos crea la carpeta del archivo si no existe y carga los
// pagos ya guardados.
func NuevoRepositorioPagos(ruta string) (*RepositorioPagos, error) {
	if err := os.MkdirAll(filepath.Dir(ruta), os.ModePerm); err != nil {
		return nil, fmt.Errorf("no se pudo crear la carpeta de datos: %w", err)
	}
	r := &RepositorioPagos{ruta: ruta, porToken: map[string]int{}}
	if err := r.cargar(); err != nil {
		return nil, err
	}
	return r, nil
}

// Ruta devuelve la ruta del archivo de pagos (para los ecos).
func (r *RepositorioPagos) Ruta() string { return r.ruta }

func (r *RepositorioPagos) cargar() error {
	archivo, err := os.Open(r.ruta)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("error abriendo %s: %w", r.ruta, err)
	}
	defer archivo.Close()

	lector := bufio.NewScanner(archivo)
	lector.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	cargados, omitidos := 0, 0
	for lector.Scan() {
		linea := lector.Bytes()
		if len(linea) == 0 {
			continue
		}
		var pago modelo.Pago
		if err := json.Unmarshal(linea, &pago); err != nil || pago.Token == "" {
			omitidos++
			fmt.Printf("[Datos] Línea inválida en %s, se omite\n", r.ruta)
			continue
		}
		if _, existe := r.porToken[pago.Token]; existe {
			omitidos++
			continue
		}
		r.porToken[pago.Token] = len(r.pagos)
		r.pagos = append(r.pagos, pago)
		cargados++
	}
	if err := lector.Err(); err != nil {
		return fmt.Errorf("error leyendo %s: %w", r.ruta, err)
	}
	fmt.Printf("[Datos] %d pagos cargados desde %s (%d líneas omitidas)\n", cargados, r.ruta, omitidos)
	return nil
}

// ObtenerPorToken busca un pago por su token de idempotencia.
func (r *RepositorioPagos) ObtenerPorToken(token string) (modelo.Pago, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	i, ok := r.porToken[token]
	if !ok {
		return modelo.Pago{}, false
	}
	return r.pagos[i], true
}

// Guardar agrega el pago al archivo (con sincronización a disco) y luego a
// la memoria. Si falla el guardado no queda nada en memoria: el token no se
// consume.
func (r *RepositorioPagos) Guardar(pago modelo.Pago) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, existe := r.porToken[pago.Token]; existe {
		return ErrTokenDuplicado
	}

	linea, err := json.Marshal(pago)
	if err != nil {
		return fmt.Errorf("error convirtiendo el pago a JSON: %w", err)
	}
	archivo, err := os.OpenFile(r.ruta, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error abriendo %s: %w", r.ruta, err)
	}
	defer archivo.Close()
	if _, err := archivo.Write(append(linea, '\n')); err != nil {
		return fmt.Errorf("error escribiendo en %s: %w", r.ruta, err)
	}
	if err := archivo.Sync(); err != nil {
		return fmt.Errorf("error sincronizando %s: %w", r.ruta, err)
	}

	r.porToken[pago.Token] = len(r.pagos)
	r.pagos = append(r.pagos, pago)
	return nil
}

// ListarPorUsuario devuelve los pagos de un usuario, el más reciente primero.
func (r *RepositorioPagos) ListarPorUsuario(usuarioId string) []modelo.Pago {
	r.mu.Lock()
	defer r.mu.Unlock()
	var lista []modelo.Pago
	for i := len(r.pagos) - 1; i >= 0; i-- { // recorrido inverso: desempata por orden de llegada
		if r.pagos[i].UsuarioId == usuarioId {
			lista = append(lista, r.pagos[i])
		}
	}
	sort.SliceStable(lista, func(a, b int) bool { return lista[a].Fecha.After(lista[b].Fecha) })
	return lista
}
