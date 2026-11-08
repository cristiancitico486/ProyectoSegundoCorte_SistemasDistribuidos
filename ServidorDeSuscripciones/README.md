# ServidorDeSuscripciones

Planes de suscripción y pagos simulados con **tokens idempotentes** (puerto **8085**, nombre en Consul `suscripciones-service`). Solo biblioteca estándar de Go. Mismo esquema de capas que los demás módulos: main → controlador → fachada → repositorio.

## Rutas

| Ruta | Respuesta |
|---|---|
| `GET /planes` | `[{"id","nombre","precio","moneda":"COP","duracionDias":30,"descripcion"}]` — básico $9.900, estándar $19.900, premium $29.900 |
| `POST /pagos` | pago con token idempotente (abajo) |
| `GET /suscripciones/{usuarioId}` | `{"usuarioId","activa","planId","plan","vigenteHasta","pagos":[{"pagoId","token","planId","monto","moneda","fecha"}]}` — pagos del más reciente al más antiguo; sin pagos: `activa:false`, `pagos:[]` |
| `GET /health` | `{"status":"ok"}` |

## POST /pagos

Cuerpo: `{"token":"tok-<uuid>","usuarioId":"u-ab12","planId":"premium","simulacion":"ninguna|demora|rechazo"}`

- `token`: 8 a 64 caracteres `[A-Za-z0-9_-]`. `simulacion` es opcional y solo sirve para la demostración.
- **Token nuevo** → se aprueba, **se almacena** y responde `201` con `"repetido":false`.
- **Token ya usado** (mismo usuario y plan) → `200` con el resultado original y `"repetido":true`. No cobra de nuevo.
- **Reenvío mientras el primero sigue en curso** → espera ese resultado y responde `200` `"repetido":true`: se cobra una sola vez.
- **Token usado con otro usuario o plan** → `409 token_en_conflicto`.
- `"demora"`: el servidor tarda 3,5 s en responder (el cliente aborta a los 2 s y reenvía). El pago se procesa en una goroutine propia, **no atada a la petición**, así que termina y se almacena aunque el cliente ya se haya ido.
- `"rechazo"`: `402 pago_rechazado`; el token **no** se consume. Tampoco se consume si falla el almacenamiento (`500 error_interno`).
- Otros errores `400`: `cuerpo_invalido`, `token_invalido`, `usuario_invalido`, `plan_invalido`.
- Errores: `{"error":"<codigo>","mensaje":"<texto>"}`. Un pago aprobado deja al usuario con ese plan vigente 30 días.

## Almacenamiento

`datos/pagos.jsonl`: un pago por línea, con sincronización a disco. Al arrancar se relee y los tokens usados se recuerdan tras reiniciar.

## Ecos de la consola

```
[Pago] Petición recibida: token=tok-... usuario=u-... plan=premium
[Pago] Token NUEVO: procesando pago...
[Pago] Pago APROBADO: 29900 COP
[Pago] Pago ALMACENADO: pagoId=... en datos/pagos.jsonl
[Pago] Token YA UTILIZADO (pagoId=...): se devuelve el resultado original, sin cobrar de nuevo
[Pago] Reenvío con el mismo token mientras el pago está en curso: se espera el resultado
[Pago] Pago RECHAZADO (simulado): el token no se consume
[Pago] Token EN CONFLICTO: ya se usó con otro usuario o plan
[Pago] El cliente cerró la conexión antes de recibir la respuesta: el pago sigue procesándose
```

## Ejecutar

```
go run .                                  # escucha en :8085
CONSUL_ADDR=localhost:8500 go run .       # además se registra en Consul
go test ./...
```

Prueba manual del reintento (el primero vence a los 2 s, el segundo recibe el resultado original):

```
curl -m 2 -X POST localhost:8085/pagos -H 'Content-Type: application/json' \
  -d '{"token":"tok-demo-0001","usuarioId":"u1","planId":"premium","simulacion":"demora"}'
curl -X POST localhost:8085/pagos -H 'Content-Type: application/json' \
  -d '{"token":"tok-demo-0001","usuarioId":"u1","planId":"premium","simulacion":"demora"}'
```

Cierra limpio con Ctrl+C o SIGTERM (espera a los pagos en curso y se desregistra de Consul).
