// funciones.js — interfaz del cliente: catálogo, reproductor, canal de
// reacciones (WebSocket) e historial. El streaming lo hace bundle.js
// (cliente_receptor_fragmentos.js), que llama a __marcarAudioCargado.
(function () {
    'use strict';

    const CFG = window.CONFIG;
    const REACCIONES = { me_gusta: '👍', corazon: '❤️', aplauso: '👏', fuego: '🔥', risa: '😂' };

    const estado = {
        usuarioId: obtenerUsuarioId(),
        tipos: [],
        audioSeleccionado: null, // detalle del audio elegido en el catálogo
        audioCargado: null,      // audioId del audio que está en el reproductor
        registradoEn: null       // audioId en el que el servidor nos tiene registrados
    };

    // ---------- utilidades ----------
    function obtenerUsuarioId() {
        try {
            const guardado = localStorage.getItem('usuarioId');
            if (guardado) return guardado;
        } catch (e) { /* sin localStorage: se genera uno por sesión */ }
        const bytes = new Uint8Array(6);
        (window.crypto || { getRandomValues: a => a.forEach((_, i) => { a[i] = Math.random() * 256; }) }).getRandomValues(bytes);
        const id = 'u-' + Array.from(bytes, b => b.toString(16).padStart(2, '0')).join('');
        try { localStorage.setItem('usuarioId', id); } catch (e) { /* ignorar */ }
        return id;
    }

    function el(tag, props, ...hijos) {
        const nodo = document.createElement(tag);
        Object.assign(nodo, props || {});
        hijos.forEach(h => nodo.append(h));
        return nodo;
    }

    function writeLog(message, level) {
        const d = document.getElementById('log');
        if (!d) return;
        const p = document.createElement('div');
        p.className = level || '';
        p.textContent = `[${new Date().toLocaleTimeString()}] ${message}`;
        d.appendChild(p);
        d.scrollTop = d.scrollHeight;
    }

    async function api(ruta) {
        const r = await fetch(CFG.api + ruta);
        if (!r.ok) {
            let detalle = '';
            try { const j = await r.json(); detalle = j.mensaje || j.error || ''; } catch (e) { /* sin cuerpo */ }
            throw new Error(`${r.status} ${detalle}`.trim());
        }
        return r.json();
    }

    function setEstadoCatalogo(texto, nivel) {
        const d = document.getElementById('estado-catalogo');
        d.textContent = texto || '';
        d.className = nivel || 'muted';
    }

    // ---------- catálogo ----------
    async function cargarTipos() {
        try {
            estado.tipos = await api('/tipos');
            const cont = document.getElementById('tipos');
            cont.replaceChildren(...estado.tipos.map(t =>
                el('button', { textContent: t.nombre, onclick: (ev) => elegirTipo(t, ev.currentTarget) })));
            setEstadoCatalogo('');
        } catch (e) {
            setEstadoCatalogo(`No se pudo cargar el catálogo: ${e.message}`, 'error');
            writeLog(`No se pudo cargar el catálogo: ${e.message}`, 'error');
        }
    }

    async function elegirTipo(tipo, boton) {
        document.querySelectorAll('#tipos button').forEach(b => b.classList.remove('activo'));
        boton.classList.add('activo');
        document.getElementById('detalle').replaceChildren();
        try {
            const audios = await api(`/tipos/${tipo.id}/audios`);
            const ul = document.getElementById('audios');
            if (audios.length === 0) {
                ul.replaceChildren(el('li', { textContent: 'No hay audios de este tipo.', className: 'muted' }));
                return;
            }
            ul.replaceChildren(...audios.map(a =>
                el('li', {}, el('a', { textContent: a.titulo, onclick: () => elegirAudio(a) }))));
            setEstadoCatalogo('');
        } catch (e) {
            setEstadoCatalogo(`No se pudo cargar la lista: ${e.message}`, 'error');
        }
    }

    async function elegirAudio(resumen) {
        try {
            const d = await api(`/audios/${resumen.id}`);
            const tipo = estado.tipos.find(t => t.id === d.tipoId);
            d.tipoNombre = tipo ? tipo.nombre : '';
            estado.audioSeleccionado = d;

            const filas = ordenarMetadatos(d.metadatos || {}, d.tipoNombre).map(([k, v]) =>
                el('tr', {}, el('th', { textContent: k }), el('td', { textContent: v })));
            document.getElementById('detalle').replaceChildren(
                el('h3', { textContent: d.titulo }),
                el('div', { textContent: `Tipo: ${d.tipoNombre} · Id: ${d.id}`, className: 'muted' }),
                el('table', {}, ...filas),
                el('button', { textContent: '▶ Reproducir', onclick: reproducirSeleccionado }));
            setEstadoCatalogo('');
        } catch (e) {
            setEstadoCatalogo(`No se pudo cargar el detalle: ${e.message}`, 'error');
        }
    }

    // Metadatos que exige el enunciado por tipo; se muestran primero, en este orden.
    const CLAVES_PRINCIPALES = {
        'musica': ['Artista Principal', 'Álbum', 'Género Musical'],
        'podcasts': ['Nombre del Podcast', 'Título del Episodio', 'Anfitrión'],
        'audiolibros': ['Título del Libro', 'Autor', 'Narrador'],
        'ruido blanco': ['Tipo de Sonido', 'Fuente del Audio', 'Uso Sugerido']
    };

    function normalizar(t) {
        return (t || '').normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLowerCase().trim();
    }

    // Devuelve [clave, valor] con las claves principales primero y el resto
    // (opcionales, de clave libre) después, en orden alfabético.
    function ordenarMetadatos(metadatos, tipoNombre) {
        const principales = CLAVES_PRINCIPALES[normalizar(tipoNombre)] || [];
        const primeras = principales.filter(k => k in metadatos).map(k => [k, metadatos[k]]);
        const resto = Object.entries(metadatos)
            .filter(([k]) => !principales.includes(k))
            .sort((a, b) => a[0].localeCompare(b[0], 'es'));
        return primeras.concat(resto);
    }

    function reproducirSeleccionado() {
        const a = estado.audioSeleccionado;
        if (!a) return;
        // Salir del canal del audio anterior antes de pedir el nuevo.
        desregistrarActual();
        estado.audioCargado = null; // el pause que sigue ya no afecta a ningún canal
        const player = document.getElementById('audio-player');
        if (player) player.pause();
        document.getElementById('status').textContent = `Solicitando "${a.titulo}"...`;
        window.iniciar_streaming_audio(a.id, a.titulo, a.tipoNombre, estado.usuarioId);
    }

    // Lo llama cliente_receptor_fragmentos.js justo antes de asignar el audio al reproductor.
    window.__marcarAudioCargado = function (audioId) {
        estado.audioCargado = audioId;
        const a = estado.audioSeleccionado;
        document.getElementById('status').textContent =
            `Reproduciendo: ${a && a.id === audioId ? a.titulo : 'audio ' + audioId}`;
    };

    // ---------- canal de reacciones (WebSocket) ----------
    let ws = null;
    const cola = [];

    function setEstadoWS(texto, nivel) {
        const s = document.getElementById('estado-ws');
        s.textContent = texto;
        s.className = nivel || '';
    }

    function conectarWS() {
        const url = `${CFG.ws}?usuarioId=${encodeURIComponent(estado.usuarioId)}`;
        ws = new WebSocket(url);
        ws.onopen = () => {
            setEstadoWS('conectado', 'success');
            writeLog('Conectado al canal de reacciones.', 'success');
            const player = document.getElementById('audio-player');
            if (estado.audioCargado !== null && player && !player.paused) {
                enviarWS({ tipo: 'registrar', audioId: estado.audioCargado }); // reconexión
            }
            while (cola.length) ws.send(JSON.stringify(cola.shift()));
        };
        ws.onmessage = (ev) => recibirWS(ev.data);
        ws.onclose = () => {
            setEstadoWS('desconectado (reintentando)', 'error');
            estado.registradoEn = null;
            actualizarBotonesReaccion();
            setTimeout(conectarWS, 3000);
        };
        ws.onerror = () => { /* onclose se encarga del reintento */ };
    }

    function enviarWS(obj) {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify(obj));
        } else {
            cola.push(obj);
        }
    }

    function desregistrarActual() {
        if (estado.registradoEn !== null) {
            enviarWS({ tipo: 'desregistrar', audioId: estado.registradoEn });
        }
    }

    function recibirWS(texto) {
        let m;
        try { m = JSON.parse(texto); } catch (e) { return; }
        if (m.tipo === 'ack') {
            if (m.operacion === 'registrar') {
                estado.registradoEn = m.audioId;
                writeLog(`Registrado en el canal del audio ${m.audioId} (oyentes: ${m.oyentes}).`, 'success');
            } else if (estado.registradoEn === m.audioId) {
                estado.registradoEn = null;
                writeLog(`Desregistrado del canal del audio ${m.audioId}.`, 'info');
            }
            actualizarBotonesReaccion();
        } else if (m.tipo === 'reaccion') {
            const feed = document.getElementById('reacciones');
            const p = el('div', { textContent:
                `[${new Date(m.timestamp).toLocaleTimeString()}] ${m.usuarioId === estado.usuarioId ? 'Tú' : m.usuarioId} ` +
                `reaccionó ${REACCIONES[m.reaccion] || m.reaccion} (audio ${m.audioId})` });
            feed.appendChild(p);
            feed.scrollTop = feed.scrollHeight;
            writeLog(`Reacción recibida: ${m.reaccion} de ${m.usuarioId} en el audio ${m.audioId}.`, 'info');
        } else if (m.tipo === 'error') {
            writeLog(`Canal de reacciones: ${m.codigo} — ${m.mensaje}`, 'error');
        }
    }

    function actualizarBotonesReaccion() {
        document.querySelectorAll('#botones-reaccion button').forEach(b => {
            b.disabled = estado.registradoEn === null;
        });
    }

    function crearBotonesReaccion() {
        const cont = document.getElementById('botones-reaccion');
        cont.replaceChildren(...Object.entries(REACCIONES).map(([codigo, emoji]) =>
            el('button', {
                textContent: emoji, title: codigo, className: 'reaccion-btn', disabled: true,
                onclick: () => {
                    if (estado.registradoEn !== null) {
                        enviarWS({ tipo: 'reaccion', audioId: estado.registradoEn, reaccion: codigo });
                    }
                }
            })));
    }

    // ---------- listeners del reproductor (play / pause) ----------
    function attachAudioListeners() {
        const audio = document.getElementById('audio-player');
        if (!audio) {
            writeLog('No se encontró el elemento audio#audio-player.', 'error');
            return;
        }
        audio.addEventListener('play', function () {
            writeLog('Reproducción iniciada (play)', 'success');
            // Registro en el canal de notificaciones del audio que suena
            if (estado.audioCargado !== null) {
                enviarWS({ tipo: 'registrar', audioId: estado.audioCargado });
            }
        });
        audio.addEventListener('pause', function () {
            writeLog('Reproducción pausada (pause)', 'error');
            // Desregistro del canal (también se dispara al terminar el audio)
            if (estado.audioCargado !== null) {
                enviarWS({ tipo: 'desregistrar', audioId: estado.audioCargado });
            }
        });
    }

    // ---------- historial ----------
    async function verHistorial() {
        const cont = document.getElementById('tabla-historial');
        try {
            const h = await api(`/historial/${encodeURIComponent(estado.usuarioId)}`);
            if (!h.audios || h.audios.length === 0) {
                cont.replaceChildren(el('p', { textContent: 'Aún no has reproducido audios.', className: 'muted' }));
                return;
            }
            const filas = h.audios.map(a => el('tr', {},
                el('td', { textContent: a.titulo }), el('td', { textContent: a.tipo }),
                el('td', { textContent: String(a.cantidad) })));
            cont.replaceChildren(
                el('table', {},
                    el('tr', {}, el('th', { textContent: 'Audio' }), el('th', { textContent: 'Tipo' }), el('th', { textContent: 'Veces' })),
                    ...filas),
                el('div', { textContent: `Total de reproducciones: ${h.totalReproducciones}`, className: 'muted' }));
        } catch (e) {
            cont.replaceChildren(el('p', { textContent: `No se pudo cargar el historial: ${e.message}`, className: 'error' }));
        }
    }

    // ---------- arranque ----------
    function iniciar() {
        document.getElementById('usuario-id').textContent = estado.usuarioId;
        document.getElementById('btn-historial').onclick = verHistorial;
        window.__writeAudioLog = writeLog;
        crearBotonesReaccion();
        attachAudioListeners();
        cargarTipos();
        conectarWS();
        montarSuscripciones();
    }

    // ---------- suscripción y pago (módulo suscripciones.js) ----------
    function montarSuscripciones() {
        const panel = document.getElementById('panel-suscripciones');
        if (!window.Suscripciones) {
            panel.textContent = 'El módulo de suscripciones (suscripciones.js) no está disponible.';
            return;
        }
        panel.textContent = '';
        window.Suscripciones.montar({
            contenedor: '#panel-suscripciones',
            usuarioId: () => estado.usuarioId,
            apiBase: CFG.api,
            alEco: (eco) => {
                const nivel = /rechazad|fall/i.test(eco.mensaje) ? 'error'
                    : /exitoso|almacenado/i.test(eco.mensaje) ? 'success' : 'info';
                writeLog(`(${eco.origen}) ${eco.mensaje}`, nivel);
            }
        });
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', iniciar);
    } else {
        iniciar();
    }
})();