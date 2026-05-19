// app.js — wires the HTML controls to window.blockdev (provided by WASM).

(async function bootstrap() {
  const go = new Go();
  try {
    const result = await WebAssembly.instantiateStreaming(
      fetch("blockdev.wasm"),
      go.importObject
    );
    // run() returns when main() returns; our main() blocks on select{} forever,
    // so this never resolves. That's fine — JS keeps polling window.blockdev.
    go.run(result.instance);
  } catch (e) {
    document.getElementById("log").innerHTML =
      `<span class="err">Failed to load WASM: ${e.message}</span>`;
    return;
  }
  initUI();
})();

// readHistory: blockNum → { byte, source }
// Populated only by explicit Read button clicks. Cleared on init/discard.
// The read row in the state table reflects this map, NOT the device's
// derived read-result; that way the row visualizes "what the user has
// actually asked the device for", not "what each block would return if read."
const readHistory = new Map();

function initUI() {
  document.getElementById("btn-init").addEventListener("click", onInit);
  document.getElementById("btn-write").addEventListener("click", onWrite);
  document.getElementById("btn-read").addEventListener("click", onRead);
  document.getElementById("btn-snapshot").addEventListener("click", onSnapshot);
  document.getElementById("btn-discard").addEventListener("click", onDiscard);
  document.getElementById("btn-restore").addEventListener("click", onRestore);
  document.getElementById("btn-copy-hex").addEventListener("click", onCopyHex);

  document.getElementById("log").innerHTML = "";
  renderModuleStamp();
  log("ready", { ok: true }, "WASM loaded — click Initialize to create a device");
  render();
}

function renderModuleStamp() {
  const el = document.getElementById("module-stamp");
  if (!el) return;
  const v = window.blockdev.version();
  if (v.ok) {
    el.textContent = `calling ${v.module} ${v.version} — compiled to WebAssembly`;
  } else {
    el.textContent = "(module version unavailable)";
  }
}

// -- handlers ---------------------------------------------------------------

function onInit() {
  const blocks = clampInt(document.getElementById("blocks").value, 1, 64, 16);
  const r = window.blockdev.init(blocks);
  log(`init(${blocks})`, r, r.ok ? "base seeded: block N filled with byte('a'+N%26)" : null);
  if (r.ok) {
    document.getElementById("write-block").max = String(blocks - 1);
    document.getElementById("read-block").max = String(blocks - 1);
    document.getElementById("restore-hex").value = "";
    readHistory.clear();
  }
  render();
}

function setActionErr(id, msg) {
  document.getElementById(id).textContent = msg || "";
}

function onWrite() {
  setActionErr("write-err", "");
  const blockStr = document.getElementById("write-block").value.trim();
  if (blockStr === "") {
    setActionErr("write-err", "Enter a block number first.");
    log("writeBlock", { ok: false, error: "enter a block number first" });
    return;
  }
  const n = clampInt(blockStr, 0, 1023, -1);
  const charStr = document.getElementById("write-char").value;
  if (!charStr) {
    setActionErr("write-err", "Enter a character to write.");
    log("writeBlock", { ok: false, error: "enter a character" });
    return;
  }
  const ascii = charStr.charCodeAt(0);
  const r = window.blockdev.writeBlock(n, ascii);
  log(`writeBlock(${n}, '${charStr}' = ${ascii})`, r);
  if (r.ok) {
    document.getElementById("write-block").value = "";
    readHistory.delete(n);
  } else {
    setActionErr("write-err", r.error);
  }
  render();
}

function onRead() {
  setActionErr("read-err", "");
  const blockStr = document.getElementById("read-block").value.trim();
  if (blockStr === "") {
    setActionErr("read-err", "Enter a block number first.");
    log("readBlock", { ok: false, error: "enter a block number first" });
    return;
  }
  const n = clampInt(blockStr, 0, 1023, -1);
  const r = window.blockdev.readBlock(n);
  let extra = "";
  if (r.ok) {
    const char = (r.byte >= 32 && r.byte <= 126) ? String.fromCharCode(r.byte) : "·";
    extra = `byte=${r.byte} ('${char}'), source=${r.source}`;
    readHistory.set(n, { byte: r.byte, source: r.source });
    document.getElementById("read-block").value = "";
  } else {
    setActionErr("read-err", r.error);
  }
  log(`readBlock(${n})`, r, extra);
  render();
}

function onSnapshot() {
  const r = window.blockdev.snapshot();
  let extra = "";
  if (r.ok) {
    document.getElementById("restore-hex").value = r.blob;
    extra = formatBlobSummary(r.blob, r.size, r.entries);
  }
  log("snapshot()", r, extra);
}

function onDiscard() {
  const hexEl = document.getElementById("restore-hex");
  const hadHex = hexEl.value.trim().length > 0;
  const r = window.blockdev.discard();
  hexEl.value = "";
  readHistory.clear();
  let extra = "device released; base retained in memory; read history cleared";
  if (hadHex) {
    extra += "\nhex textarea also cleared — if you didn't copy it elsewhere, the overlay is unrecoverable";
  }
  log("discard()", r, extra);
  render();
}

function onRestore() {
  const hex = document.getElementById("restore-hex").value.trim();
  if (!hex) {
    log("restore()", { ok: false, error: "paste a hex blob first (or run Snapshot)" });
    return;
  }
  const r = window.blockdev.restore(hex);
  const short = hex.length > 32 ? hex.slice(0, 16) + "…" + hex.slice(-16) : hex;
  log(`restore(${short})`, r);
  if (r.ok) {
    document.getElementById("restore-hex").value = "";
  }
  render();
}

