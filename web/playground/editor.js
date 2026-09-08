// editor.js wires a plain <textarea>/<select>/<iframe>/status-list page to
// the eventModelingRender / eventModelingFormat globals cmd/wasm exposes.
// It knows nothing about WHERE it is hosted — only the DOM element ids
// below — so this exact file is copied verbatim into the website's
// playground (see em-hcl-spec's Makefile `build-wasm` target and the
// website's deploy workflow) rather than reimplemented there. Treat any
// change to the DOM contract (the ids referenced here) or to the WASM
// function names/return shape as a breaking change to both call sites.
//
// Expected host markup:
//   <textarea id="editor">...</textarea>
//   <select id="profile"><option value="workshop">...<option value="valid">...<option value="strict">...</select>
//   <iframe id="preview"></iframe>
//   <ul id="diagnostics"></ul>
//   <button id="format-btn">Format</button>
//   <span id="status"></span>
(function () {
  "use strict";

  var SEED = typeof window.EVENT_MODELING_SEED === 'string' ? window.EVENT_MODELING_SEED : '';

  var DEBOUNCE_MS = 300;

  var editorEl, profileEl, previewEl, diagnosticsEl, formatBtnEl, statusEl;
  var debounceHandle = null;
  var hasRendered = false;

  function setStatus(text) {
    if (statusEl) statusEl.textContent = text;
  }

  function renderDiagnostics(diagnostics) {
    diagnosticsEl.innerHTML = '';
    if (!diagnostics || diagnostics.length === 0) {
      var empty = document.createElement('li');
      empty.className = 'diagnostic-empty';
      empty.textContent = 'No diagnostics.';
      diagnosticsEl.appendChild(empty);
      return;
    }
    diagnostics.forEach(function (diagnostic) {
      var item = document.createElement('li');
      item.className = 'diagnostic diagnostic-' + String(diagnostic.severity || '').toLowerCase();

      var head = document.createElement('code');
      head.textContent = (diagnostic.severity || '') + ' ' + (diagnostic.code || '');
      item.appendChild(head);

      var summary = document.createTextNode(': ' + (diagnostic.summary || ''));
      item.appendChild(summary);

      if (diagnostic.detail) {
        item.appendChild(document.createTextNode(' — ' + diagnostic.detail));
      }
      if (diagnostic.line) {
        var where = document.createElement('span');
        where.className = 'diagnostic-location';
        where.textContent = ' (line ' + diagnostic.line + ', column ' + diagnostic.column + ')';
        item.appendChild(where);
      }
      diagnosticsEl.appendChild(item);
    });
  }

  // renderNow calls into WASM immediately (no debounce) — used both by the
  // debounced input handler and directly after Format, so a Format that
  // changes the source is reflected in the preview without waiting.
  function renderNow() {
    if (typeof window.eventModelingRender !== 'function') return;
    var result = window.eventModelingRender(editorEl.value, profileEl.value);
    if (result && result.error) {
      setStatus('Error: ' + result.error);
      return;
    }
    renderDiagnostics(result.diagnostics);
    if (result.html) {
      previewEl.srcdoc = result.html;
      hasRendered = true;
      setStatus('Rendered.');
    } else {
      if (!hasRendered) {
        previewEl.srcdoc =
          '<!doctype html><meta charset="utf-8"><body style="font:14px system-ui;padding:16px;color:#a33">' +
          'Model has errors — see diagnostics.</body>';
      }
      setStatus(hasRendered ? 'Model has errors; showing last valid diagram.' : 'Model has errors.');
    }
  }

  function scheduleRender() {
    if (debounceHandle) clearTimeout(debounceHandle);
    debounceHandle = setTimeout(renderNow, DEBOUNCE_MS);
  }

  function formatNow() {
    if (typeof window.eventModelingFormat !== 'function') return;
    var result = window.eventModelingFormat(editorEl.value);
    if (result && result.error) {
      setStatus('Error: ' + result.error);
      return;
    }
    if (result.diagnostics && result.diagnostics.length > 0) {
      renderDiagnostics(result.diagnostics);
      setStatus('Could not format — fix the errors below first.');
      return;
    }
    editorEl.value = result.source;
    renderNow();
  }

  // ready is invoked by cmd/wasm once the module has finished initializing
  // (window.onEventModelingReady, wired below, before the module loads).
  function ready() {
    setStatus('Ready.');
    editorEl.disabled = false;
    formatBtnEl.disabled = false;
    renderNow();
  }

  // init runs synchronously as soon as this file is parsed — deliberately
  // NOT deferred to DOMContentLoaded. cmd/wasm calls window.onEventModelingReady
  // (set here) as soon as the module finishes loading, which can happen
  // before DOMContentLoaded would otherwise fire; the callback must exist
  // by then. This is safe only because the host page includes this script
  // via <script src="editor.js"> placed AFTER the markup below in the
  // document (so the elements already exist) and BEFORE the wasm-loading
  // script (so this callback is registered before the module can call it).
  // Both host pages (this one and the website's /playground/) must keep
  // that ordering.
  function init() {
    editorEl = document.getElementById('editor');
    profileEl = document.getElementById('profile');
    previewEl = document.getElementById('preview');
    diagnosticsEl = document.getElementById('diagnostics');
    formatBtnEl = document.getElementById('format-btn');
    statusEl = document.getElementById('status');

    editorEl.value = SEED;
    editorEl.disabled = true;
    formatBtnEl.disabled = true;

    editorEl.addEventListener('input', scheduleRender);
    profileEl.addEventListener('change', renderNow);
    formatBtnEl.addEventListener('click', formatNow);

    window.onEventModelingReady = ready;
  }

  init();
})();
