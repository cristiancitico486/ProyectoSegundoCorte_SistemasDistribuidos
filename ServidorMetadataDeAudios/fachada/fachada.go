// Package fachada centraliza la lógica de negocio entre el controlador
// (HTTP) y el repositorio (acceso a datos), siguiendo el patrón de capas.
package fachada

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"metadatos.local/servidor-metadata/dto"
	"metadatos.local/servidor-metadata/modelo"
	"metadatos.local/servidor-metadata/repositorio"
)

// Límites de tamaño para no aceptar basura en el catálogo.
const (
	maxLongitudTitulo = 200
	maxLongitudClave  = 100
	maxLongitudValor  = 500
	maxMetadatos      = 30
)

// ErrorNegocio es un error esperado de la lógica de negocio. Codigo es el
// identificador estable que viaja al cliente en el campo "error"; Faltantes
// solo se usa en "metadatos_invalidos".
type ErrorNegocio struct {
	Codigo    string
	Mensaje   string
	Faltantes []string
}

func (e *ErrorNegocio) Error() string { return e.Mensaje }

var (
	ErrTipoNoEncontrado  = &ErrorNegocio{Codigo: "tipo_no_encontrado", Mensaje: "el tipo indicado no existe"}
	ErrTipoInvalido      = &ErrorNegocio{Codigo: "tipo_invalido", Mensaje: "el tipoId indicado no existe"}
	ErrAudioNoEncontrado = &ErrorNegocio{Codigo: "audio_no_encontrado", Mensaje: "el audio indicado no existe"}
	ErrTituloObligatorio = &ErrorNegocio{Codigo: "titulo_obligatorio", Mensaje: "el título es obligatorio"}
)

type FachadaMetadatos struct {
	repo *repositorio.RepositorioCatalogo
}

func NuevaFachadaMetadatos(repo *repositorio.RepositorioCatalogo) *FachadaMetadatos {
	return &FachadaMetadatos{repo: repo}
}

// ListarTipos responde a la consulta correspondiente.
func (f *FachadaMetadatos) ListarTipos() []dto.TipoAudioDTO {
	tipos := f.repo.ListarTipos()
	resultado := make([]dto.TipoAudioDTO, 0, len(tipos))
	for _, t := range tipos {
		resultado = append(resultado, dto.TipoAudioDTO{Id: t.Id, Nombre: t.Nombre})
	}
	return resultado
}

// ListarAudiosPorTipo responde a la consulta correspondiente.
func (f *FachadaMetadatos) ListarAudiosPorTipo(tipoId int) ([]dto.AudioResumenDTO, error) {
	if !f.repo.ExisteTipo(tipoId) {
		return nil, ErrTipoNoEncontrado
	}
	audios := f.repo.ListarAudiosPorTipo(tipoId)
	resultado := make([]dto.AudioResumenDTO, 0, len(audios))
	for _, a := range audios {
		resultado = append(resultado, dto.AudioResumenDTO{Id: a.Id, Titulo: a.Titulo})
	}
	return resultado, nil
}

// ObtenerDetalleAudio responde a la consulta correspondiente.
func (f *FachadaMetadatos) ObtenerDetalleAudio(id int) (dto.AudioDetalleDTO, error) {
	audio, ok := f.repo.ObtenerAudioPorId(id)
	if !ok {
		return dto.AudioDetalleDTO{}, ErrAudioNoEncontrado
	}
	return dto.AudioDetalleDTO{
		Id:        audio.Id,
		TipoId:    audio.TipoId,
		Titulo:    audio.Titulo,
		Metadatos: audio.Metadatos,
		NombreMp3: audio.NombreMp3,
	}, nil
}

