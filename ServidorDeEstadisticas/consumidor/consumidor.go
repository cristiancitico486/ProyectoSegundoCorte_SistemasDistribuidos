// Package consumidor encapsula el consumo de la cola de RabbitMQ en la que
// el ServidorDeStreaming publica cada reproducción. Reintenta la conexión
// mientras RabbitMQ no esté disponible, así el orden de arranque de los
// servidores no importa.
package consumidor

import (
	"fmt"
	"os"
	"time"

	"github.com/streadway/amqp"
)

const NombreCola = "reproducciones_audio"

const (
	urlPorDefecto        = "amqp://admin:1234@localhost:5672/"
	esperaEntreIntentos  = 3 * time.Second
	mensajesSinConfirmar = 1
)

// ManejadorMensaje procesa el cuerpo de un mensaje. Si devuelve error, el
// mensaje se descarta (no se vuelve a encolar).
type ManejadorMensaje func(cuerpo []byte) error

type ConsumidorCola struct {
	url string
}

// NuevoConsumidorCola usa la variable de entorno RABBITMQ_URL o, si no
// existe, la URL local por defecto.
func NuevoConsumidorCola() *ConsumidorCola {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = urlPorDefecto
	}
	return &ConsumidorCola{url: url}
}

// Consumir bloquea para siempre: conecta, declara la cola y entrega cada
// mensaje al manejador. Si se cae la conexión, vuelve a intentarlo.
func (c *ConsumidorCola) Consumir(manejador ManejadorMensaje) {
	for {
		if err := c.consumirUnaConexion(manejador); err != nil {
			fmt.Printf("[Cola] %v. Reintentando en %s...\n", err, esperaEntreIntentos)
		}
		time.Sleep(esperaEntreIntentos)
	}
}

func (c *ConsumidorCola) consumirUnaConexion(manejador ManejadorMensaje) error {
	conexion, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("no se pudo conectar a RabbitMQ: %w", err)
	}
	defer conexion.Close()

	canal, err := conexion.Channel()
	if err != nil {
		return fmt.Errorf("no se pudo abrir el canal: %w", err)
	}
	defer canal.Close()

	// Mismos parámetros que usa el publicador: durable, sin autoborrado.
	if _, err := canal.QueueDeclare(NombreCola, true, false, false, false, nil); err != nil {
		return fmt.Errorf("no se pudo declarar la cola: %w", err)
	}
	if err := canal.Qos(mensajesSinConfirmar, 0, false); err != nil {
		return fmt.Errorf("no se pudo configurar el canal: %w", err)
	}

	mensajes, err := canal.Consume(NombreCola, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("no se pudo iniciar el consumo: %w", err)
	}

	fmt.Printf("[Cola] Conectado a RabbitMQ, esperando mensajes en %q...\n", NombreCola)
	for mensaje := range mensajes {
		if err := manejador(mensaje.Body); err != nil {
			fmt.Printf("[Cola] Mensaje descartado: %v\n", err)
			mensaje.Nack(false, false)
			continue
		}
		mensaje.Ack(false)
	}
	return fmt.Errorf("se cerró la conexión con RabbitMQ")
}
