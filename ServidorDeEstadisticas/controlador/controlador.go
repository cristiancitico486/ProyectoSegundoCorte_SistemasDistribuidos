// Package controlador conecta la cola con la lógica del servidor de
// estadísticas (la C de MVC). Cada mensaje consumido imprime un eco por
// printf, como exige el requerimiento.
package controlador

import (
	"encoding/json"
	"fmt"

	"estadisticas.local/servidor-estadisticas/consumidor"
	"estadisticas.local/servidor-estadisticas/dto"
	"estadisticas.local/servidor-estadisticas/fachada"
	"estadisticas.local/servidor-estadisticas/vista"
)

type ControladorEstadisticas struct {
	fachada    *fachada.FachadaEstadisticas
	consumidor *consumidor.ConsumidorCola
}

func NuevoControladorEstadisticas(f *fachada.FachadaEstadisticas, c *consumidor.ConsumidorCola) *ControladorEstadisticas {
	return &ControladorEstadisticas{fachada: f, consumidor: c}
}

// Iniciar empieza a consumir la cola y no retorna.
func (c *ControladorEstadisticas) Iniciar() {
	c.consumidor.Consumir(c.atenderMensaje)
}

// atenderMensaje procesa un mensaje de la cola: eco, decodificación JSON,
// almacenamiento e impresión por pantalla.
func (c *ControladorEstadisticas) atenderMensaje(cuerpo []byte) error {
	fmt.Printf("[Cola] Mensaje consumido de %q: %s\n", consumidor.NombreCola, string(cuerpo))

	var entrada dto.ReproduccionDTO
	if err := json.Unmarshal(cuerpo, &entrada); err != nil {
		return fmt.Errorf("JSON inválido: %w", err)
	}

	reproduccion, resumen, err := c.fachada.RegistrarReproduccion(entrada)
	if err != nil {
		return err
	}

	vista.MostrarReproduccion(reproduccion)
	vista.MostrarResumen(resumen)
	return nil
}
