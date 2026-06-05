package core

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/DiegoModesto/berserker-player/server/internal/db"
	"github.com/DiegoModesto/berserker-player/server/internal/model"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	return New(database)
}

// seedMedia insere um álbum com N faixas via upserts (sem ffmpeg/arquivos reais).
func seedMedia(t *testing.T, s *Store, artist, album, genre string, year int, titles ...string) (artistID, albumID string, songIDs []string) {
	t.Helper()
	var err error
	artistID, err = s.UpsertArtist(artist)
	if err != nil {
		t.Fatal(err)
	}
	albumID, err = s.UpsertAlbum(album, artistID, year, genre)
	if err != nil {
		t.Fatal(err)
	}
	for i, title := range titles {
		id, err := s.UpsertMediaFile(MediaInput{
			Path: filepath.Join("/music", album, title+".mp3"), Title: title,
			AlbumID: albumID, ArtistID: artistID, Track: i + 1, Duration: 100 + i, Suffix: "mp3", MTime: int64(i + 1),
		})
		if err != nil {
			t.Fatal(err)
		}
		songIDs = append(songIDs, id)
	}
	if err := s.RecomputeCounts(); err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildSearchIndex(); err != nil {
		t.Fatal(err)
	}
	return artistID, albumID, songIDs
}

func TestUsersCRUD(t *testing.T) {
	s := newStore(t)
	if n, _ := s.CountUsers(); n != 0 {
		t.Fatalf("esperava 0 usuários, %d", n)
	}
	u, err := s.CreateUser("admin", "hash", true)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.UserByUsername("admin")
	if err != nil || got.ID != u.ID || !got.IsAdmin {
		t.Fatalf("UserByUsername inesperado: %+v err=%v", got, err)
	}
	byID, err := s.UserByID(u.ID)
	if err != nil || byID.Username != "admin" {
		t.Fatalf("UserByID inesperado: %+v", byID)
	}
	if _, err := s.UserByUsername("ninguem"); err != ErrNotFound {
		t.Fatalf("esperava ErrNotFound, %v", err)
	}
	if n, _ := s.CountUsers(); n != 1 {
		t.Fatalf("esperava 1 usuário, %d", n)
	}
}

func TestRefreshTokens(t *testing.T) {
	s := newStore(t)
	u, _ := s.CreateUser("u", "h", false)
	id, err := s.SaveRefreshToken(u.ID, "hash1", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	rec, err := s.FindRefreshToken("hash1")
	if err != nil || rec.ID != id || rec.Revoked {
		t.Fatalf("FindRefreshToken inesperado: %+v err=%v", rec, err)
	}
	if err := s.RevokeRefreshToken(id); err != nil {
		t.Fatal(err)
	}
	rec, _ = s.FindRefreshToken("hash1")
	if !rec.Revoked {
		t.Fatal("token deveria estar revogado")
	}
	_, _ = s.SaveRefreshToken(u.ID, "hash2", time.Now().Add(time.Hour))
	if err := s.RevokeAllUserTokens(u.ID); err != nil {
		t.Fatal(err)
	}
	rec, _ = s.FindRefreshToken("hash2")
	if !rec.Revoked {
		t.Fatal("RevokeAllUserTokens deveria revogar hash2")
	}
	if _, err := s.FindRefreshToken("inexistente"); err != ErrNotFound {
		t.Fatalf("esperava ErrNotFound")
	}
}

func TestLibraryListingAndGet(t *testing.T) {
	s := newStore(t)
	u, _ := s.CreateUser("u", "h", false)
	artistID, albumID, songIDs := seedMedia(t, s, "Hirasawa", "Berserk OST", "Soundtrack", 1997, "Guts", "Forces")

	page, err := s.ListAlbums(u.ID, AlbumQuery{Limit: 10})
	if err != nil || page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("ListAlbums inesperado: %+v err=%v", page, err)
	}
	if page.Items[0].SongCount != 2 || page.Items[0].ArtistName != "Hirasawa" {
		t.Fatalf("álbum agregado errado: %+v", page.Items[0])
	}
	al, err := s.GetAlbum(u.ID, albumID)
	if err != nil || al.Name != "Berserk OST" {
		t.Fatalf("GetAlbum: %+v err=%v", al, err)
	}
	if _, err := s.GetAlbum(u.ID, "inexistente"); err != ErrNotFound {
		t.Fatalf("esperava ErrNotFound")
	}
	songs, err := s.SongsByAlbum(u.ID, albumID)
	if err != nil || len(songs) != 2 || songs[0].Title != "Guts" {
		t.Fatalf("SongsByAlbum: %+v err=%v", songs, err)
	}
	song, err := s.GetSong(u.ID, songIDs[0])
	if err != nil || song.Title != "Guts" {
		t.Fatalf("GetSong: %+v err=%v", song, err)
	}

	arts, err := s.ListArtists(u.ID, "name", "asc", 0, 10)
	if err != nil || arts.Total != 1 {
		t.Fatalf("ListArtists: %+v err=%v", arts, err)
	}
	art, err := s.GetArtist(u.ID, artistID)
	if err != nil || art.AlbumCount != 1 || art.SongCount != 2 {
		t.Fatalf("GetArtist: %+v err=%v", art, err)
	}
	albs, err := s.AlbumsByArtist(u.ID, artistID)
	if err != nil || len(albs) != 1 {
		t.Fatalf("AlbumsByArtist: %+v err=%v", albs, err)
	}

	// Path interno para streaming.
	p, suf, err := s.GetSongPath(songIDs[0])
	if err != nil || suf != "mp3" || p == "" {
		t.Fatalf("GetSongPath: %q %q err=%v", p, suf, err)
	}
}