async function onCopyHex() {
  const hex = document.getElementById("restore-hex").value;
  if (!hex.trim()) {
    log("copy", { ok: false, error: "textarea is empty — run Snapshot first" });
    return;
  }
  try {
    await navigator.clipboard.writeText(hex);
    log("copy", { ok: true }, `${hex.length} hex chars (${hex.length / 2} bytes) copied to clipboard`);
  } catch (e) {
    log("copy", { ok: false, error: `clipboard write failed: ${e.message}` });
  }
}

// -- rendering --------------------------------------------------------------

function render() {
  const grid = document.getElementById("state-grid");
  const s = window.blockdev.state();

  // No base yet — page just loaded.
  if (!s.ok) {
    grid.innerHTML = `<div class="empty-state">
      No active device. Click <b>Initialize</b> above to create a base
      (seeded with letters a, b, c, …) and an empty overlay.
    </div>`;
    return;
  }

  // Base exists but device was discarded.
  if (s.discarded) {
    grid.innerHTML = `<div class="discarded-banner">
      ⚠ <b>Device discarded.</b> Overlay is gone. The base byte slice is still
      held in Go memory (BlockDevice retains it by reference, not copy) — paste a
      saved hex blob below and click Restore to bring the overlay back, or click
      Initialize to start fresh.
    </div>`;
    return;
  }

  // Active device — render the full state.
  const blocks = s.blocks;
  let html = '<table><thead><tr><th class="label">block #</th>';
  for (let i = 0; i < blocks; i++) html += `<th>${i}</th>`;
  html += "</tr></thead><tbody>";

  html += '<tr><td class="label">base</td>';
  for (let i = 0; i < blocks; i++) {
    html += renderCell(s.base[i], "base");
  }
  html += "</tr>";

  html += '<tr><td class="label">overlay</td>';
  for (let i = 0; i < blocks; i++) {
    if (s.overlay[i] === null || s.overlay[i] === undefined) {
      html += '<td class="empty">·</td>';
    } else {
      html += renderCell(s.overlay[i], "overlay");
    }
  }
  html += "</tr>";

  html += '<tr class="row-read"><td class="label">read</td>';
  for (let i = 0; i < blocks; i++) {
    if (readHistory.has(i)) {
      const { byte, source } = readHistory.get(i);
      const cls = source === "overlay" ? "overlay" : "base";
      html += renderCell(byte, cls);
    } else {
      html += '<td class="unread">?</td>';
    }
  }
  html += "</tr>";

  html += "</tbody></table>";
  html += '<div class="legend">' +
    'Blue = stored in <b>overlay</b> · Red = <b>read</b> served from overlay · ' +
    'Dim letter on read row = read served from base · ? = block not yet read · · = zero byte' +
    "</div>";
  grid.innerHTML = html;
}

function renderCell(byte, sourceClass) {
  if (byte === 0 || byte === null || byte === undefined) {
    return `<td class="${sourceClass} empty">·</td>`;
  }
  const c = (byte >= 32 && byte <= 126) ? String.fromCharCode(byte) : byte.toString(16).padStart(2, "0");
  return `<td class="${sourceClass}">${escape(c)}</td>`;
}

// -- helpers ----------------------------------------------------------------

// formatBlobSummary parses the hex blob and shows a digestible structure
// rather than dumping all 8208 hex chars per entry into the log.
function formatBlobSummary(blob, size, entries) {
  const lines = [`${size} bytes, ${entries} entr${entries === 1 ? "y" : "ies"} × 4104 bytes each`];
  const limit = Math.min(entries, 4); // show first 4 to keep log tidy
  for (let i = 0; i < limit; i++) {
    const off = i * 8208; // 4104 bytes × 2 hex chars per byte
    const blockHex = blob.substring(off, off + 16);
    const blockNum = parseInt(blockHex, 16);
    const firstByte = parseInt(blob.substring(off + 16, off + 18), 16);
    const ch = (firstByte >= 32 && firstByte <= 126)
      ? `'${String.fromCharCode(firstByte)}'`
      : `0x${firstByte.toString(16).padStart(2, "0")}`;
    lines.push(`[${i}] block#=${blockNum}, data=${ch}×4096`);
  }
  if (entries > limit) {
    lines.push(`(… ${entries - limit} more entries; full hex is in the textarea below)`);
  }
  lines.push("");
  lines.push("Each entry = 8 bytes (block#) + 4096 bytes (data) = 4104 bytes = 8208 hex chars.");
  return lines.join("\n");
}

// -- logging ----------------------------------------------------------------

function log(label, result, extra) {
  const line = document.createElement("div");
  line.className = "line";
  const status = result.ok
    ? `<span class="ok">✓</span>`
    : `<span class="err">✗ ${escape(result.error || "error")}</span>`;
  let html = `<span class="cmd">${escape(label)}</span> → ${status}`;
  if (extra) {
    html += `<span class="extra">${escape(extra)}</span>`;
  }
  line.innerHTML = html;
  const logEl = document.getElementById("log");
  logEl.appendChild(line);
  logEl.scrollTop = logEl.scrollHeight;
}

// -- utilities --------------------------------------------------------------

function clampInt(v, min, max, fallback) {
  const n = parseInt(v, 10);
  if (isNaN(n)) return fallback;
  return Math.max(min, Math.min(max, n));
}

function escape(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}
