package scanner

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watch monitora a biblioteca via fsnotify e dispara scans incrementais
// (com debounce) quando arquivos mudam. Bloqueia até o contexto ser cancelado.
func (s *Scanner) Watch(ctx context.Context, debounce time.Duration) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	addRecursive(w, s.musicFolder)

	if debounce <= 0 {
		debounce = 2 * time.Second
	}
	var timer *time.Timer
	trigger := func() {
		if _, err := s.Scan(context.Background()); err != nil {
			s.log.Error("scan (watcher) falhou", "err", err)
		}
	}

	s.log.Info("watcher ativo", "music", s.musicFolder)
	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-w.Events:
			if !ok {
				return nil
			}
			// Novos diretórios passam a ser monitorados.
			if event.Op&fsnotify.Create != 0 {
				if fi, err := os.Stat(event.Name); err == nil && fi.IsDir() {
					addRecursive(w, event.Name)
				}
			}
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounce, trigger)
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			s.log.Warn("watcher erro", "err", err)
		}
	}
}

func addRecursive(w *fsnotify.Watcher, root string) {
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			_ = w.Add(path)
		}
		return nil
	})
}
