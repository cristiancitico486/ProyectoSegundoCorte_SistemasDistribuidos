// Package fachada contiene la lógica de negocio de las suscripciones: la
// validación del pago y, sobre todo, la idempotencia por token.
//
// Garantías del pago (POST /pagos):
//   - Un token se cobra UNA sola vez. Si llega de nuevo, se devuelve el
//     resultado original (repetido=true) sin cobrar.
//   - Si el reenvío llega mientras el primero sigue en curso, espera ese
//     resultado en lugar de procesar otro pago.
//   - El pago se procesa y se almacena en una goroutine propia, SIN atarlo al
//     contexto de la petición: si el cliente aborta su primer intento (por
//     ejemplo con la simulación "demora"), el pago igual termina y queda
//     almacenado, y el reenvío con el mismo token recibe el resultado original.
//   - Un pago rechazado o un fallo al almacenar NO consume el token.
package fachada

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"suscripciones.local/servidor-suscripciones/dto"
	"suscripciones.local/servidor-suscripciones/modelo"
	"suscripciones.local/servidor-suscripciones/repositorio"
)

const (
	maxLongitudUsuario = 100
	// retrasoDemora es lo que tarda el servidor con la simulación "demora":
	// más que el timeout del cliente (2 s) para forzar el reenvío.
	retrasoDemora = 3500 * time.Millisecond
)

var formatoToken = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)

// ErrorNegocio es un error esperado de la lógica de negocio. Codigo es el
// identificador estable que viaja al cliente en el campo "error".
type ErrorNegocio struct {
	Codigo  string
	Mensaje string
}

func (e *ErrorNegocio) Error() string { return e.Mensaje }

var (
	ErrCuerpoInvalido   = &ErrorNegocio{"cuerpo_invalido", "el cuerpo de la petición no es válido"}
	ErrTokenInvalido    = &ErrorNegocio{"token_invalido", "el token debe tener entre 8 y 64 caracteres (letras, números, _ o -)"}
	ErrUsuarioInvalido  = &ErrorNegocio{"usuario_invalido", "el usuarioId es obligatorio (hasta 100 caracteres)"}
	ErrPlanInvalido     = &ErrorNegocio{"plan_invalido", "el plan indicado no existe"}
	ErrPagoRechazado    = &ErrorNegocio{"pago_rechazado", "el pago fue rechazado; el token no se consumió, puedes reintentar"}
	ErrTokenEnConflicto = &ErrorNegocio{"token_en_conflicto", "el token ya se usó con otro usuario o plan"}
	ErrAlmacenamiento   = &ErrorNegocio{"error_interno", "no se pudo almacenar el pago; el token no se consumió, puedes reintentar"}
)

// procesamiento es un pago en curso, identificado por su token.
type procesamiento struct {
	usuarioId string
	planId    string
	listo     chan struct{} // se cierra al terminar (con pago o con error)
	pago      modelo.Pago
	err       error
}

type FachadaSuscripciones struct {
	repo            *repositorio.RepositorioPagos
	ahora           func() time.Time
	retrasoSimulado time.Duration

	mu         sync.Mutex
	enCurso    map[string]*procesamiento
	pendientes sync.WaitGroup
}

func NuevaFachadaSuscripciones(repo *repositorio.RepositorioPagos) *FachadaSuscripciones {
	return &FachadaSuscripciones{
		repo:            repo,
		ahora:           time.Now,
		retrasoSimulado: retrasoDemora,
		enCurso:         map[string]*procesamiento{},
	}
}

// ConRetrasoSimulado cambia lo que tarda la simulación "demora" (por defecto
// 3,5 s). Existe para que las pruebas no esperen tanto.
func (f *FachadaSuscripciones) ConRetrasoSimulado(d time.Duration) *FachadaSuscripciones {
	f.retrasoSimulado = d
	return f
}

// ListarPlanes responde GET /planes.
func (f *FachadaSuscripciones) ListarPlanes() []dto.PlanDTO {
	planes := modelo.Planes()
	resultado := make([]dto.PlanDTO, 0, len(planes))
	for _, p := range planes {
		resultado = append(resultado, dto.PlanDTO{
			Id: p.Id, Nombre: p.Nombre, Precio: p.Precio, Moneda: p.Moneda,
			DuracionDias: p.DuracionDias, Descripcion: p.Descripcion,
		})
	}
	return resultado
}

