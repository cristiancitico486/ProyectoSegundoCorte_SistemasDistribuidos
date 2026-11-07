// Package vista es la capa de presentación por consola (la V de MVC). Solo
// imprime menús y lee lo que el usuario escribe; no decide qué hacer con esa
// entrada, eso es responsabilidad del controlador.
package vista

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"cliente.local/cliente-audio/dto"
)

// lector es el único bufio.Reader sobre os.Stdin de todo el cliente. Debe
// reutilizarse (no crear otro con bufio.NewReader(os.Stdin) en otro paquete),
// porque dos lectores independientes sobre el mismo stdin se pisan el buffer
// entre sí y pierden o duplican entradas.
var lector = bufio.NewReader(os.Stdin)

const encabezado = "Spotify"

// LeerOpcion pide un número al usuario y lo devuelve. Devuelve -1 si lo
// escrito no era un número, para que el controlador lo trate como opción
// inválida sin necesidad de repetir el parseo en cada lugar. Si la entrada
// se cierra (EOF, por ejemplo al redirigir un archivo o al correr en un
// pipe), termina el programa en vez de quedarse en un bucle leyendo líneas
// vacías para siempre.
func LeerOpcion() int {
	fmt.Print("Seleccione una opción: ")
	linea, err := lector.ReadString('\n')
	if err == io.EOF {
		fmt.Println("\nEntrada finalizada, cerrando el cliente.")
		os.Exit(0)
	}
	linea = strings.TrimSpace(linea)
	opcion, convErr := strconv.Atoi(linea)
	if convErr != nil {
		return -1
	}
	return opcion
}

// MostrarMenuPrincipal es la pantalla inicial.
func MostrarMenuPrincipal() {
	fmt.Println()
	fmt.Println(encabezado)
	fmt.Println("1. Ver tipos de audio")
	fmt.Println("2. Salir")
}

// MostrarTipos lista los tipos de audio disponibles más "Atrás".
func MostrarTipos(tipos []dto.TipoAudioDTO) {
	fmt.Println()
	fmt.Println(encabezado)
	for i, t := range tipos {
		fmt.Printf("%d. %s\n", i+1, t.Nombre)
	}
	fmt.Printf("%d. Atrás\n", len(tipos)+1)
}

// MostrarAudios lista los audios de un tipo más "Atrás".
func MostrarAudios(tipo dto.TipoAudioDTO, audios []dto.AudioResumenDTO) {
	fmt.Println()
	fmt.Println(encabezado)
	fmt.Printf("Tipo: %s\n\n", tipo.Nombre)
	for i, a := range audios {
		fmt.Printf("%d. %s\n", i+1, a.Titulo)
	}
	fmt.Printf("%d. Atrás\n", len(audios)+1)
}

// MostrarDetalle muestra los metadatos de un audio (ordenados por nombre)
// y ofrece reproducirlo o volver.
func MostrarDetalle(detalle dto.AudioDetalleDTO) {
	fmt.Println()
	fmt.Println(encabezado)
	fmt.Printf("%s\n\n", detalle.Titulo)
	claves := make([]string, 0, len(detalle.Metadatos))
	for clave := range detalle.Metadatos {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	for _, clave := range claves {
		fmt.Printf("  %s: %s\n", clave, detalle.Metadatos[clave])
	}
	fmt.Println()
	fmt.Println("1. Reproducir")
	fmt.Println("2. Atrás")
}

// MostrarError imprime un mensaje de error de forma consistente. No es
// redundante con los ecos de red: este es el mensaje que ve el usuario final.
func MostrarError(mensaje string) {
	fmt.Println()
	fmt.Printf("Ocurrió un error: %s\n", mensaje)
}

// MostrarOpcionInvalida se usa cuando el usuario escribe algo fuera de rango.
func MostrarOpcionInvalida() {
	fmt.Println("Opción inválida, intente de nuevo.")
}

// MostrarPantallaReproduccion es la pantalla que se ve mientras suena el
// audio; permite salir en cualquier momento.
func MostrarPantallaReproduccion(titulo string) {
	fmt.Println()
	fmt.Println(encabezado)
	fmt.Printf("%s\n\n", titulo)
	fmt.Println("Reproduciendo audio")
	fmt.Println("1. Salir")
}
