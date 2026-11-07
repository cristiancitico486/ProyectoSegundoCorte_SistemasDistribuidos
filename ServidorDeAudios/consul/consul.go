// Package consul registra el servicio en Consul usando su API HTTP (sin
// dependencias externas). Está DESACTIVADO por defecto: solo se registra si
// la variable de entorno CONSUL_ADDR está definida (por ejemplo
// "localhost:8500"). Si no lo está, el servicio arranca igual que antes.
//
// Variables opcionales: SERVICE_ADDR (dirección que Consul usará para
// alcanzar este servicio; por defecto 127.0.0.1).
package consul

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const esperaEntreIntentos = 3 * time.Second

type check struct {
	HTTP                           string `json:"HTTP,omitempty"`
	TCP                            string `json:"TCP,omitempty"`
	Interval                       string `json:"Interval"`
	DeregisterCriticalServiceAfter string `json:"DeregisterCriticalServiceAfter"`
}

type registro struct {
	ID      string `json:"ID"`
	Name    string `json:"Name"`
	Address string `json:"Address"`
	Port    int    `json:"Port"`
	Check   check  `json:"Check"`
}

// Registrar registra el servicio en segundo plano (reintenta hasta que
// Consul responda, así el orden de arranque no importa) y devuelve la
// función que lo desregistra. Si rutaHealth es "", usa un check TCP al
// puerto; si no, un check HTTP a esa ruta (por ejemplo "/health").
func Registrar(nombre string, puerto int, rutaHealth string) func() {
	addr := os.Getenv("CONSUL_ADDR")
	if addr == "" {
		return func() {}
	}
	return registrarEn(addr, nombre, puerto, rutaHealth)
}

func registrarEn(addr, nombre string, puerto int, rutaHealth string) func() {
	if !strings.HasPrefix(addr, "http") {
		addr = "http://" + addr
	}
	host := os.Getenv("SERVICE_ADDR")
	if host == "" {
		host = "127.0.0.1"
	}
	hostPuerto := fmt.Sprintf("%s:%d", host, puerto)
	c := check{Interval: "10s", DeregisterCriticalServiceAfter: "1m"}
	if rutaHealth == "" {
		c.TCP = hostPuerto
	} else {
		c.HTTP = "http://" + hostPuerto + rutaHealth
	}
	reg := registro{ID: fmt.Sprintf("%s-%d", nombre, puerto), Name: nombre, Address: host, Port: puerto, Check: c}

	cuerpo, _ := json.Marshal(reg)
	cancelar := make(chan struct{})
	go func() {
		for {
			if err := enviar(http.MethodPut, addr+"/v1/agent/service/register", cuerpo); err == nil {
				fmt.Printf("[Consul] Servicio %q registrado en %s (%s)\n", nombre, addr, hostPuerto)
				return
			} else {
				fmt.Printf("[Consul] No se pudo registrar %q: %v. Reintentando en %s...\n", nombre, err, esperaEntreIntentos)
			}
			select {
			case <-cancelar:
				return
			case <-time.After(esperaEntreIntentos):
			}
		}
	}()

	return func() {
		close(cancelar)
		if err := enviar(http.MethodPut, addr+"/v1/agent/service/deregister/"+reg.ID, nil); err != nil {
			fmt.Printf("[Consul] No se pudo desregistrar %q: %v\n", nombre, err)
			return
		}
		fmt.Printf("[Consul] Servicio %q desregistrado\n", nombre)
	}
}

func enviar(metodo, url string, cuerpo []byte) error {
	req, err := http.NewRequest(metodo, url, bytes.NewReader(cuerpo))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Consul respondió %s", resp.Status)
	}
	return nil
}
