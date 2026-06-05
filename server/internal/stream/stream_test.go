package stream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSupportedFormat(t *testing.T) {
	for _, f := range []string{"mp3", "opus", "aac"} {
		if !SupportedFormat(f) {
			t.Fatalf("%s deveria ser suportado", f)
		}
	}
	if SupportedFormat("raw") || SupportedFormat("flac") {
		t.Fatal("formato não suportado retornou true")
	}
}

func TestServeFileRange(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "a.mp3")
	content := make([]byte, 5000)
	for i := range content {
		content[i] = byte(i % 251)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	// Sem Range → 200 + Accept-Ranges.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)
	if err := ServeFile(rr, req, path); err != nil {
		t.Fatal(err)
	}
	if rr.Code != http.StatusOK || rr.Header().Get("Accept-Ranges") != "bytes" {
		t.Fatalf("esperava 200 + Accept-Ranges, code=%d", rr.Code)
	}
	if rr.Header().Get("Content-Type") != "audio/mpeg" {
		t.Fatalf("content-type errado: %s", rr.Header().Get("Content-Type"))
	}

	// Com Range → 206 parcial.
	rr = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/stream", nil)
	req.Header.Set("Range", "bytes=0-1023")
	_ = ServeFile(rr, req, path)
	if rr.Code != http.StatusPartialContent {
		t.Fatalf("esperava 206, code=%d", rr.Code)
	}
	if rr.Body.Len() != 1024 {
		t.Fatalf("esperava 1024 bytes, %d", rr.Body.Len())
	}
}

func TestModTime(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "x.mp3")
	_ = os.WriteFile(path, []byte("data"), 0o644)
	if ModTime(path).IsZero() {
		t.Fatal("ModTime de arquivo existente não deveria ser zero")
	}
	if !ModTime("/nao/existe").IsZero() {
		t.Fatal("ModTime de arquivo inexistente deveria ser zero")
	}
}

func TestServeFileMissing(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/stream", nil)
	if err := ServeFile(rr, req, "/nao/existe.mp3"); err == nil {
		t.Fatal("esperava erro para arquivo inexistente")
	}
}

func TestTranscode(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg ausente")
	}
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.mp3")
	if err := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-codec:a", "libmp3lame", src).Run(); err != nil {
		t.Skipf("ffmpeg gen falhou: %v", err)
	}
	tr := NewTranscoder("ffmpeg")
	rr := httptest.NewRecorder()
	if err := tr.Stream(context.Background(), rr, src, "opus", 96, 0); err != nil {
		t.Fatalf("transcode: %v", err)
	}
	if rr.Body.Len() == 0 || rr.Header().Get("Content-Type") != "audio/ogg" {
		t.Fatalf("saída transcode inválida: len=%d ct=%s", rr.Body.Len(), rr.Header().Get("Content-Type"))
	}

	// Formato não suportado.
	if err := tr.Stream(context.Background(), httptest.NewRecorder(), src, "flac", 0, 0); err == nil {
		t.Fatal("esperava erro para formato não suportado")
	}
}
