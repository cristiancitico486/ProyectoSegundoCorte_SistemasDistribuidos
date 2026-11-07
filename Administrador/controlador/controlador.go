// Package controlador orquesta la navegación del administrador. No sabe
// nada de REST: solo llama a la fachada y decide qué pantalla mostrar
// después, según la opción elegida.
package controlador

import (
	"fmt"

	"administrador.local/administrador-cliente/fachada"
	"administrador.local/administrador-cliente/vista"
)

// Iniciar arranca el menú principal y no retorna hasta que el
// administrador elige salir.
func Iniciar(f *fachada.FachadaAdministrador) {
	for {
		vista.MostrarMenuPrincipal()
		switch vista.LeerOpcion() {
		case 1:
			subirAudio(f)
		case 2:
			fmt.Println("Hasta pronto.")
			return
		default:
			vista.MostrarOpcionInvalida()
		}
	}
}

// subirAudio implementa el flujo completo de subida: elegir tipo, capturar
// título y metadatos, confirmar, registrar en el servidor de metadatos y
// subir el mp3 al servidor de audios.
func subirAudio(f *fachada.FachadaAdministrador) {
	tipos, err := f.ObtenerTipos()
	if err != nil {
		vista.MostrarError(err.Error())
		return
	}
	if len(tipos) == 0 {
		vista.MostrarError("el servidor de metadatos no tiene tipos registrados")
		return
	}

	vista.MostrarTipos(tipos)
	opcion := vista.LeerOpcion()
	if opcion < 1 || opcion > len(tipos) {
		vista.MostrarOpcionInvalida()
		return
	}
	tipo := tipos[opcion-1]

	titulo := vista.PedirTitulo()
	if titulo == "" {
		vista.MostrarError("el título no puede estar vacío")
		return
	}

	metadatos := vista.PedirMetadatos()
	rutaMp3 := vista.PedirRutaMp3()

	if !vista.MostrarResumenConfirmacion(tipo, titulo, metadatos, rutaMp3) {
		fmt.Println("Operación cancelada.")
		return
	}

	id, err := f.RegistrarYSubirAudio(tipo.Id, titulo, metadatos, rutaMp3)
	if err != nil {
		vista.MostrarError(err.Error())
		return
	}
	vista.MostrarExito(id)
}
