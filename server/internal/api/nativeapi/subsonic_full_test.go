package nativeapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Exercita o grosso da camada Subsonic com biblioteca semeada.
func TestSubsonicFullSurface(t *testing.T) {
	srv, store := newTestServer(t)
	seedLibrary(t, srv)
	_ = store

	get := func(path string) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		srv.Router().ServeHTTP(rr, httptest.NewRequest("GET", path, nil))
		return rr
	}
	creds := "u=admin&p=pw&f=json"

	// IDs reais via getArtists/getAlbumList2.
	if get("/rest/getLicense?"+creds).Code != 200 {
		t.Fatal("getLicense")
	}
	if get("/rest/getArtists?"+creds).Code != 200 {
		t.Fatal("getArtists")
	}

	tok := login(t, srv, "admin", "pw")
	var artists struct {
		Items []struct{ ID string } `json:"items"`
	}
	authGet(t, srv, tok, "/api/v1/artists", &artists)
	artistID := artists.Items[0].ID
	var albums struct {
		Items []struct{ ID string } `json:"items"`
	}
	authGet(t, srv, tok, "/api/v1/albums", &albums)
	albumID := albums.Items[0].ID

	if get("/rest/getArtist?id="+artistID+"&"+creds).Code != 200 {
		t.Fatal("getArtist")
	}
	if get("/rest/getAlbum?id="+albumID+"&"+creds).Code != 200 {
		t.Fatal("getAlbum")
	}
	if get("/rest/search3?query=One&"+creds).Code != 200 {
		t.Fatal("search3")
	}
	if get("/rest/getCoverArt?id="+albumID+"&size=64&"+creds).Code != 200 {
		t.Fatal("getCoverArt")
	}
	if get("/rest/star?id="+albumID+"&"+creds).Code != 200 {
		t.Fatal("star")
	}
	if get("/rest/unstar?id="+albumID+"&"+creds).Code != 200 {
		t.Fatal("unstar")
	}

	// Song id para stream/scrobble.
	var detail struct {
		Songs []struct{ ID string } `json:"songs"`
	}
	authGet(t, srv, tok, "/api/v1/albums/"+albumID, &detail)
	songID := detail.Songs[0].ID

	rr := get("/rest/stream?id=" + songID + "&" + creds)
	if rr.Code != http.StatusOK && rr.Code != http.StatusPartialContent {
		t.Fatalf("stream subsonic: %d", rr.Code)
	}
	if get("/rest/stream?id="+songID+"&format=opus&maxBitRate=96&"+creds).Code != 200 {
		t.Fatal("stream subsonic transcode")
	}
	if get("/rest/scrobble?id="+songID+"&"+creds).Code != 200 {
		t.Fatal("scrobble")
	}

	// enc:hex password também autentica.
	enc := "u=admin&p=enc:7077&f=json" // hex("pw")=7077
	if get("/rest/ping?"+enc).Code != 200 {
		t.Fatal("enc password")
	}
}
