// Package fachada contiene la lógica de negocio de las reacciones: quién
// puede reaccionar y a quién se le reenvía.
package fachada

import (
	"errors"
	"time"

	"reacciones.local/servidor-reacciones/dto"
	"reacciones.local/servidor-reacciones/repositorio"
)

var (
	ErrNoRegistrado     = errors.New("primero debes registrarte en el audio (registrar)")
	ErrReaccionInvalida = errors.New("reacción no válida")
)

var reaccionesValidas = map[string]bool{
	"me_gusta": true, "corazon": true, "aplauso": true, "fuego": true, "risa": true,
}

type FachadaReacciones struct {
	repo *repositorio.RepositorioRegistro
}

func NuevaFachadaReacciones(repo *repositorio.RepositorioRegistro) *FachadaReacciones {
	return &FachadaReacciones{repo: repo}
}

func (f *FachadaReacciones) Registrar(c *repositorio.Cliente, audioId int) dto.Ack {
	oyentes := f.repo.Registrar(c, audioId)
	return dto.Ack{Tipo: "ack", Operacion: "registrar", AudioId: audioId, Oyentes: oyentes}
}

// Desregistrar es idempotente: si no estaba registrado responde igual.
func (f *FachadaReacciones) Desregistrar(c *repositorio.Cliente, audioId int) dto.Ack {
	_, oyentes := f.repo.Desregistrar(c, audioId)
	return dto.Ack{Tipo: "ack", Operacion: "desregistrar", AudioId: audioId, Oyentes: oyentes}
}

// Reaccionar valida la reacción y devuelve el mensaje junto con todos los
// destinatarios (los registrados en ese audio, incluido el emisor).
func (f *FachadaReacciones) Reaccionar(c *repositorio.Cliente, audioId int, reaccion string) (dto.ReaccionSalida, []*repositorio.Cliente, error) {
	if !reaccionesValidas[reaccion] {
		return dto.ReaccionSalida{}, nil, ErrReaccionInvalida
	}
	if actual, ok := f.repo.AudioDe(c); !ok || actual != audioId {
		return dto.ReaccionSalida{}, nil, ErrNoRegistrado
	}
	msg := dto.ReaccionSalida{
		Tipo: "reaccion", AudioId: audioId, UsuarioId: c.UsuarioId,
		Reaccion: reaccion, Timestamp: time.Now().Format(time.RFC3339),
	}
	return msg, f.repo.Oyentes(audioId), nil
}

func (f *FachadaReacciones) Desconectar(c *repositorio.Cliente) { f.repo.Quitar(c) }
