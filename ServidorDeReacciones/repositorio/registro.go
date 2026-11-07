// Package repositorio guarda en memoria quién está registrado en qué audio.
// No hay persistencia: las reacciones son efímeras.
package repositorio

import "sync"

// Cliente es una conexión WebSocket registrable. Enviar entrega un mensaje
// ya serializado por la conexión abierta del cliente.
type Cliente struct {
	UsuarioId string
	Enviar    func([]byte) error
}

type RepositorioRegistro struct {
	mu       sync.Mutex
	porAudio map[int]map[*Cliente]struct{}
	audioDe  map[*Cliente]int
}

func NuevoRepositorioRegistro() *RepositorioRegistro {
	return &RepositorioRegistro{
		porAudio: map[int]map[*Cliente]struct{}{},
		audioDe:  map[*Cliente]int{},
	}
}

// Registrar deja al cliente en el audio (si estaba en otro, lo saca de ahí)
// y devuelve cuántos oyentes tiene ahora ese audio.
func (r *RepositorioRegistro) Registrar(c *Cliente, audioId int) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.quitar(c)
	if r.porAudio[audioId] == nil {
		r.porAudio[audioId] = map[*Cliente]struct{}{}
	}
	r.porAudio[audioId][c] = struct{}{}
	r.audioDe[c] = audioId
	return len(r.porAudio[audioId])
}

// Desregistrar saca al cliente solo si estaba registrado en ese audio.
func (r *RepositorioRegistro) Desregistrar(c *Cliente, audioId int) (bool, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if actual, ok := r.audioDe[c]; !ok || actual != audioId {
		return false, len(r.porAudio[audioId])
	}
	r.quitar(c)
	return true, len(r.porAudio[audioId])
}

// Quitar se usa al desconectarse el cliente.
func (r *RepositorioRegistro) Quitar(c *Cliente) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.quitar(c)
}

func (r *RepositorioRegistro) quitar(c *Cliente) {
	audioId, ok := r.audioDe[c]
	if !ok {
		return
	}
	delete(r.audioDe, c)
	delete(r.porAudio[audioId], c)
	if len(r.porAudio[audioId]) == 0 {
		delete(r.porAudio, audioId)
	}
}

// AudioDe dice en qué audio está registrado el cliente.
func (r *RepositorioRegistro) AudioDe(c *Cliente) (int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	audioId, ok := r.audioDe[c]
	return audioId, ok
}

// Oyentes devuelve una copia de los clientes registrados en el audio.
func (r *RepositorioRegistro) Oyentes(audioId int) []*Cliente {
	r.mu.Lock()
	defer r.mu.Unlock()
	copia := make([]*Cliente, 0, len(r.porAudio[audioId]))
	for c := range r.porAudio[audioId] {
		copia = append(copia, c)
	}
	return copia
}
