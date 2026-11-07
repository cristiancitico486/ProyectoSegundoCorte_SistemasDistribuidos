// Package repositorio es la capa de acceso a datos: guarda y lee los
// archivos .mp3 en la carpeta local "audios/". El nombre de archivo usado en
// todo el sistema es "{id}.mp3", donde id es el mismo id que asigna el
// ServidorMetadataDeAudios al registrar los metadatos.
package repositorio

import (
	"fmt"
	"os"
	"path/filepath"
)

const carpetaAudios = "audios"

type RepositorioAudios struct{}

func NuevoRepositorioAudios() *RepositorioAudios {
	os.MkdirAll(carpetaAudios, os.ModePerm)
	return &RepositorioAudios{}
}

func rutaArchivo(nombreMp3 string) string {
	return filepath.Join(carpetaAudios, nombreMp3)
}

// GuardarMp3 escribe los bytes del audio en disco con el nombre "{id}.mp3".
func (r *RepositorioAudios) GuardarMp3(nombreMp3 string, data []byte) error {
	if err := os.WriteFile(rutaArchivo(nombreMp3), data, 0644); err != nil {
		return fmt.Errorf("error al guardar %s: %w", nombreMp3, err)
	}
	return nil
}

// Existe indica si el archivo ya está guardado en disco.
func (r *RepositorioAudios) Existe(nombreMp3 string) bool {
	_, err := os.Stat(rutaArchivo(nombreMp3))
	return err == nil
}

// RutaCompleta devuelve la ruta en disco de un mp3, lista para servirla.
func (r *RepositorioAudios) RutaCompleta(nombreMp3 string) string {
	return rutaArchivo(nombreMp3)
}
