// Package stream serve áudio: direct play com suporte a Range (HTTP 206).
// A transcodificação sob demanda é adicionada na fase avançada.
package stream

import (
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var contentTypes = map[string]string{
	".mp3":  "audio/mpeg",
	".flac": "audio/flac",
	".ogg":  "audio/ogg",
	".oga":  "audio/ogg",
	".opus": "audio/opus",
	".m4a":  "audio/mp4",
	".aac":  "audio/aac",
	".wav":  "audio/wav",
	".wma":  "audio/x-ms-wma",
}

// ServeFile faz direct play do arquivo, delegando o tratamento de Range ao
// http.ServeContent (responde 200 sem Range ou 206 Partial Content com Range).
func ServeFile(w http.ResponseWriter, r *http.Request, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	if ct, ok := contentTypes[filepath.Ext(path)]; ok {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, filepath.Base(path), st.ModTime(), f)
	return nil
}

// ModTime devolve o mtime do arquivo (helper para cache de artwork).
func ModTime(path string) time.Time {
	if st, err := os.Stat(path); err == nil {
		return st.ModTime()
	}
	return time.Time{}
}
