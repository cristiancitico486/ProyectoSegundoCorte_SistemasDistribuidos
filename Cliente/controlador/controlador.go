// Package controlador orquesta la navegación entre pantallas (la C de MVC).
// No sabe nada de REST ni de streaming: solo llama a la fachada y decide,
// según la opción elegida, qué pantalla de la vista mostrar después.
package controlador

import (
	"fmt"

	"cliente.local/cliente-audio/dto"
	"cliente.local/cliente-audio/fachada"
	"cliente.local/cliente-audio/vista"
)

// Iniciar arranca el menú principal y no retorna hasta que el usuario
// elige salir.
func Iniciar(f *fachada.FachadaCliente) {
	for {
		vista.MostrarMenuPrincipal()
		switch vista.LeerOpcion() {
		case 1:
			menuTipos(f)
		case 2:
			fmt.Println("Hasta pronto.")
			return
		default:
			vista.MostrarOpcionInvalida()
		}
	}
}

// menuTipos muestra los tipos de audio. Devuelve true si el usuario reprodujo un audio y
// por lo tanto hay que volver directo al menú principal, en vez de
// quedarse navegando en esta pantalla.
func menuTipos(f *fachada.FachadaCliente) bool {
	tipos, err := f.ObtenerTipos()
	if err != nil {
		vista.MostrarError(err.Error())
		return false
	}

	for {
		vista.MostrarTipos(tipos)
		opcion := vista.LeerOpcion()
		volver := len(tipos) + 1

		switch {
		case opcion == volver:
			return false
		case opcion >= 1 && opcion <= len(tipos):
			if menuAudios(f, tipos[opcion-1]) {
				return true
			}
		default:
			vista.MostrarOpcionInvalida()
		}
	}
}

// menuAudios muestra los audios de un tipo. Mismo contrato de retorno que menuTipos. Recibe
// el TipoAudioDTO completo (no solo el id) para poder pasarle su nombre a
// la reproducción, que lo necesita para el mensaje que se publica en la
// cola de estadísticas.
func menuAudios(f *fachada.FachadaCliente, tipo dto.TipoAudioDTO) bool {
	audios, err := f.ObtenerAudiosPorTipo(tipo.Id)
	if err != nil {
		vista.MostrarError(err.Error())
		return false
	}

	for {
		vista.MostrarAudios(tipo, audios)
		opcion := vista.LeerOpcion()
		volver := len(audios) + 1

		switch {
		case opcion == volver:
			return false
		case opcion >= 1 && opcion <= len(audios):
			if menuDetalle(f, audios[opcion-1].Id, tipo.Nombre) {
				return true
			}
		default:
			vista.MostrarOpcionInvalida()
		}
	}
}

// menuDetalle muestra el detalle de un audio y permite reproducirlo. Si el usuario
// reproduce y luego sale, se propaga true para que las pantallas anteriores
// también se cierren y el flujo vuelva directo al menú principal.
func menuDetalle(f *fachada.FachadaCliente, audioId int, tipoNombre string) bool {
	detalle, err := f.ObtenerDetalleAudio(audioId)
	if err != nil {
		vista.MostrarError(err.Error())
		return false
	}

	for {
		vista.MostrarDetalle(detalle)
		switch vista.LeerOpcion() {
		case 1:
			if err := f.Reproducir(detalle.Id, detalle.Titulo, tipoNombre); err != nil {
				vista.MostrarError(err.Error())
				continue
			}
			return true
		case 2:
			return false
		default:
			vista.MostrarOpcionInvalida()
		}
	}
}
