package nativeapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func subsonicGet(t *testing.T, srv *Server, path string) map[string]any {
	t.Helper()
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, httptest.NewRequest("GET", path, nil))
	var out struct {
		Resp map[string]any `json:"subsonic-response"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	return out.Resp
}

func TestSubsonicPingAuth(t *testing.T) {
	srv, _ := newTestServer(t)

	// Credenciais corretas → status ok.
	resp := subsonicGet(t, srv, "/rest/ping?u=admin&p=pw&f=json&c=test&v=1.16.1")
	if resp["status"] != "ok" {
		t.Fatalf("esperava ok, obtive %v", resp)
	}

	// Senha errada → failed.
	resp = subsonicGet(t, srv, "/rest/ping?u=admin&p=errada&f=json")
	if resp["status"] != "failed" {
		t.Fatalf("esperava failed com senha errada, obtive %v", resp)
	}

	// Token auth (t/s) explicitamente não suportado.
	resp = subsonicGet(t, srv, "/rest/ping?u=admin&t=abc&s=xyz&f=json")
	if resp["status"] != "failed" {
		t.Fatalf("token auth deveria falhar, obtive %v", resp)
	}
}

func TestSubsonicAlbumList(t *testing.T) {
	srv, _ := newTestServer(t)
	seedLibrary(t, srv)

	resp := subsonicGet(t, srv, "/rest/getAlbumList2?u=admin&p=pw&type=newest&f=json")
	if resp["status"] != "ok" {
		t.Fatalf("status %v", resp)
	}
	list, ok := resp["albumList2"].(map[string]any)
	if !ok {
		t.Fatalf("sem albumList2: %v", resp)
	}
	albums, _ := list["album"].([]any)
	if len(albums) != 1 {
		t.Fatalf("esperava 1 álbum, obtive %d", len(albums))
	}
}

func TestSubsonicProtectedNoCreds(t *testing.T) {
	srv, _ := newTestServer(t)
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, httptest.NewRequest("GET", "/rest/getArtists?f=json", nil))
	// Sem credenciais → envelope failed (HTTP 200 com erro Subsonic).
	if rr.Code != http.StatusOK {
		t.Fatalf("Subsonic responde 200 com envelope; obtive %d", rr.Code)
	}
	var out struct {
		Resp map[string]any `json:"subsonic-response"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.Resp["status"] != "failed" {
		t.Fatalf("esperava failed sem credenciais")
	}
}
