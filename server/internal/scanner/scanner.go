// Package scanner percorre a biblioteca, extrai metadados e popula o índice.
package scanner

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DiegoModesto/berserker-player/server/internal/core"
	"github.com/dhowden/tag"
)

var audioExts = map[string]bool{
	".mp3": true, ".flac": true, ".ogg": true, ".oga": true, ".opus": true,
	".m4a": true, ".aac": true, ".wav": true, ".wma": true, ".alac": true,
}

var coverNames = []string{"cover.jpg", "cover.png", "cover.jpeg", "folder.jpg", "folder.png", "front.jpg"}

type Scanner struct {
	store       *core.Store
	musicFolder string
	ffprobe     string
	log         *slog.Logger
	mu          sync.Mutex
	scanning    bool
	lastResult  Result
}

type Result struct {
	Scanned    int       `json:"scanned"`
	Added      int       `json:"added"`
	Removed    int       `json:"removed"`
	Errors     int       `json:"errors"`
	Duration   string    `json:"duration"`
	FinishedAt time.Time `json:"finishedAt"`
}

func New(store *core.Store, musicFolder, ffprobe string, log *slog.Logger) *Scanner {
	return &Scanner{store: store, musicFolder: musicFolder, ffprobe: ffprobe, log: log}
}

func (s *Scanner) IsScanning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scanning
}

func (s *Scanner) LastResult() Result {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastResult
}

// parsed é o resultado da fase de parsing (concorrente) de um arquivo.
type parsed struct {
	path     string
	info     fileInfo
	coverDir string
	hasCover bool
	err      error
}

type fileInfo struct {
	title, album, artist, albumArtist, genre string
	year, track, disc                        int
	duration, bitRate, sampleRate            int
	size, mtime                              int64
	suffix                                   string
	embeddedCover                            bool
}

// Scan executa um scan completo (full) da biblioteca.
func (s *Scanner) Scan(ctx context.Context) (Result, error) {
	s.mu.Lock()
	if s.scanning {
		s.mu.Unlock()
		return Result{}, nil
	}
	s.scanning = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.scanning = false
		s.mu.Unlock()
	}()

	start := time.Now()
	var res Result

	existing, err := s.store.ExistingPaths()
	if err != nil {
		return res, err
	}
	seen := map[string]bool{}

	// Coleta de arquivos.
	var files []string
	_ = filepath.WalkDir(s.musicFolder, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if audioExts[strings.ToLower(filepath.Ext(path))] {
			files = append(files, path)
		}
		return nil
	})

	// Parsing concorrente.
	workers := runtime.NumCPU()
	if workers > 8 {
		workers = 8
	}
	jobs := make(chan string)
	results := make(chan parsed)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				results <- s.parseFile(p)
			}
		}()
	}
	go func() {
		for _, f := range files {
			select {
			case <-ctx.Done():
				close(jobs)
				return
			case jobs <- f:
			}
		}
		close(jobs)
	}()
	go func() { wg.Wait(); close(results) }()

	// Persistência serializada (single writer).
	coverDone := map[string]bool{}
	for r := range results {
		res.Scanned++
		if r.err != nil {
			res.Errors++
			s.log.Warn("falha ao processar arquivo", "path", r.path, "err", r.err)
			continue
		}
		seen[r.path] = true
		if mt, ok := existing[r.path]; ok && mt == r.info.mtime {
			continue // inalterado
		}
		if err := s.persist(r); err != nil {
			res.Errors++
			s.log.Warn("falha ao persistir", "path", r.path, "err", err)
			continue
		}
		if _, ok := existing[r.path]; !ok {
			res.Added++
		}
		if r.hasCover && !coverDone[r.coverDir] {
			coverDone[r.coverDir] = true
		}
	}

	// Remoção de órfãos.
	var orphans []string
	for p := range existing {
		if !seen[p] {
			orphans = append(orphans, p)
		}
	}
	if len(orphans) > 0 {
		if err := s.store.DeleteByPaths(orphans); err != nil {
			return res, err
		}
		res.Removed = len(orphans)
	}

	if err := s.store.CleanupEmpty(); err != nil {
		return res, err
	}
	if err := s.store.RecomputeCounts(); err != nil {
		return res, err
	}
	if err := s.store.RebuildSearchIndex(); err != nil {
		return res, err
	}

	res.Duration = time.Since(start).Round(time.Millisecond).String()
	res.FinishedAt = time.Now().UTC()
	s.mu.Lock()
	s.lastResult = res
	s.mu.Unlock()
	s.log.Info("scan concluído", "scanned", res.Scanned, "added", res.Added, "removed", res.Removed, "errors", res.Errors, "dur", res.Duration)
	return res, nil
}

