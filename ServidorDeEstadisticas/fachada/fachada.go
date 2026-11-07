// Package fachada contiene la lógica de negocio del servidor de
// estadísticas: validar el mensaje recibido, convertirlo a modelo,
// pedirle al repositorio que lo almacene y responder las consultas.
package fachada

import (
	"errors"
	"strings"

	"estadisticas.local/servidor-estadisticas/dto"
	"estadisticas.local/servidor-estadisticas/modelo"
	"estadisticas.local/servidor-estadisticas/repositorio"
)

var ErrReproduccionInvalida = errors.New("la reproducción no trae título o tipo")
var ErrUsuarioInvalido = errors.New("el usuarioId es obligatorio")

type FachadaEstadisticas struct {
	repo *repositorio.RepositorioReproducciones
}

func NuevaFachadaEstadisticas(repo *repositorio.RepositorioReproducciones) *FachadaEstadisticas {
	return &FachadaEstadisticas{repo: repo}
}

// RegistrarReproduccion valida el DTO recibido de la cola, lo almacena y
// devuelve los contadores acumulados. Si el mensaje no trae usuario cuenta
// en el resumen global pero no en ningún historial.
func (f *FachadaEstadisticas) RegistrarReproduccion(entrada dto.ReproduccionDTO) (modelo.Reproduccion, modelo.Resumen, error) {
	if entrada.Titulo == "" || entrada.Tipo == "" {
		return modelo.Reproduccion{}, modelo.Resumen{}, ErrReproduccionInvalida
	}
	reproduccion := modelo.Reproduccion{
		AudioId:   entrada.AudioId,
		Titulo:    entrada.Titulo,
		Tipo:      entrada.Tipo,
		Fecha:     entrada.Fecha,
		UsuarioId: strings.TrimSpace(entrada.UsuarioId),
	}
	resumen, err := f.repo.Guardar(reproduccion)
	if err != nil {
		return modelo.Reproduccion{}, modelo.Resumen{}, err
	}
	return reproduccion, resumen, nil
}

// ConsultarHistorial responde "qué audios reprodujo este usuario y cuántas
// veces". Un usuario sin reproducciones devuelve la lista vacía.
func (f *FachadaEstadisticas) ConsultarHistorial(usuarioId string) (dto.HistorialDTO, error) {
	usuarioId = strings.TrimSpace(usuarioId)
	if usuarioId == "" {
		return dto.HistorialDTO{}, ErrUsuarioInvalido
	}
	historial := f.repo.ObtenerHistorial(usuarioId)

	respuesta := dto.HistorialDTO{
		UsuarioId:           historial.UsuarioId,
		TotalReproducciones: historial.TotalReproducciones,
		Audios:              make([]dto.ItemHistorialDTO, 0, len(historial.Audios)),
	}
	for _, item := range historial.Audios {
		respuesta.Audios = append(respuesta.Audios, dto.ItemHistorialDTO{
			AudioId:            item.AudioId,
			Titulo:             item.Titulo,
			Tipo:               item.Tipo,
			Cantidad:           item.Cantidad,
			UltimaReproduccion: item.UltimaReproduccion,
		})
	}
	return respuesta, nil
}
