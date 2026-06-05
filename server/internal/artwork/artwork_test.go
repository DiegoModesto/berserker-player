package artwork

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/DiegoModesto/berserker-player/server/internal/db"
)

func makeJPEG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 100, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	if path != "" {
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCoverFromFileAndResize(t *testing.T) {
	tmp := t.TempDir()
	database, err := db.Open(filepath.Join(tmp, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	coverPath := filepath.Join(tmp, "cover.jpg")
	makeJPEG(t, coverPath, 400, 300)

	// Insere artista + álbum com cover_path.
	_, _ = database.Exec(`INSERT INTO artists(id,name) VALUES('a','Art')`)
	_, _ = database.Exec(`INSERT INTO albums(id,name,artist_id,cover_path,created_at) VALUES('al','Alb','a',?, '2024-01-01T00:00:00Z')`, coverPath)

	r := New(database, filepath.Join(tmp, "cache"))

	// Original.
	data, ct, err := r.Cover("al", 0)
	if err != nil || len(data) == 0 || ct != "image/jpeg" {
		t.Fatalf("Cover original: ct=%s len=%d err=%v", ct, len(data), err)
	}
	// Redimensionado (gera cache).
	small, ct2, err := r.Cover("al", 100)
	if err != nil || ct2 != "image/jpeg" {
		t.Fatalf("Cover resize: %v", err)
	}
	img, _, err := image.Decode(bytes.NewReader(small))
	if err != nil {
		t.Fatalf("imagem redimensionada inválida: %v", err)
	}
	if img.Bounds().Dx() != 100 {
		t.Fatalf("largura esperada 100, %d", img.Bounds().Dx())
	}
	// Segunda chamada vem do cache (mesmo tamanho).
	cached, _, err := r.Cover("al", 100)
	if err != nil || len(cached) != len(small) {
		t.Fatalf("cache divergente: %d vs %d", len(cached), len(small))
	}
}

func TestCoverNoCover(t *testing.T) {
	tmp := t.TempDir()
	database, _ := db.Open(filepath.Join(tmp, "t.db"))
	defer database.Close()
	r := New(database, filepath.Join(tmp, "cache"))
	if _, _, err := r.Cover("inexistente", 0); err != ErrNoCover {
		t.Fatalf("esperava ErrNoCover, %v", err)
	}
}

func TestCoverEmbedded(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg ausente")
	}
	tmp := t.TempDir()
	mp3 := filepath.Join(tmp, "song.mp3")
	// Áudio com capa embutida (APIC).
	cmd := exec.Command("ffmpeg", "-y",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-f", "lavfi", "-i", "color=c=blue:s=48x48",
		"-map", "0:a", "-map", "1:v", "-frames:v", "1",
		"-id3v2_version", "3", "-disposition:v", "attached_pic", mp3)
	if err := cmd.Run(); err != nil {
		t.Skipf("ffmpeg embed falhou: %v", err)
	}

	database, _ := db.Open(filepath.Join(tmp, "t.db"))
	defer database.Close()
	_, _ = database.Exec(`INSERT INTO artists(id,name) VALUES('a','Art')`)
	_, _ = database.Exec(`INSERT INTO albums(id,name,artist_id,cover_path,created_at) VALUES('al','Alb','a','', '2024-01-01T00:00:00Z')`)
	_, _ = database.Exec(`INSERT INTO media_files(id,path,title,album_id,artist_id,has_embedded_cover,created_at) VALUES('m',?, 'T','al','a',1,'2024-01-01T00:00:00Z')`, mp3)

	r := New(database, filepath.Join(tmp, "cache"))
	data, _, err := r.Cover("al", 0)
	if err != nil || len(data) == 0 {
		t.Fatalf("capa embutida não resolvida: len=%d err=%v", len(data), err)
	}
}

func TestItoa(t *testing.T) {
	cases := map[int]string{0: "0", 5: "5", 100: "100", 4533: "4533"}
	for in, want := range cases {
		if got := itoa(in); got != want {
			t.Fatalf("itoa(%d)=%q want %q", in, got, want)
		}
	}
}
