package scanner

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/DiegoModesto/berserker-player/server/internal/core"
	"github.com/DiegoModesto/berserker-player/server/internal/db"
)

// genMP3 sintetiza um mp3 curto com tags via ffmpeg; pula o teste se faltar ffmpeg.
func genMP3(t *testing.T, path, title, artist, album string) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg ausente; pulando teste de integração do scanner")
	}
	cmd := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-metadata", "title="+title, "-metadata", "artist="+artist, "-metadata", "album="+album,
		"-codec:a", "libmp3lame", "-q:a", "9", path)
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err := cmd.Run(); err != nil {
		t.Skipf("ffmpeg falhou ao gerar fixture (%v); pulando", err)
	}
}

func TestScanIntegration(t *testing.T) {
	tmp := t.TempDir()
	music := filepath.Join(tmp, "music")
	albumDir := filepath.Join(music, "Berserk OST")
	if err := os.MkdirAll(albumDir, 0o755); err != nil {
		t.Fatal(err)
	}
	genMP3(t, filepath.Join(albumDir, "01.mp3"), "Guts Theme", "Susumu Hirasawa", "Berserk OST")
	genMP3(t, filepath.Join(albumDir, "02.mp3"), "Forces", "Susumu Hirasawa", "Berserk OST")

	database, err := db.Open(filepath.Join(tmp, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := core.New(database)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	sc := New(store, music, "ffprobe", log)

	res, err := sc.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Scanned != 2 || res.Added != 2 {
		t.Fatalf("esperava 2 scanned/added, obtive %+v", res)
	}

	// Álbum e suas faixas indexados.
	page, err := store.ListAlbums("u", core.AlbumQuery{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 {
		t.Fatalf("esperava 1 álbum, obtive %d", page.Total)
	}
	album := page.Items[0]
	if album.Name != "Berserk OST" || album.SongCount != 2 {
		t.Fatalf("álbum inesperado: %+v", album)
	}
	songs, err := store.SongsByAlbum("u", album.ID)
	if err != nil || len(songs) != 2 {
		t.Fatalf("esperava 2 faixas, obtive %d (err=%v)", len(songs), err)
	}
	if songs[0].Duration <= 0 {
		t.Fatalf("ffprobe deveria preencher duração: %+v", songs[0])
	}

	// Busca FTS.
	sr, err := store.Search("u", "Guts", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(sr.Songs) == 0 {
		t.Fatal("busca por 'Guts' deveria retornar a faixa")
	}

	// Scan incremental: re-scan não deve adicionar nada.
	res2, _ := sc.Scan(context.Background())
	if res2.Added != 0 {
		t.Fatalf("re-scan não deveria adicionar; obtive %+v", res2)
	}
}
