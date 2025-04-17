package webui

import (
	_ "embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed public.zip
var static []byte

func StartServer(addr string) error {
	fsys, err := fs.Sub(UnzipToFS(static), ".")
	if err != nil {
		return err
	}

	// Create file server handler
	fss := http.FS(fsys)
	fileServer := http.FileServer(fss)

	// Create main handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFileFS(w, r, fsys, "index.html")
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	// Start the server
	log.Printf("Starting server on %s", addr)
	return http.ListenAndServe(addr, nil)
}
