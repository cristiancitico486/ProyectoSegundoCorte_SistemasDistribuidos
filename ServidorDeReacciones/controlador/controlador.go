// Package controlador atiende el WebSocket de reacciones (la C de MVC). No
// tiene lógica de negocio: decodifica, delega en la fachada y responde. Cada
// evento imprime un eco por printf.
package controlador

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"reacciones.local/servidor-reacciones/dto"
	"reacciones.local/servidor-reacciones/fachada"
	"reacciones.local/servidor-reacciones/repositorio"
	"reacciones.local/servidor-reacciones/ws"
)

const (
	maxMensaje    = 1024
	intervaloPing = 30 * time.Second
	plazoSilencio = 75 * time.Second
)

type ControladorReacciones struct {
	fachada *fachada.FachadaReacciones
}

func NuevoControladorReacciones(f *fachada.FachadaReacciones) *ControladorReacciones {
	return &ControladorReacciones{fachada: f}
}

func (c *ControladorReacciones) RegistrarRutas(mux *http.ServeMux) {
	mux.HandleFunc("GET /ws", c.Conectar)
	mux.HandleFunc("GET /health", c.Salud)
}

func (c *ControladorReacciones) Salud(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dto.Salud{Status: "ok"})
}

// Conectar hace el handshake y atiende la conexión hasta que se cierre.
func (c *ControladorReacciones) Conectar(w http.ResponseWriter, r *http.Request) {
	usuarioId := strings.TrimSpace(r.URL.Query().Get("usuarioId"))
	if usuarioId == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(dto.ErrorHTTP{Error: "usuario_invalido", Mensaje: "usuarioId es obligatorio"})
		return
	}
	conn, err := ws.Aceptar(w, r, maxMensaje)
	if err != nil {
		return
	}
	cliente := &repositorio.Cliente{UsuarioId: usuarioId, Enviar: conn.EscribirTexto}
	fmt.Printf("[WS] Usuario %q conectado desde %s\n", usuarioId, r.RemoteAddr)

	hecho := make(chan struct{})
	defer func() {
		close(hecho)
		c.fachada.Desconectar(cliente)
		conn.Cerrar()
		fmt.Printf("[WS] Usuario %q desconectado\n", usuarioId)
	}()
	go func() {
		t := time.NewTicker(intervaloPing)
		defer t.Stop()
		for {
			select {
			case <-hecho:
				return
			case <-t.C:
				if conn.Ping() != nil {
					return
				}
			}
		}
	}()

	for {
		cuerpo, err := conn.LeerMensaje(plazoSilencio)
		if errors.Is(err, ws.ErrMensajeGrande) {
			c.enviarError(cliente, "mensaje_invalido", "el mensaje supera 1 KB")
			return
		}
		if err != nil {
			if err != io.EOF {
				fmt.Printf("[WS] Conexión de %q terminada: %v\n", usuarioId, err)
			}
			return
		}
		c.atender(cliente, cuerpo)
	}
}

func (c *ControladorReacciones) atender(cliente *repositorio.Cliente, cuerpo []byte) {
	var in dto.Entrada
	if err := json.Unmarshal(cuerpo, &in); err != nil || in.AudioId == nil {
		c.enviarError(cliente, "mensaje_invalido", "se esperaba JSON con tipo y audioId")
		return
	}
	audioId := *in.AudioId
	switch in.Tipo {
	case "registrar":
		ack := c.fachada.Registrar(cliente, audioId)
		fmt.Printf("[WS] %q registrado en audio %d (oyentes: %d)\n", cliente.UsuarioId, audioId, ack.Oyentes)
		c.enviarJSON(cliente, ack)
	case "desregistrar":
		ack := c.fachada.Desregistrar(cliente, audioId)
		fmt.Printf("[WS] %q desregistrado del audio %d (oyentes: %d)\n", cliente.UsuarioId, audioId, ack.Oyentes)
		c.enviarJSON(cliente, ack)
	case "reaccion":
		msg, destinos, err := c.fachada.Reaccionar(cliente, audioId, in.Reaccion)
		switch {
		case errors.Is(err, fachada.ErrNoRegistrado):
			c.enviarError(cliente, "no_registrado", err.Error())
		case errors.Is(err, fachada.ErrReaccionInvalida):
			c.enviarError(cliente, "reaccion_invalida", "reacciones válidas: me_gusta, corazon, aplauso, fuego, risa")
		case err != nil:
			c.enviarError(cliente, "mensaje_invalido", err.Error())
		default:
			b, _ := json.Marshal(msg)
			for _, d := range destinos {
				d.Enviar(b)
			}
			fmt.Printf("[WS] Reacción %q de %q en audio %d reenviada a %d oyentes\n", msg.Reaccion, msg.UsuarioId, audioId, len(destinos))
		}
	default:
		c.enviarError(cliente, "mensaje_invalido", "tipo desconocido: registrar, desregistrar o reaccion")
	}
}

func (c *ControladorReacciones) enviarJSON(cliente *repositorio.Cliente, v any) {
	b, _ := json.Marshal(v)
	cliente.Enviar(b)
}

func (c *ControladorReacciones) enviarError(cliente *repositorio.Cliente, codigo, mensaje string) {
	c.enviarJSON(cliente, dto.Error{Tipo: "error", Codigo: codigo, Mensaje: mensaje})
}