// ProcesarPago atiende POST /pagos con idempotencia por token. El contexto
// solo controla cuánto espera ESTA petición; no cancela el pago.
func (f *FachadaSuscripciones) ProcesarPago(ctx context.Context, entrada dto.PagoSolicitudDTO) (dto.PagoRespuestaDTO, error) {
	token := strings.TrimSpace(entrada.Token)
	if !formatoToken.MatchString(token) {
		return dto.PagoRespuestaDTO{}, ErrTokenInvalido
	}
	usuarioId, err := validarUsuario(entrada.UsuarioId)
	if err != nil {
		return dto.PagoRespuestaDTO{}, err
	}
	plan, ok := modelo.BuscarPlan(strings.TrimSpace(entrada.PlanId))
	if !ok {
		return dto.PagoRespuestaDTO{}, ErrPlanInvalido
	}
	simulacion := strings.ToLower(strings.TrimSpace(entrada.Simulacion))
	switch simulacion {
	case "", "ninguna":
		simulacion = ""
	case "demora", "rechazo":
	default:
		return dto.PagoRespuestaDTO{}, &ErrorNegocio{"cuerpo_invalido", `simulacion debe ser "ninguna", "demora" o "rechazo"`}
	}

	fmt.Printf("[Pago] Petición recibida: token=%s usuario=%s plan=%s\n", token, usuarioId, plan.Id)

	f.mu.Lock()
	// 1) Token ya usado: se devuelve el resultado original.
	if pago, existe := f.repo.ObtenerPorToken(token); existe {
		f.mu.Unlock()
		if pago.UsuarioId != usuarioId || pago.PlanId != plan.Id {
			fmt.Println("[Pago] Token EN CONFLICTO: ya se usó con otro usuario o plan")
			return dto.PagoRespuestaDTO{}, ErrTokenEnConflicto
		}
		fmt.Printf("[Pago] Token YA UTILIZADO (pagoId=%s): se devuelve el resultado original, sin cobrar de nuevo\n", pago.PagoId)
		return respuestaDePago(pago, true), nil
	}
	// 2) Token en curso: se espera ese mismo pago.
	if actual, existe := f.enCurso[token]; existe {
		f.mu.Unlock()
		if actual.usuarioId != usuarioId || actual.planId != plan.Id {
			fmt.Println("[Pago] Token EN CONFLICTO: ya se usó con otro usuario o plan")
			return dto.PagoRespuestaDTO{}, ErrTokenEnConflicto
		}
		fmt.Println("[Pago] Reenvío con el mismo token mientras el pago está en curso: se espera el resultado")
		return esperar(ctx, actual, true)
	}
	// 3) Token nuevo: se procesa en segundo plano.
	nuevo := &procesamiento{usuarioId: usuarioId, planId: plan.Id, listo: make(chan struct{})}
	f.enCurso[token] = nuevo
	f.pendientes.Add(1)
	f.mu.Unlock()

	go f.procesar(token, nuevo, plan, simulacion)
	return esperar(ctx, nuevo, false)
}

// esperar bloquea hasta que termine el pago o se cancele ESTA petición (el
// pago sigue su curso aunque la petición se cancele).
func esperar(ctx context.Context, p *procesamiento, repetido bool) (dto.PagoRespuestaDTO, error) {
	select {
	case <-p.listo:
		if p.err != nil {
			return dto.PagoRespuestaDTO{}, p.err
		}
		return respuestaDePago(p.pago, repetido), nil
	case <-ctx.Done():
		return dto.PagoRespuestaDTO{}, ctx.Err()
	}
}

