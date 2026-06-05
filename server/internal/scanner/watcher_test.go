package scanner

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/DiegoModesto/berserker-player/server/internal/core"
	"github.com/DiegoModesto/berserker-player/server/internal/db"
)

func TestWatcherTriggersScan(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg ausente")
	}
	tmp := t.TempDir()
	music := filepath.Join(tmp, "music")
	_ = os.MkdirAll(music, 0o755)

	database, err := db.Open(filepath.Join(tmp, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := core.New(database)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	sc := New(store, music, "ffprobe", log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = sc.Watch(ctx, 200*time.Millisecond) }()
	time.Sleep(150 * time.Millisecond) // deixa o watcher registrar

	// Cria um arquivo de áudio → deve disparar scan.
	out := filepath.Join(music, "new.mp3")
	if err := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-metadata", "title=New", "-metadata", "album=W", "-codec:a", "libmp3lame", out).Run(); err != nil {
		t.Skipf("ffmpeg falhou: %v", err)
	}

	// Aguarda indexação (debounce + scan).
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		page, _ := store.ListAlbums("u", core.AlbumQuery{Limit: 10})
		if page.Total >= 1 {
			return // sucesso
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("watcher não indexou o novo arquivo a tempo")
}
