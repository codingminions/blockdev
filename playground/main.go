// Playground WASM wrapper. Exposes the blockdev API as JS-callable functions
// so a static HTML page can drive write/read/snapshot/restore from the browser.
//
// Build:
//
//	GOOS=js GOARCH=wasm go build -o blockdev.wasm .
//
// Then serve the HTML page (alongside wasm_exec.js copied from
// `$(go env GOROOT)/lib/wasm/wasm_exec.js`) over any static HTTP server.
//
// JS API attached to window.blockdev (all functions return {ok, error?, ...}):
//
//	init(blocks)            create a fresh device on a base of `blocks` × 4 KB
//	writeBlock(n, ascii)    fill block n with `ascii` × 4096 bytes
//	readBlock(n)            read first byte of block n; report source layer
//	snapshot()              serialize the overlay; return hex
//	discard()               drop *BlockDevice; base retained
//	restore(blobHex)        Deserialize(blob, base); rebuild overlay
//	state()                 per-block view of base/overlay/read for the UI
package main

import (
	"encoding/binary"
	"encoding/hex"
	"syscall/js"

	"github.com/codingminions/blockdev"
)

// Singleton state for the playground tab.
var (
	bd           *blockdev.BlockDevice
	base         []byte
	overlayState map[int]byte // block# -> first byte (tracked here for fast state())
)

func main() {
	js.Global().Set("blockdev", js.ValueOf(map[string]any{
		"init":       js.FuncOf(jsInit),
		"writeBlock": js.FuncOf(jsWriteBlock),
		"readBlock":  js.FuncOf(jsReadBlock),
		"snapshot":   js.FuncOf(jsSnapshot),
		"discard":    js.FuncOf(jsDiscard),
		"restore":    js.FuncOf(jsRestore),
		"state":      js.FuncOf(jsState),
	}))
	select {} // keep the Go runtime alive while JS calls in
}

func okResult(extras ...map[string]any) any {
	m := map[string]any{"ok": true}
	for _, e := range extras {
		for k, v := range e {
			m[k] = v
		}
	}
	return m
}

func errResult(msg string) any {
	return map[string]any{"ok": false, "error": msg}
}

// init(blocks) — fresh device. Base is seeded with byte('a' + N % 26) per
// block N so the playground UI can show a recognizable base distinct from
// overlay writes (which use uppercase letters by default).
func jsInit(this js.Value, args []js.Value) any {
	if len(args) != 1 {
		return errResult("init expects 1 argument (blocks)")
	}
	blocks := args[0].Int()
	if blocks <= 0 || blocks > 1024 {
		return errResult("blocks must be in range [1, 1024]")
	}
	base = make([]byte, blocks*blockdev.BlockSize)
	for i := 0; i < blocks; i++ {
		fill := byte('a' + (i % 26))
		for j := 0; j < blockdev.BlockSize; j++ {
			base[i*blockdev.BlockSize+j] = fill
		}
	}
	newBd, err := blockdev.New(base)
	if err != nil {
		return errResult(err.Error())
	}
	bd = newBd
	overlayState = make(map[int]byte)
	return okResult(map[string]any{
		"blocks":    blocks,
		"blockSize": blockdev.BlockSize,
	})
}

func jsWriteBlock(this js.Value, args []js.Value) any {
	if bd == nil {
		return errResult("call init() first")
	}
	if len(args) != 2 {
		return errResult("writeBlock expects 2 arguments (blockNum, ascii)")
	}
	blockNum := args[0].Int()
	ascii := args[1].Int()
	if ascii < 0 || ascii > 255 {
		return errResult("ascii must be 0-255")
	}
	off := int64(blockNum) * blockdev.BlockSize
	data := make([]byte, blockdev.BlockSize)
	for i := range data {
		data[i] = byte(ascii)
	}
	if _, err := bd.WriteAt(data, off); err != nil {
		return errResult(err.Error())
	}
	overlayState[blockNum] = byte(ascii)
	return okResult()
}

func jsReadBlock(this js.Value, args []js.Value) any {
	if bd == nil {
		return errResult("call init() first")
	}
	if len(args) != 1 {
		return errResult("readBlock expects 1 argument (blockNum)")
	}
	blockNum := args[0].Int()
	off := int64(blockNum) * blockdev.BlockSize
	p := make([]byte, blockdev.BlockSize)
	if _, err := bd.ReadAt(p, off); err != nil {
		return errResult(err.Error())
	}
	source := "base"
	if _, ok := overlayState[blockNum]; ok {
		source = "overlay"
	}
	return okResult(map[string]any{
		"byte":   int(p[0]),
		"source": source,
	})
}

func jsSnapshot(this js.Value, args []js.Value) any {
	if bd == nil {
		return errResult("call init() first")
	}
	blob := bd.Serialize()
	return okResult(map[string]any{
		"blob":    hex.EncodeToString(blob),
		"size":    len(blob),
		"entries": len(blob) / blockdev.EntrySize,
	})
}

// discard() — release the device and its overlay tracking. Base is retained
// so restore() can rebuild from a snapshot blob.
func jsDiscard(this js.Value, args []js.Value) any {
	bd = nil
	overlayState = nil
	return okResult()
}

func jsRestore(this js.Value, args []js.Value) any {
	if base == nil {
		return errResult("call init() first; base is needed for restore")
	}
	if len(args) != 1 {
		return errResult("restore expects 1 argument (blobHex)")
	}
	blob, err := hex.DecodeString(args[0].String())
	if err != nil {
		return errResult("invalid hex: " + err.Error())
	}
	newBd, err := blockdev.Deserialize(blob, base)
	if err != nil {
		return errResult(err.Error())
	}
	bd = newBd
	overlayState = make(map[int]byte)
	for i := 0; i < len(blob); i += blockdev.EntrySize {
		blockNum := int(binary.BigEndian.Uint64(blob[i : i+8]))
		overlayState[blockNum] = blob[i+8]
	}
	return okResult()
}

// state() — works even when bd is nil (after discard), so the UI can show a
// "device discarded" banner while still proving the base survived in Go memory.
func jsState(this js.Value, args []js.Value) any {
	if base == nil {
		return errResult("call init() first")
	}
	discarded := bd == nil
	blocks := len(base) / blockdev.BlockSize
	baseBytes := make([]any, blocks)
	overlayBytes := make([]any, blocks)
	readBytes := make([]any, blocks)
	for i := 0; i < blocks; i++ {
		baseBytes[i] = int(base[i*blockdev.BlockSize])
		switch {
		case discarded:
			overlayBytes[i] = nil
			readBytes[i] = nil
		default:
			if b, ok := overlayState[i]; ok {
				overlayBytes[i] = int(b)
				readBytes[i] = int(b)
			} else {
				overlayBytes[i] = nil
				readBytes[i] = int(base[i*blockdev.BlockSize])
			}
		}
	}
	return okResult(map[string]any{
		"blocks":    blocks,
		"base":      baseBytes,
		"overlay":   overlayBytes,
		"read":      readBytes,
		"discarded": discarded,
	})
}
