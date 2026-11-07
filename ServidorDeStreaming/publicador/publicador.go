// Package publicador encapsula la publicación asíncrona en RabbitMQ: cada
// reproducción nueva se envía a la cola que consume el ServidorDeEstadisticas.
// Si RabbitMQ no está disponible al arrancar, la conexión se reintenta en la
// siguiente publicación, así el streaming nunca depende de la cola.
package publicador

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/streadway/amqp"
	"streaming.local/servidor-streaming/dto"
)

const NombreCola = "reproducciones_audio"

const urlPorDefecto = "amqp://admin:1234@localhost:5672/"

type PublicadorReproducciones struct {
	mu       sync.Mutex
	url      string
	conexion *amqp.Connection
	canal    *amqp.Channel
}

// NuevoPublicador usa la variable de entorno RABBITMQ_URL o, si no existe,
// la URL local por defecto. Intenta conectar de una vez; si falla, devuelve
// igualmente un publicador que reintentará al publicar.
func NuevoPublicador() (*PublicadorReproducciones, error) {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = urlPorDefecto
	}
	p := &PublicadorReproducciones{url: url}
	return p, p.conectar()
}

// conectar abre la conexión, el canal y declara la cola. Debe llamarse con
// el mutex tomado (o antes de compartir el publicador).
func (p *PublicadorReproducciones) conectar() error {
	conexion, err := amqp.Dial(p.url)
	if err != nil {
		return fmt.Errorf("error conectando a RabbitMQ: %w", err)
	}

	canal, err := conexion.Channel()
	if err != nil {
		conexion.Close()
		return fmt.Errorf("error abriendo canal: %w", err)
	}

	if _, err := canal.QueueDeclare(NombreCola, true, false, false, false, nil); err != nil {
		canal.Close()
		conexion.Close()
		return fmt.Errorf("error declarando la cola: %w", err)
	}

	p.conexion = conexion
	p.canal = canal
	return nil
}

// PublicarReproduccion envía el mensaje de una reproducción nueva a la
// cola. Se llama al iniciar el streaming, no al terminarlo.
func (p *PublicadorReproducciones) PublicarReproduccion(msg dto.ReproduccionDTO) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("error convirtiendo mensaje a JSON: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.canal == nil || p.conexion.IsClosed() {
		if err := p.conectar(); err != nil {
			return err
		}
	}

	err = p.canal.Publish("", NombreCola, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
	if err != nil {
		return fmt.Errorf("error publicando en la cola: %w", err)
	}

	fmt.Printf("[Cola] Mensaje publicado en %q: %s\n", NombreCola, string(body))
	return nil
}

// Cerrar libera el canal y la conexión si están abiertos.
func (p *PublicadorReproducciones) Cerrar() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.canal != nil {
		p.canal.Close()
	}
	if p.conexion != nil {
		p.conexion.Close()
	}
}
