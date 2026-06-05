package nativeapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DiegoModesto/berserker-player/server/internal/model"
)

func stringReader(s string) io.Reader { return strings.NewReader(s) }

func mediaToken(t *testing.T, srv *Server, access string) string {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/v1/auth/media-token", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	var out struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	return out.Token
}

func raw(t *testing.T, srv *Server, method, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	return rr
}

func TestAllNativeEndpoints(t *testing.T) {
	srv, _ := newTestServer(t)
	seedLibrary(t, srv)
	token := login(t, srv, "admin", "pw")

	// Artists list + detail.
	var artists model.Page[model.Artist]
	if authGet(t, srv, token, "/api/v1/artists?sort=name&order=asc", &artists) != 200 || artists.Total == 0 {
		t.Fatal("artists list")
	}
	if authGet(t, srv, token, "/api/v1/artists/"+artists.Items[0].ID, nil) != 200 {
		t.Fatal("artist detail")
	}
	if authGet(t, srv, token, "/api/v1/artists/inexistente", nil) != 404 {
		t.Fatal("artist 404 esperado")
	}

	// Album detail → pega song id.
	var albums model.Page[model.Album]
	authGet(t, srv, token, "/api/v1/albums?sort=year&order=desc", &albums)
	var detail struct {
		Songs []model.Song `json:"songs"`
	}
	authGet(t, srv, token, "/api/v1/albums/"+albums.Items[0].ID, &detail)
	songID := detail.Songs[0].ID
	albumID := albums.Items[0].ID

	if authGet(t, srv, token, "/api/v1/songs/"+songID, nil) != 200 {
		t.Fatal("song detail")
	}

	// Rating + unstar.
	if authSend(t, srv, "POST", token, "/api/v1/rating", map[string]any{"id": songID, "type": "song", "rating": 3}, nil) != 204 {
		t.Fatal("rating")
	}
	if authSend(t, srv, "POST", token, "/api/v1/star", map[string]any{"id": songID, "type": "song"}, nil) != 204 {
		t.Fatal("star")
	}
	if authSend(t, srv, "DELETE", token, "/api/v1/star", map[string]any{"id": songID, "type": "song"}, nil) != 204 {
		t.Fatal("unstar")
	}
	if authSend(t, srv, "POST", token, "/api/v1/scrobble", map[string]any{"songId": songID, "event": "nowplaying"}, nil) != 204 {
		t.Fatal("scrobble nowplaying")
	}

	// Playlists list (vazio).
	if authGet(t, srv, token, "/api/v1/playlists", nil) != 200 {
		t.Fatal("playlists list")
	}

	// Admin scan + status.
	if raw(t, srv, "POST", "/api/v1/admin/scan", token).Code != http.StatusAccepted {
		t.Fatal("admin scan")
	}
	if authGet(t, srv, token, "/api/v1/admin/scan/status", nil) != 200 {
		t.Fatal("scan status")
	}

	// OpenAPI público.
	if raw(t, srv, "GET", "/api/v1/openapi.yaml", "").Code == 0 {
		t.Fatal("openapi")
	}

	// Mídia: media token → stream (direct + transcode) e cover.
	mt := mediaToken(t, srv, token)
	if mt == "" {
		t.Fatal("media token vazio")
	}
	// Direct play (Range).
	req := httptest.NewRequest("GET", "/api/v1/stream/"+songID+"?token="+mt, nil)
	req.Header.Set("Range", "bytes=0-99")
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusPartialContent {
		t.Fatalf("stream direct esperava 206, %d", rr.Code)
	}
	// Transcode.
	rr = raw(t, srv, "GET", "/api/v1/stream/"+songID+"?token="+mt+"&format=opus&maxBitRate=96", "")
	if rr.Code != http.StatusOK || rr.Body.Len() == 0 {
		t.Fatalf("stream transcode falhou: %d len=%d", rr.Code, rr.Body.Len())
	}
	// Cover (capa gerada no seed).
	rr = raw(t, srv, "GET", "/api/v1/cover/"+albumID+"?token="+mt+"&size=64", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("cover esperava 200, %d", rr.Code)
	}
	// Stream sem token → 401.
	if raw(t, srv, "GET", "/api/v1/stream/"+songID, "").Code != 401 {
		t.Fatal("stream sem token deveria 401")
	}
}

