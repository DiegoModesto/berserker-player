package nativeapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/DiegoModesto/berserker-player/server/internal/model"
)

// seedLibrary gera um álbum com 2 faixas via ffmpeg e escaneia. Pula sem ffmpeg.
func seedLibrary(t *testing.T, srv *Server) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg ausente")
	}
	dir := filepath.Join(srv.cfg.MusicFolder, "Album X")
	_ = os.MkdirAll(dir, 0o755)
	// Capa na pasta (cobre artwork + endpoint /cover).
	_ = exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "color=c=red:s=64x64",
		"-frames:v", "1", filepath.Join(dir, "cover.jpg")).Run()
	for i, title := range []string{"One", "Two"} {
		f := filepath.Join(dir, []string{"01.mp3", "02.mp3"}[i])
		cmd := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "sine=frequency=440:duration=1",
			"-metadata", "title="+title, "-metadata", "artist=Tester", "-metadata", "album=Album X",
			"-codec:a", "libmp3lame", "-q:a", "9", f)
		if err := cmd.Run(); err != nil {
			t.Skipf("ffmpeg falhou: %v", err)
		}
	}
	if _, err := srv.scanner.Scan(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func authGet(t *testing.T, srv *Server, token, path string, out any) int {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if out != nil && rr.Code == http.StatusOK {
		_ = json.Unmarshal(rr.Body.Bytes(), out)
	}
	return rr.Code
}

func authSend(t *testing.T, srv *Server, method, token, path string, body any, out any) int {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if out != nil && (rr.Code == http.StatusOK || rr.Code == http.StatusCreated) {
		_ = json.Unmarshal(rr.Body.Bytes(), out)
	}
	return rr.Code
}

func TestLibraryFlow(t *testing.T) {
	srv, _ := newTestServer(t)
	seedLibrary(t, srv)
	token := login(t, srv, "admin", "pw")

	// Álbuns
	var albums model.Page[model.Album]
	if code := authGet(t, srv, token, "/api/v1/albums", &albums); code != 200 {
		t.Fatalf("albums %d", code)
	}
	if albums.Total != 1 || len(albums.Items) != 1 {
		t.Fatalf("esperava 1 álbum: %+v", albums)
	}
	albumID := albums.Items[0].ID

	// Detalhe do álbum com faixas
	var detail struct {
		model.Album
		Songs []model.Song `json:"songs"`
	}
	if code := authGet(t, srv, token, "/api/v1/albums/"+albumID, &detail); code != 200 {
		t.Fatalf("album detail %d", code)
	}
	if len(detail.Songs) != 2 {
		t.Fatalf("esperava 2 faixas: %d", len(detail.Songs))
	}
	songID := detail.Songs[0].ID

	// Busca
	var sr struct {
		Songs []model.Song `json:"songs"`
	}
	if code := authGet(t, srv, token, "/api/v1/search?q=One", &sr); code != 200 {
		t.Fatalf("search %d", code)
	}
	if len(sr.Songs) == 0 {
		t.Fatal("busca deveria achar 'One'")
	}

	// Favoritar a faixa e checar reflexo no GET
	if code := authSend(t, srv, "POST", token, "/api/v1/star", map[string]string{"id": songID, "type": "song"}, nil); code != 204 {
		t.Fatalf("star %d", code)
	}
	var song model.Song
	authGet(t, srv, token, "/api/v1/songs/"+songID, &song)
	if !song.Starred {
		t.Fatal("faixa deveria estar favoritada")
	}

	// Rating
	if code := authSend(t, srv, "POST", token, "/api/v1/rating", map[string]any{"id": songID, "type": "song", "rating": 4}, nil); code != 204 {
		t.Fatalf("rating %d", code)
	}

	// Scrobble (submission)
	if code := authSend(t, srv, "POST", token, "/api/v1/scrobble", map[string]any{"songId": songID, "event": "submission"}, nil); code != 204 {
		t.Fatalf("scrobble %d", code)
	}
	authGet(t, srv, token, "/api/v1/songs/"+songID, &song)
	if song.Rating != 4 || song.PlayCount != 1 {
		t.Fatalf("rating/playcount inesperados: %+v", song)
	}
}

func TestSmartPlaylistAPI(t *testing.T) {
	srv, _ := newTestServer(t)
	seedLibrary(t, srv)
	token := login(t, srv, "admin", "pw")

	var pl model.Playlist
	code := authSend(t, srv, "POST", token, "/api/v1/playlists/smart",
		map[string]any{"name": "Tudo", "rules": map[string]any{"sort": "title", "limit": 10}}, &pl)
	if code != 201 {
		t.Fatalf("create smart %d", code)
	}
	var detail struct {
		Songs []model.Song `json:"songs"`
	}
	if code := authGet(t, srv, token, "/api/v1/playlists/"+pl.ID, &detail); code != 200 {
		t.Fatalf("get smart %d", code)
	}
	if len(detail.Songs) != 2 {
		t.Fatalf("smart playlist deveria avaliar 2 faixas, %d", len(detail.Songs))
	}
}

func TestPlaylistCRUD(t *testing.T) {
	srv, _ := newTestServer(t)
	seedLibrary(t, srv)
	token := login(t, srv, "admin", "pw")

	var albums model.Page[model.Album]
	authGet(t, srv, token, "/api/v1/albums", &albums)
	var detail struct {
		Songs []model.Song `json:"songs"`
	}
	authGet(t, srv, token, "/api/v1/albums/"+albums.Items[0].ID, &detail)
	ids := []string{detail.Songs[0].ID, detail.Songs[1].ID}

	// Criar
	var pl model.Playlist
	if code := authSend(t, srv, "POST", token, "/api/v1/playlists",
		map[string]any{"name": "Favoritas", "songIds": ids}, &pl); code != 201 {
		t.Fatalf("create %d", code)
	}
	if pl.SongCount != 2 {
		t.Fatalf("esperava 2 faixas: %+v", pl)
	}

	// Reordenar (inverter)
	var updated model.Playlist
	if code := authSend(t, srv, "PUT", token, "/api/v1/playlists/"+pl.ID,
		map[string]any{"songIds": []string{ids[1], ids[0]}}, &updated); code != 200 {
		t.Fatalf("update %d", code)
	}
	var got struct {
		Songs []model.Song `json:"songs"`
	}
	authGet(t, srv, token, "/api/v1/playlists/"+pl.ID, &got)
	if got.Songs[0].ID != ids[1] {
		t.Fatal("ordem não foi atualizada")
	}

	// Deletar
	if code := authSend(t, srv, "DELETE", token, "/api/v1/playlists/"+pl.ID, nil, nil); code != 204 {
		t.Fatalf("delete %d", code)
	}
	if code := authGet(t, srv, token, "/api/v1/playlists/"+pl.ID, nil); code != 404 {
		t.Fatalf("esperava 404 após delete, obtive %d", code)
	}
}
