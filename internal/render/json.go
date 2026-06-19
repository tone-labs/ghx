package render

import (
	"encoding/json"
	"io"
)

// JSON writes v as indented JSON. This is the stable machine contract a
// downstream consumer (a script or an automated agent) reads; bodies are always
// full here — truncation is a human-view concern only.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
