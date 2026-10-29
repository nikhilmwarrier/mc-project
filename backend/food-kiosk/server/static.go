package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// mountSPA serves the built frontend at the root. Unknown paths fall back to
// index.html so client-side routes like /admin work on a hard refresh. API
// patterns are more specific, so they always win over this catch-all.
func mountSPA(mux *http.ServeMux, dir string) {
	if dir == "" {
		return
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return
	}

	index := filepath.Join(dir, "index.html")
	fs := http.FileServer(http.Dir(dir))

	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		clean := filepath.Clean("/" + r.URL.Path)
		file := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(clean, "/")))
		if filepath.Base(clean) == "manifest.webmanifest" {
			w.Header().Set("Content-Type", "application/manifest+json")
		}
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, index)
	}))
}