func TestBadRequests(t *testing.T) {
	srv, _ := newTestServer(t)
	token := login(t, srv, "admin", "pw")

	bad := func(method, path, body string) int {
		req := httptest.NewRequest(method, path, stringReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		srv.Router().ServeHTTP(rr, req)
		return rr.Code
	}

	cases := []struct {
		method, path, body string
		want               int
	}{
		{"POST", "/api/v1/star", `{}`, 400},                                    // ref inválido
		{"DELETE", "/api/v1/star", `{"id":""}`, 400},                           // ref inválido
		{"POST", "/api/v1/rating", `{"id":"x","type":"song","rating":9}`, 400}, // rating fora de faixa
		{"POST", "/api/v1/scrobble", `{}`, 400},                                // songId ausente
		{"POST", "/api/v1/playlists", `{}`, 400},                               // nome ausente
		{"POST", "/api/v1/playlists/smart", `{}`, 400},                         // nome ausente
		{"GET", "/api/v1/search", ``, 400},                                     // q ausente
		{"DELETE", "/api/v1/playlists/inexistente", ``, 404},                   // não encontrado
	}
	for _, c := range cases {
		if got := bad(c.method, c.path, c.body); got != c.want {
			t.Errorf("%s %s: esperava %d, obtive %d", c.method, c.path, c.want, got)
		}
	}
}

func TestMeUserGone(t *testing.T) {
	srv, store := newTestServer(t)
	token := login(t, srv, "admin", "pw")
	// Remove o usuário após emitir o token → /me deve responder 404.
	if _, err := store.DB().Exec(`DELETE FROM users`); err != nil {
		t.Fatal(err)
	}
	if raw(t, srv, "GET", "/api/v1/me", token).Code != 404 {
		t.Fatal("esperava 404 quando usuário não existe mais")
	}
}

func TestWebappCookieAuth(t *testing.T) {
	srv, store := newTestServer(t)
	_ = store.DB() // toca o acessor

	// Login como webapp (X-Client) → refresh vai por cookie httpOnly, não no corpo.
	req := httptest.NewRequest("POST", "/api/v1/auth/login", stringReader(`{"username":"admin","password":"pw"}`))
	req.Header.Set("X-Client", "webapp")
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("login webapp %d", rr.Code)
	}
	var tp tokenPair
	_ = json.Unmarshal(rr.Body.Bytes(), &tp)
	if tp.RefreshToken != "" {
		t.Fatal("webapp não deveria receber refresh no corpo")
	}
	cookies := rr.Result().Cookies()
	var refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "refreshToken" {
			refreshCookie = c
		}
	}
	if refreshCookie == nil || !refreshCookie.HttpOnly {
		t.Fatal("esperava cookie refreshToken httpOnly")
	}

	// Refresh via cookie (sem corpo).
	req = httptest.NewRequest("POST", "/api/v1/auth/refresh", nil)
	req.Header.Set("X-Client", "webapp")
	req.AddCookie(refreshCookie)
	rr = httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("refresh via cookie %d", rr.Code)
	}
}

func TestRefreshFlow(t *testing.T) {
	srv, _ := newTestServer(t)
	// Login nativo devolve refresh no corpo.
	body := `{"username":"admin","password":"pw"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/auth/login", stringReader(body))
	srv.Router().ServeHTTP(rr, req)
	var tp tokenPair
	_ = json.Unmarshal(rr.Body.Bytes(), &tp)
	if tp.RefreshToken == "" {
		t.Fatal("esperava refresh token no corpo (cliente nativo)")
	}

	// Refresh com o token → novo par.
	rr = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/v1/auth/refresh", stringReader(`{"refreshToken":"`+tp.RefreshToken+`"}`))
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("refresh esperava 200, %d", rr.Code)
	}

	// Reuso do token revogado → 401.
	rr = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/v1/auth/refresh", stringReader(`{"refreshToken":"`+tp.RefreshToken+`"}`))
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Fatalf("reuso de refresh deveria 401, %d", rr.Code)
	}

	// Refresh inválido → 401.
	rr = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/v1/auth/refresh", stringReader(`{"refreshToken":"lixo"}`))
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Fatalf("refresh inválido deveria 401, %d", rr.Code)
	}
}