func TestSearchFTS(t *testing.T) {
	s := newStore(t)
	u, _ := s.CreateUser("u", "h", false)
	seedMedia(t, s, "Hirasawa", "Berserk OST", "OST", 1997, "Guts", "Forces")
	res, err := s.Search(u.ID, "Forces", 10)
	if err != nil || len(res.Songs) == 0 {
		t.Fatalf("busca por 'Forces' falhou: %+v err=%v", res, err)
	}
	res, _ = s.Search(u.ID, "Berserk", 10)
	if len(res.Albums) == 0 {
		t.Fatal("busca por 'Berserk' deveria achar álbum")
	}
	res, _ = s.Search(u.ID, "   ", 10)
	if len(res.Songs)+len(res.Albums)+len(res.Artists) != 0 {
		t.Fatal("query vazia deveria retornar nada")
	}
}

func TestAnnotations(t *testing.T) {
	s := newStore(t)
	u, _ := s.CreateUser("u", "h", false)
	_, albumID, songIDs := seedMedia(t, s, "A", "Album", "G", 2000, "S1", "S2")

	if err := s.Star(u.ID, songIDs[0], model.ItemSong); err != nil {
		t.Fatal(err)
	}
	song, _ := s.GetSong(u.ID, songIDs[0])
	if !song.Starred {
		t.Fatal("faixa deveria estar favoritada")
	}
	if err := s.Unstar(u.ID, songIDs[0], model.ItemSong); err != nil {
		t.Fatal(err)
	}
	song, _ = s.GetSong(u.ID, songIDs[0])
	if song.Starred {
		t.Fatal("faixa não deveria estar favoritada")
	}

	if err := s.SetRating(u.ID, songIDs[0], model.ItemSong, 5); err != nil {
		t.Fatal(err)
	}
	song, _ = s.GetSong(u.ID, songIDs[0])
	if song.Rating != 5 {
		t.Fatalf("rating esperado 5, %d", song.Rating)
	}

	// Scrobble incrementa; repetição imediata é deduplicada.
	now := time.Now()
	_ = s.Scrobble(u.ID, songIDs[0], now)
	_ = s.Scrobble(u.ID, songIDs[0], now.Add(2*time.Second)) // dedup
	song, _ = s.GetSong(u.ID, songIDs[0])
	if song.PlayCount != 1 {
		t.Fatalf("playcount esperado 1 (dedup), %d", song.PlayCount)
	}
	_ = s.Scrobble(u.ID, songIDs[0], now.Add(time.Minute)) // fora da janela
	song, _ = s.GetSong(u.ID, songIDs[0])
	if song.PlayCount != 2 {
		t.Fatalf("playcount esperado 2, %d", song.PlayCount)
	}

	// Filtro starred em álbuns.
	_ = s.Star(u.ID, albumID, model.ItemAlbum)
	page, _ := s.ListAlbums(u.ID, AlbumQuery{Filter: "starred", Limit: 10})
	if page.Total != 1 {
		t.Fatalf("filtro starred deveria achar 1 álbum, %d", page.Total)
	}
}

