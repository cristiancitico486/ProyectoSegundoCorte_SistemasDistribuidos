/* suscripciones.js: planes y pagos con token idempotente. Sin dependencias; funciona abierto como file://.
   const susc = Suscripciones.montar({ contenedor: "#panel-suscripciones", usuarioId: () => id, apiBase, alEco });
   Devuelve { cargarPlanes, pagar, reutilizarUltimoToken, consultarEstado, destruir }. Ninguna función lanza excepciones. */
(function (global) {
  "use strict";
  var CSS = ".susc-raiz{font:inherit;color:inherit}.susc-raiz *{box-sizing:border-box}" +
    ".susc-planes{display:flex;flex-wrap:wrap;gap:.75rem;margin:.5rem 0}" +
    ".susc-plan{border:1px solid rgba(128,128,128,.45);border-radius:8px;padding:.75rem;min-width:11rem;flex:1}" +
    ".susc-plan h4{margin:0 0 .25rem}.susc-precio{font-weight:700}.susc-raiz small{opacity:.75}" +
    ".susc-raiz button{font:inherit;padding:.35rem .8rem;border:1px solid rgba(128,128,128,.6);border-radius:6px;background:transparent;color:inherit;cursor:pointer;margin-top:.4rem}" +
    ".susc-raiz button:disabled{opacity:.45;cursor:not-allowed}.susc-sim{display:flex;flex-wrap:wrap;gap:1rem;margin:.4rem 0}" +
    ".susc-estado{font-weight:600;margin:.4rem 0}.susc-ecos{list-style:none;margin:.5rem 0 0;padding:.5rem;max-height:14rem;overflow:auto;" +
    "border:1px solid rgba(128,128,128,.45);border-radius:6px;font:.85rem ui-monospace,monospace}.susc-servidor{color:#1a7f37}.susc-cliente{opacity:.9}";
  var pausa = function (ms) { return new Promise(function (r) { setTimeout(r, ms); }); };
  var hora = function () { return new Date().toLocaleTimeString("es-CO"); };
  function el(tag, cls, txt) { var e = document.createElement(tag); if (cls) e.className = cls; if (txt != null) e.textContent = txt; return e; }
  function nuevoToken() {
    var c = global.crypto;
    if (c && c.randomUUID) return "tok-" + c.randomUUID();
    var b = new Uint8Array(16), s = "";
    if (c && c.getRandomValues) c.getRandomValues(b); else for (var i = 0; i < 16; i++) b[i] = Math.floor(Math.random() * 256);
    for (var j = 0; j < 16; j++) s += ("0" + b[j].toString(16)).slice(-2);
    return "tok-" + s;
  }

  function montar(op) {
    op = op || {};
    var cfg = { api: String(op.apiBase || "http://localhost:9000/api").replace(/\/+$/, ""), timeout: op.timeoutMs || 2000,
      max: op.maxIntentos || 3, espera: op.esperaEntreIntentosMs != null ? op.esperaEntreIntentosMs : 500 };
    var raiz = typeof op.contenedor === "string" ? document.querySelector(op.contenedor) : op.contenedor;
    var vivo = true, ultimo = null, ocupado = false, ctls = [], ui = {}, botones = [];
    function fallo(codigo, mensaje) { return { ok: false, estado: "error", repetido: false, almacenado: false, token: null, intentos: 0, respuesta: null, error: { codigo: codigo, mensaje: mensaje } }; }
    if (!raiz) {
      console.error("[Suscripciones] contenedor no encontrado:", op.contenedor);
      var vacio = function () { return Promise.resolve(fallo("contenedor_no_encontrado", "contenedor no encontrado")); };
      return { cargarPlanes: vacio, pagar: vacio, reutilizarUltimoToken: vacio, consultarEstado: vacio, destruir: function () {} };
    }

    function eco(origen, mensaje) {
      var e = { hora: hora(), origen: origen, mensaje: mensaje };
      console.log("[" + e.hora + "] " + mensaje);
      if (typeof op.alEco === "function") { try { op.alEco(e); } catch (x) {} }
      else if (ui.ecos && vivo) { ui.ecos.appendChild(el("li", "susc-" + origen, "[" + e.hora + "] " + mensaje)); ui.ecos.scrollTop = ui.ecos.scrollHeight; }
    }
    function usuario() { try { return String(typeof op.usuarioId === "function" ? op.usuarioId() : op.usuarioId || "").trim(); } catch (e) { return ""; } }
    async function pedir(metodo, ruta, cuerpo, timeout) {
      var ctl = new AbortController(), t = timeout ? setTimeout(function () { ctl.abort(); }, timeout) : null;
      ctls.push(ctl);
      try {
        var r = await fetch(cfg.api + ruta, { method: metodo, signal: ctl.signal,
          headers: cuerpo ? { "Content-Type": "application/json" } : undefined, body: cuerpo ? JSON.stringify(cuerpo) : undefined });
        var d = null; try { d = await r.json(); } catch (e) {}
        return { status: r.status, datos: d };
      } finally { if (t) clearTimeout(t); ctls.splice(ctls.indexOf(ctl), 1); }
    }
    function progreso(t) { if (ui.progreso && vivo) ui.progreso.textContent = t; }
    function marcar(b) { ocupado = b; botones.forEach(function (x) { x.disabled = b; }); if (ui.repetir) ui.repetir.disabled = b || !ultimo; }

    async function enviarPago(planId, sim, tokenFijo, usuarioFijo) {
      var uid = usuarioFijo || usuario(), res = fallo("sin_respuesta", "sin respuesta");
      if (!uid) { eco("cliente", "[Pago] No hay usuarioId: no se puede pagar"); return fallo("usuario_invalido", "no hay usuarioId"); }
      var token = tokenFijo || nuevoToken(), motivo = "";
      res.token = token;
      if (!tokenFijo) eco("cliente", "[Pago] Token generado: " + token);
      for (var i = 1; i <= cfg.max && vivo; i++) {
        res.intentos = i;
        if (i > 1) eco("cliente", "[Pago] Sin respuesta en " + cfg.timeout + " ms: se reenvía con el MISMO token (intento " + i + "/" + cfg.max + ")");
        eco("cliente", "[Pago] Enviando pago (plan=" + planId + ", intento " + i + "/" + cfg.max + ", timeout " + cfg.timeout + " ms)");
        progreso("Procesando pago… intento " + i + "/" + cfg.max);
        var r;
        try { r = await pedir("POST", "/pagos", { token: token, usuarioId: uid, planId: planId, simulacion: sim || "ninguna" }, cfg.timeout); }
        catch (e) { motivo = e && e.name === "AbortError" ? "sin respuesta en " + cfg.timeout + " ms" : "error de red (" + (e && e.message) + ")"; if (i < cfg.max) await pausa(cfg.espera); continue; }
        var d = r.datos || {};
        if (r.status === 200 || r.status === 201) {
          eco("servidor", "[Pago] Pago exitoso: aprobado por el servidor (" + d.monto + " " + d.moneda + ")");
          if (d.almacenado) eco("servidor", "[Pago] Pago almacenado por el servidor (pagoId=" + d.pagoId + ")");
          if (d.repetido) eco("servidor", "[Pago] El servidor ya había usado este token: devolvió el resultado original, sin cobrar de nuevo");
          ultimo = { token: token, planId: planId, usuarioId: uid };
          res.ok = true; res.estado = d.estado || "aprobado"; res.repetido = !!d.repetido; res.almacenado = !!d.almacenado; res.respuesta = d; res.error = null;
          return res;
        }
        if (r.status === 502 || r.status === 503 || r.status === 504) { motivo = "servicio no disponible (HTTP " + r.status + ")"; if (i < cfg.max) await pausa(cfg.espera); continue; }
        res.error = { codigo: d.error || "http_" + r.status, mensaje: d.mensaje || "HTTP " + r.status };
        res.estado = r.status === 402 ? "rechazado" : "error";
        eco("servidor", r.status === 402 ? "[Pago] Pago rechazado: " + res.error.mensaje : "[Pago] Error en el pago (" + res.error.codigo + "): " + res.error.mensaje);
        return res;
      }
      res.error = { codigo: "sin_respuesta", mensaje: motivo };
      eco("cliente", "[Pago] Falló tras " + res.intentos + " intentos: " + motivo);
      return res;
    }
    async function envolver(fn) {
      marcar(true);
      try { var r = await fn(); await consultarEstado(); return r; }
      catch (e) { eco("cliente", "[Pago] Error inesperado: " + (e && e.message)); return fallo("error_inesperado", String(e && e.message)); }
      finally { progreso(""); if (vivo) marcar(false); }
    }
    function pagar(planId, o) { return envolver(function () { return enviarPago(planId, (o && o.simulacion) || "ninguna"); }); }
    function reutilizarUltimoToken() {
      if (!ultimo) { eco("cliente", "[Pago] Todavía no hay un pago exitoso cuyo token reutilizar"); return Promise.resolve(fallo("sin_pago_previo", "no hay un pago previo")); }
      var u = ultimo;
      eco("cliente", "[Pago] Simulando reutilización del token " + u.token + " (el del último pago)");
      return envolver(function () { return enviarPago(u.planId, "ninguna", u.token, u.usuarioId); });
    }
    async function consultarEstado() {
      var uid = usuario();
      if (!uid) return fallo("usuario_invalido", "no hay usuarioId");
      try {
        var r = await pedir("GET", "/suscripciones/" + encodeURIComponent(uid), null, 5000);
        if (r.status !== 200) throw new Error("HTTP " + r.status);
        var s = r.datos;
        if (vivo) {
          ui.estado.textContent = s.activa ? "Suscripción activa: " + s.plan + " (hasta " + new Date(s.vigenteHasta).toLocaleDateString("es-CO", { day: "numeric", month: "long", year: "numeric" }) + ")" : "Sin suscripción activa";
          ui.token.textContent = ultimo ? "Último token usado: " + ultimo.token : "";
          if (ui.repetir) ui.repetir.disabled = ocupado || !ultimo;
        }
        return s;
      } catch (e) {
        if (vivo) ui.estado.textContent = "No se pudo consultar la suscripción (" + (e && e.message) + ")";
        return fallo("consulta_fallida", String(e && e.message));
      }
    }
    async function cargarPlanes() {
      try {
        var r = await pedir("GET", "/planes", null, 5000);
        if (r.status !== 200) throw new Error("HTTP " + r.status);
        if (!vivo) return r.datos;
        ui.planes.textContent = ""; botones = [];
        r.datos.forEach(function (p) {
          var c = el("div", "susc-plan"), b = el("button", null, "Pagar");
          c.appendChild(el("h4", null, p.nombre));
          c.appendChild(el("div", "susc-precio", "$" + new Intl.NumberFormat("es-CO").format(p.precio) + " " + p.moneda));
          c.appendChild(el("small", null, "por " + p.duracionDias + " días. " + p.descripcion)); c.appendChild(el("br"));
          b.onclick = function () { pagar(p.id, { simulacion: ui.lenta.checked ? "demora" : ui.rechazo.checked ? "rechazo" : "ninguna" }); };
          botones.push(b); c.appendChild(b); ui.planes.appendChild(c);
        });
        marcar(ocupado);
        return r.datos;
      } catch (e) {
        if (vivo) ui.planes.textContent = "No se pudieron cargar los planes (" + (e && e.message) + "). Revisa que el gateway esté en " + cfg.api;
        return [];
      }
    }

    if (!document.getElementById("susc-estilos")) { var st = el("style", null, CSS); st.id = "susc-estilos"; document.head.appendChild(st); }
    raiz.textContent = ""; raiz.classList.add("susc-raiz");
    ui.estado = el("div", "susc-estado", "Consultando suscripción…"); ui.token = el("small"); ui.planes = el("div", "susc-planes", "Cargando planes…");
    var sim = el("div", "susc-sim");
    function casilla(txt, otra) { var l = el("label"), c = el("input"); c.type = "checkbox"; c.onchange = function () { if (c.checked) otra().checked = false; }; l.appendChild(c); l.appendChild(document.createTextNode(" " + txt)); sim.appendChild(l); return c; }
    ui.lenta = casilla("Simular respuesta lenta (el cliente reenvía con el mismo token)", function () { return ui.rechazo; });
    ui.rechazo = casilla("Simular rechazo del pago", function () { return ui.lenta; });
    ui.repetir = el("button", null, "Simular token repetido"); ui.repetir.disabled = true; ui.repetir.onclick = reutilizarUltimoToken;
    ui.progreso = el("small");
    [ui.estado, ui.token, ui.planes, sim, ui.repetir, el("br"), ui.progreso].forEach(function (n) { raiz.appendChild(n); });
    if (typeof op.alEco !== "function") { raiz.appendChild(el("h4", null, "Ecos del pago")); ui.ecos = el("ul", "susc-ecos"); raiz.appendChild(ui.ecos); }
    cargarPlanes(); consultarEstado();

    return { cargarPlanes: cargarPlanes, pagar: pagar, reutilizarUltimoToken: reutilizarUltimoToken, consultarEstado: consultarEstado,
      destruir: function () { vivo = false; ctls.forEach(function (c) { c.abort(); }); raiz.textContent = ""; raiz.classList.remove("susc-raiz"); } };
  }
  global.Suscripciones = { montar: montar };
})(typeof window !== "undefined" ? window : this);
