// Package vista es la capa de presentación por consola del administrador.
// Solo imprime pantallas y lee lo que el usuario escribe; no decide qué
// hacer con esa entrada, eso es responsabilidad del controlador.
package vista

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"administrador.local/administrador-cliente/dto"
)

var lector = bufio.NewReader(os.Stdin)

const encabezado = "Spotify - Administrador"

// leerLinea lee una línea de texto libre (título, ruta de archivo, clave o
// valor de metadato). Si la entrada se cierra (EOF), termina el programa.
func leerLinea(mensaje string) string {
	fmt.Print(mensaje)
	linea, err := lector.ReadString('\n')
	if err == io.EOF {
		fmt.Println("\nEntrada finalizada, cerrando el administrador.")
		os.Exit(0)
	}
	return strings.TrimSpace(linea)
}

// LeerOpcion pide un número al usuario. Devuelve -1 si lo escrito no era un
// número, para que el controlador lo trate como opción inválida.
func LeerOpcion() int {
	texto := leerLinea("Seleccione una opción: ")
	opcion, err := strconv.Atoi(texto)
	if err != nil {
		return -1
	}
	return opcion
}

// MostrarMenuPrincipal es la pantalla inicial del administrador.
func MostrarMenuPrincipal() {
	fmt.Println()
	fmt.Println(encabezado)
	fmt.Println("1. Subir nuevo audio")
	fmt.Println("2. Salir")
}

// MostrarTipos lista los tipos de audio disponibles para elegir uno.
func MostrarTipos(tipos []dto.TipoAudioDTO) {
	fmt.Println()
	fmt.Println(encabezado)
	fmt.Println("Elija el tipo del audio nuevo:")
	for i, t := range tipos {
		fmt.Printf("%d. %s\n", i+1, t.Nombre)
	}
}

// PedirTitulo solicita el título del audio nuevo.
func PedirTitulo() string {
	return leerLinea("Título del audio: ")
}

// PedirMetadatos pide pares clave/valor hasta que el administrador deja la
// clave vacía. Los metadatos son libres por diseño: el administrador
// decide qué campos aplican según el tipo elegido (Artista, Narrador, ISBN,
// etc., según lo que muestre el detalle de ese tipo en el Cliente).
func PedirMetadatos() map[string]string {
	fmt.Println("\nIngrese los metadatos del audio (clave vacía para terminar).")
	metadatos := map[string]string{}
	for {
		clave := leerLinea("  Clave: ")
		if clave == "" {
			break
		}
		valor := leerLinea("  Valor: ")
		metadatos[clave] = valor
	}
	return metadatos
}

// PedirRutaMp3 solicita la ruta local del archivo mp3 a subir.
func PedirRutaMp3() string {
	return leerLinea("Ruta local del archivo .mp3: ")
}

// MostrarResumenConfirmacion muestra lo capturado antes de enviarlo, y
// devuelve true si el administrador confirma.
func MostrarResumenConfirmacion(tipo dto.TipoAudioDTO, titulo string, metadatos map[string]string, rutaMp3 string) bool {
	fmt.Println()
	fmt.Println(encabezado)
	fmt.Println("Resumen del audio a registrar:")
	fmt.Printf("  Tipo: %s\n", tipo.Nombre)
	fmt.Printf("  Título: %s\n", titulo)
	claves := make([]string, 0, len(metadatos))
	for clave := range metadatos {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	for _, clave := range claves {
		fmt.Printf("  %s: %s\n", clave, metadatos[clave])
	}
	fmt.Printf("  Archivo: %s\n", rutaMp3)
	fmt.Println()
	fmt.Println("1. Confirmar y subir")
	fmt.Println("2. Cancelar")
	return LeerOpcion() == 1
}

// MostrarExito confirma que el audio quedó registrado y subido.
func MostrarExito(id int) {
	fmt.Printf("\nAudio registrado y subido correctamente. id=%d\n", id)
}

// MostrarError imprime un mensaje de error de forma consistente.
func MostrarError(mensaje string) {
	fmt.Printf("\nOcurrió un error: %s\n", mensaje)
}

// MostrarOpcionInvalida se usa cuando el usuario escribe algo fuera de rango.
func MostrarOpcionInvalida() {
	fmt.Println("Opción inválida, intente de nuevo.")
}