func TestPlaylists(t *testing.T) {
	s := newStore(t)
	u, _ := s.CreateUser("u", "h", false)
	_, _, ids := seedMedia(t, s, "A", "Album", "G", 2000, "S1", "S2", "S3")

	pl, err := s.CreatePlaylist(u.ID, "Mix", ids[:2])
	if err != nil || pl.SongCount != 2 {
		t.Fatalf("CreatePlaylist: %+v err=%v", pl, err)
	}
	all, _ := s.ListPlaylists(u.ID)
	if len(all) != 1 {
		t.Fatalf("esperava 1 playlist, %d", len(all))
	}
	songs, _ := s.PlaylistSongs(u.ID, pl.ID)
	if len(songs) != 2 || songs[0].ID != ids[0] {
		t.Fatalf("ordem inicial errada: %+v", songs)
	}
	// Reordena e adiciona.
	name := "Mix 2"
	if _, err := s.UpdatePlaylist(u.ID, pl.ID, &name, []string{ids[2], ids[1], ids[0]}); err != nil {
		t.Fatal(err)
	}
	songs, _ = s.PlaylistSongs(u.ID, pl.ID)
	if len(songs) != 3 || songs[0].ID != ids[2] {
		t.Fatalf("reorder falhou: %+v", songs)
	}
	got, _ := s.GetPlaylist(u.ID, pl.ID)
	if got.Name != "Mix 2" {
		t.Fatalf("rename falhou: %q", got.Name)
	}
	if err := s.DeletePlaylist(u.ID, pl.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeletePlaylist(u.ID, pl.ID); err != ErrNotFound {
		t.Fatalf("esperava ErrNotFound ao deletar de novo")
	}
}

func TestSmartPlaylists(t *testing.T) {
	s := newStore(t)
	u, _ := s.CreateUser("u", "h", false)
	seedMedia(t, s, "Rocker", "Rock Album", "Rock", 2010, "R1", "R2")
	seedMedia(t, s, "Jazzer", "Jazz Album", "Jazz", 1990, "J1")

	pl, err := s.CreateSmartPlaylist(u.ID, "Só Rock", SmartRules{Genre: "Rock", Sort: "title"})
	if err != nil {
		t.Fatal(err)
	}
	smart, rules := s.IsSmart(pl.ID)
	if !smart || rules.Genre != "Rock" {
		t.Fatalf("IsSmart/rules inesperado: %v %+v", smart, rules)
	}
	songs, err := s.EvaluateSmart(u.ID, rules)
	if err != nil || len(songs) != 2 {
		t.Fatalf("EvaluateSmart genre=Rock esperava 2: %+v err=%v", songs, err)
	}

	// Por ano.
	byYear, _ := s.EvaluateSmart(u.ID, SmartRules{MinYear: 2000})
	if len(byYear) != 2 {
		t.Fatalf("MinYear=2000 esperava 2, %d", len(byYear))
	}
	// Por artista (substring).
	byArtist, _ := s.EvaluateSmart(u.ID, SmartRules{Artist: "Jazz"})
	if len(byArtist) != 1 {
		t.Fatalf("Artist=Jazz esperava 1, %d", len(byArtist))
	}
	// Limite.
	limited, _ := s.EvaluateSmart(u.ID, SmartRules{Limit: 1})
	if len(limited) != 1 {
		t.Fatalf("Limit=1 esperava 1, %d", len(limited))
	}
	if smart, _ := s.IsSmart("inexistente"); smart {
		t.Fatal("IsSmart de id inexistente deveria ser false")
	}
}

func TestScannerSupportMethods(t *testing.T) {
	s := newStore(t)
	_, _, ids := seedMedia(t, s, "A", "Album", "G", 2000, "S1", "S2")
	paths, err := s.ExistingPaths()
	if err != nil || len(paths) != 2 {
		t.Fatalf("ExistingPaths esperava 2: %v err=%v", paths, err)
	}
	// Remove uma faixa e limpa agregados.
	if err := s.DeleteByPaths([]string{filepath.Join("/music", "Album", "S1.mp3")}); err != nil {
		t.Fatal(err)
	}
	_ = ids
	if err := s.CleanupEmpty(); err != nil {
		t.Fatal(err)
	}
	if err := s.RecomputeCounts(); err != nil {
		t.Fatal(err)
	}
	paths, _ = s.ExistingPaths()
	if len(paths) != 1 {
		t.Fatalf("após delete esperava 1 path, %d", len(paths))
	}
}