func (s *Scanner) persist(r parsed) error {
	artistName := r.info.albumArtist
	if artistName == "" {
		artistName = r.info.artist
	}
	artistID, err := s.store.UpsertArtist(artistName)
	if err != nil {
		return err
	}
	albumID, err := s.store.UpsertAlbum(r.info.album, artistID, r.info.year, r.info.genre)
	if err != nil {
		return err
	}
	// Artista da faixa (pode diferir do album artist).
	trackArtistID := artistID
	if r.info.artist != "" && r.info.artist != artistName {
		if id, err := s.store.UpsertArtist(r.info.artist); err == nil {
			trackArtistID = id
		}
	}
	if _, err := s.store.UpsertMediaFile(core.MediaInput{
		Path: r.path, Title: r.info.title, AlbumID: albumID, ArtistID: trackArtistID,
		Track: r.info.track, Disc: r.info.disc, Duration: r.info.duration,
		BitRate: r.info.bitRate, SampleRate: r.info.sampleRate, Suffix: r.info.suffix,
		Size: r.info.size, MTime: r.info.mtime, EmbeddedCover: r.info.embeddedCover,
	}); err != nil {
		return err
	}
	if r.hasCover {
		for _, name := range coverNames {
			cp := filepath.Join(r.coverDir, name)
			if _, err := os.Stat(cp); err == nil {
				_ = s.store.SetAlbumCover(albumID, cp)
				break
			}
		}
	}
	return nil
}

func (s *Scanner) parseFile(path string) parsed {
	r := parsed{path: path, coverDir: filepath.Dir(path)}
	st, err := os.Stat(path)
	if err != nil {
		r.err = err
		return r
	}
	info := fileInfo{
		size:   st.Size(),
		mtime:  st.ModTime().Unix(),
		suffix: strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), "."),
		title:  strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	if f, err := os.Open(path); err == nil {
		if m, err := tag.ReadFrom(f); err == nil {
			if v := m.Title(); v != "" {
				info.title = v
			}
			info.album = m.Album()
			info.artist = m.Artist()
			info.albumArtist = m.AlbumArtist()
			info.genre = m.Genre()
			info.year = m.Year()
			info.track, _ = m.Track()
			info.disc, _ = m.Disc()
			info.embeddedCover = m.Picture() != nil
		}
		f.Close()
	}

	// Duração/bitrate/sample rate via ffprobe.
	if d, br, sr := s.probe(path); d > 0 {
		info.duration, info.bitRate, info.sampleRate = d, br, sr
	}

	r.info = info
	// Capa na pasta?
	for _, name := range coverNames {
		if _, err := os.Stat(filepath.Join(r.coverDir, name)); err == nil {
			r.hasCover = true
			break
		}
	}
	if info.embeddedCover {
		r.hasCover = true
	}
	return r
}

type ffprobeOut struct {
	Format struct {
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
	} `json:"format"`
	Streams []struct {
		CodecType  string `json:"codec_type"`
		SampleRate string `json:"sample_rate"`
	} `json:"streams"`
}

func (s *Scanner) probe(path string) (duration, bitRate, sampleRate int) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.ffprobe, "-v", "quiet", "-print_format", "json",
		"-show_format", "-show_streams", path)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, 0
	}
	var p ffprobeOut
	if err := json.Unmarshal(out, &p); err != nil {
		return 0, 0, 0
	}
	if f, err := strconv.ParseFloat(p.Format.Duration, 64); err == nil {
		duration = int(f + 0.5)
	}
	if n, err := strconv.Atoi(p.Format.BitRate); err == nil {
		bitRate = n / 1000
	}
	for _, st := range p.Streams {
		if st.CodecType == "audio" {
			if n, err := strconv.Atoi(st.SampleRate); err == nil {
				sampleRate = n
			}
			break
		}
	}
	return duration, bitRate, sampleRate
}