// ConsultarCampos devuelve los metadatos de un tipo de audio (obligatorios
// primero): con esto el administrador web arma el formulario del tipo.
func (f *FachadaMetadatos) ConsultarCampos(tipoId int) (dto.CamposTipoDTO, error) {
	tipo, ok := f.repo.ObtenerTipo(tipoId)
	if !ok {
		return dto.CamposTipoDTO{}, ErrTipoNoEncontrado
	}
	campos := modelo.CamposDeTipo(tipoId)
	respuesta := dto.CamposTipoDTO{TipoId: tipo.Id, Tipo: tipo.Nombre, Campos: make([]dto.CampoDTO, 0, len(campos))}
	for _, c := range campos {
		respuesta.Campos = append(respuesta.Campos, dto.CampoDTO{
			Clave:       c.Clave,
			Obligatorio: c.Obligatorio,
			Sugerencias: c.Sugerencias,
		})
	}
	return respuesta, nil
}

// RegistrarAudio valida y registra los metadatos de un audio nuevo: crea el
// registro y devuelve el id que luego se usará para subir el mp3 al
// ServidorDeAudios. Los metadatos obligatorios dependen del tipo (ver
// modelo/campos.go); cualquier otro metadato se acepta como opcional.
func (f *FachadaMetadatos) RegistrarAudio(entrada dto.AudioCrearDTO) (dto.AudioCreadoDTO, error) {
	tipo, ok := f.repo.ObtenerTipo(entrada.TipoId)
	if !ok {
		return dto.AudioCreadoDTO{}, ErrTipoInvalido
	}

	titulo := strings.TrimSpace(entrada.Titulo)
	if titulo == "" {
		return dto.AudioCreadoDTO{}, ErrTituloObligatorio
	}
	if utf8.RuneCountInString(titulo) > maxLongitudTitulo {
		return dto.AudioCreadoDTO{}, &ErrorNegocio{
			Codigo:  "titulo_invalido",
			Mensaje: fmt.Sprintf("el título no puede superar %d caracteres", maxLongitudTitulo),
		}
	}

	metadatos, err := limpiarMetadatos(entrada.Metadatos)
	if err != nil {
		return dto.AudioCreadoDTO{}, err
	}

	var faltantes []string
	for _, clave := range modelo.ClavesObligatorias(entrada.TipoId) {
		if metadatos[clave] == "" {
			faltantes = append(faltantes, clave)
		}
	}
	if len(faltantes) > 0 {
		return dto.AudioCreadoDTO{}, &ErrorNegocio{
			Codigo:    "metadatos_invalidos",
			Mensaje:   fmt.Sprintf("faltan metadatos obligatorios para %s: %s", tipo.Nombre, strings.Join(faltantes, ", ")),
			Faltantes: faltantes,
		}
	}

	audio, err := f.repo.CrearAudio(entrada.TipoId, titulo, metadatos)
	if err != nil {
		return dto.AudioCreadoDTO{}, err
	}
	return dto.AudioCreadoDTO{Id: audio.Id}, nil
}

// limpiarMetadatos recorta espacios, descarta los metadatos con valor vacío
// y aplica los límites de tamaño.
func limpiarMetadatos(entrada map[string]string) (map[string]string, error) {
	limpios := make(map[string]string, len(entrada))
	for clave, valor := range entrada {
		clave = strings.TrimSpace(clave)
		valor = strings.TrimSpace(valor)
		if clave == "" || valor == "" {
			continue
		}
		if utf8.RuneCountInString(clave) > maxLongitudClave || utf8.RuneCountInString(valor) > maxLongitudValor {
			return nil, &ErrorNegocio{
				Codigo:  "metadatos_invalidos",
				Mensaje: fmt.Sprintf("cada clave admite hasta %d caracteres y cada valor hasta %d", maxLongitudClave, maxLongitudValor),
			}
		}
		limpios[clave] = valor
	}
	if len(limpios) > maxMetadatos {
		return nil, &ErrorNegocio{
			Codigo:  "metadatos_invalidos",
			Mensaje: fmt.Sprintf("se admiten hasta %d metadatos por audio", maxMetadatos),
		}
	}
	return limpios, nil
}