// procesar corre en su propia goroutine, desacoplada de la petición.
func (f *FachadaSuscripciones) procesar(token string, p *procesamiento, plan modelo.Plan, simulacion string) {
	defer f.pendientes.Done()
	fmt.Println("[Pago] Token NUEVO: procesando pago...")

	if simulacion == "demora" {
		time.Sleep(f.retrasoSimulado)
	}
	if simulacion == "rechazo" {
		fmt.Println("[Pago] Pago RECHAZADO (simulado): el token no se consume")
		f.terminar(token, p, modelo.Pago{}, ErrPagoRechazado)
		return
	}

	fmt.Printf("[Pago] Pago APROBADO: %d %s\n", plan.Precio, plan.Moneda)
	ahora := f.ahora()
	pago := modelo.Pago{
		PagoId:       nuevoPagoId(),
		Token:        token,
		UsuarioId:    p.usuarioId,
		PlanId:       plan.Id,
		Plan:         plan.Nombre,
		Monto:        plan.Precio,
		Moneda:       plan.Moneda,
		Fecha:        ahora,
		VigenteHasta: ahora.AddDate(0, 0, plan.DuracionDias),
	}
	if err := f.repo.Guardar(pago); err != nil {
		fmt.Printf("[Pago] ERROR almacenando el pago (el token no se consume): %v\n", err)
		f.terminar(token, p, modelo.Pago{}, ErrAlmacenamiento)
		return
	}
	fmt.Printf("[Pago] Pago ALMACENADO: pagoId=%s en %s\n", pago.PagoId, f.repo.Ruta())
	f.terminar(token, p, pago, nil)
}

// terminar publica el resultado. El pago ya está en el repositorio cuando se
// quita de enCurso, así que una petición siempre encuentra uno de los dos.
func (f *FachadaSuscripciones) terminar(token string, p *procesamiento, pago modelo.Pago, err error) {
	p.pago, p.err = pago, err
	f.mu.Lock()
	delete(f.enCurso, token)
	f.mu.Unlock()
	close(p.listo)
}

// ConsultarSuscripcion responde GET /suscripciones/{usuarioId}.
func (f *FachadaSuscripciones) ConsultarSuscripcion(usuarioId string) (dto.SuscripcionDTO, error) {
	usuarioId, err := validarUsuario(usuarioId)
	if err != nil {
		return dto.SuscripcionDTO{}, err
	}
	pagos := f.repo.ListarPorUsuario(usuarioId)

	respuesta := dto.SuscripcionDTO{UsuarioId: usuarioId, Pagos: make([]dto.PagoHistorialDTO, 0, len(pagos))}
	for _, p := range pagos {
		respuesta.Pagos = append(respuesta.Pagos, dto.PagoHistorialDTO{
			PagoId: p.PagoId, Token: p.Token, PlanId: p.PlanId, Monto: p.Monto,
			Moneda: p.Moneda, Fecha: p.Fecha.Format(time.RFC3339),
		})
	}
	if len(pagos) > 0 { // el más reciente define el plan vigente
		ultimo := pagos[0]
		respuesta.PlanId = ultimo.PlanId
		respuesta.Plan = ultimo.Plan
		respuesta.VigenteHasta = ultimo.VigenteHasta.Format(time.RFC3339)
		respuesta.Activa = f.ahora().Before(ultimo.VigenteHasta)
	}
	return respuesta, nil
}

// EsperarPendientes bloquea hasta que terminen los pagos en curso o venza el
// plazo; se usa al cerrar el servidor para no perder un pago a medias.
func (f *FachadaSuscripciones) EsperarPendientes(plazo time.Duration) {
	hecho := make(chan struct{})
	go func() { f.pendientes.Wait(); close(hecho) }()
	select {
	case <-hecho:
	case <-time.After(plazo):
	}
}

func respuestaDePago(p modelo.Pago, repetido bool) dto.PagoRespuestaDTO {
	return dto.PagoRespuestaDTO{
		Token: p.Token, PagoId: p.PagoId, UsuarioId: p.UsuarioId, PlanId: p.PlanId,
		Plan: p.Plan, Monto: p.Monto, Moneda: p.Moneda, Estado: "aprobado",
		Almacenado: true, Repetido: repetido,
		Fecha: p.Fecha.Format(time.RFC3339), VigenteHasta: p.VigenteHasta.Format(time.RFC3339),
	}
}

func validarUsuario(crudo string) (string, error) {
	usuario := strings.TrimSpace(crudo)
	if usuario == "" || utf8.RuneCountInString(usuario) > maxLongitudUsuario {
		return "", ErrUsuarioInvalido
	}
	for _, r := range usuario {
		if unicode.IsControl(r) {
			return "", ErrUsuarioInvalido
		}
	}
	return usuario, nil
}

func nuevoPagoId() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("pg-%d", time.Now().UnixNano())
	}
	return "pg-" + hex.EncodeToString(b)
}
