package core

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/DiegoModesto/berserker-player/server/internal/model"
)

// ---- Upserts usados pelo scanner ----

// UpsertArtist devolve o id do artista (criando se necessário), por nome.
func (s *Store) UpsertArtist(name string) (string, error) {
	if name == "" {
		name = "Unknown Artist"
	}
	var id string
	err := s.db.QueryRow(`SELECT id FROM artists WHERE name = ?`, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	id = NewID()
	_, err = s.db.Exec(`INSERT INTO artists(id, name) VALUES(?,?)`, id, name)
	return id, err
}

// UpsertAlbum devolve o id do álbum (criando se necessário), por (artista, nome).
func (s *Store) UpsertAlbum(name, artistID string, year int, genre string) (string, error) {
	if name == "" {
		name = "Unknown Album"
	}
	var id string
	err := s.db.QueryRow(`SELECT id FROM albums WHERE artist_id = ? AND name = ?`, artistID, name).Scan(&id)
	if err == nil {
		if year > 0 || genre != "" {
			_, _ = s.db.Exec(`UPDATE albums SET year = COALESCE(NULLIF(?,0), year), genre = COALESCE(NULLIF(?,''), genre) WHERE id = ?`, year, genre, id)
		}
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	id = NewID()
	_, err = s.db.Exec(`INSERT INTO albums(id, name, artist_id, year, genre, created_at) VALUES(?,?,?,?,?,?)`,
		id, name, artistID, year, genre, nowUTC().Format(rfc3339))
	return id, err
}

// MediaInput descreve uma faixa para persistência.
type MediaInput struct {
	Path          string
	Title         string
	AlbumID       string
	ArtistID      string
	Track         int
	Disc          int
	Duration      int
	BitRate       int
	SampleRate    int
	Suffix        string
	Size          int64
	MTime         int64
	EmbeddedCover bool
}

// UpsertMediaFile insere/atualiza uma faixa por path; devolve o id.
func (s *Store) UpsertMediaFile(m MediaInput) (string, error) {
	var id string
	err := s.db.QueryRow(`SELECT id FROM media_files WHERE path = ?`, m.Path).Scan(&id)
	switch {
	case err == nil:
		_, err = s.db.Exec(`UPDATE media_files SET title=?, album_id=?, artist_id=?, track=?, disc=?, duration=?, bit_rate=?, sample_rate=?, suffix=?, size=?, mtime=?, has_embedded_cover=? WHERE id=?`,
			m.Title, m.AlbumID, m.ArtistID, m.Track, m.Disc, m.Duration, m.BitRate, m.SampleRate, m.Suffix, m.Size, m.MTime, boolToInt(m.EmbeddedCover), id)
		return id, err
	case errors.Is(err, sql.ErrNoRows):
		id = NewID()
		_, err = s.db.Exec(`INSERT INTO media_files(id, path, title, album_id, artist_id, track, disc, duration, bit_rate, sample_rate, suffix, size, mtime, has_embedded_cover, created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, m.Path, m.Title, m.AlbumID, m.ArtistID, m.Track, m.Disc, m.Duration, m.BitRate, m.SampleRate, m.Suffix, m.Size, m.MTime, boolToInt(m.EmbeddedCover), nowUTC().Format(rfc3339))
		return id, err
	default:
		return "", err
	}
}

// SetAlbumCover registra o caminho da capa de um álbum (cover.jpg/folder.jpg).
func (s *Store) SetAlbumCover(albumID, coverPath string) error {
	_, err := s.db.Exec(`UPDATE albums SET cover_path = ? WHERE id = ? AND cover_path = ''`, coverPath, albumID)
	return err
}

// ExistingPaths devolve path->mtime de todas as faixas indexadas (scan incremental).
func (s *Store) ExistingPaths() (map[string]int64, error) {
	rows, err := s.db.Query(`SELECT path, mtime FROM media_files`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var p string
		var m int64
		if err := rows.Scan(&p, &m); err != nil {
			return nil, err
		}
		out[p] = m
	}
	return out, rows.Err()
}

// DeleteByPaths remove faixas órfãs (arquivos sumidos).
func (s *Store) DeleteByPaths(paths []string) error {
	for _, p := range paths {
		if _, err := s.db.Exec(`DELETE FROM media_files WHERE path = ?`, p); err != nil {
			return err
		}
	}
	return nil
}

// CleanupEmpty remove álbuns e artistas sem faixas e recomputa contadores.
func (s *Store) CleanupEmpty() error {
	stmts := []string{
		`DELETE FROM albums WHERE id NOT IN (SELECT DISTINCT album_id FROM media_files)`,
		`DELETE FROM artists WHERE id NOT IN (SELECT DISTINCT artist_id FROM media_files)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// RecomputeCounts atualiza song_count/duration de álbuns e contadores de artistas.
func (s *Store) RecomputeCounts() error {
	stmts := []string{
		`UPDATE albums SET
            song_count = (SELECT COUNT(*) FROM media_files m WHERE m.album_id = albums.id),
            duration   = (SELECT COALESCE(SUM(duration),0) FROM media_files m WHERE m.album_id = albums.id)`,
		`UPDATE artists SET
            album_count = (SELECT COUNT(*) FROM albums a WHERE a.artist_id = artists.id),
            song_count  = (SELECT COUNT(*) FROM media_files m WHERE m.artist_id = artists.id)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// RebuildSearchIndex reconstrói a tabela FTS a partir do índice atual.
func (s *Store) RebuildSearchIndex() error {
	if _, err := s.db.Exec(`DELETE FROM search_fts`); err != nil {
		return err
	}
	inserts := []string{
		`INSERT INTO search_fts(item_id, item_type, text) SELECT id, 'artist', name FROM artists`,
		`INSERT INTO search_fts(item_id, item_type, text) SELECT id, 'album', name FROM albums`,
		`INSERT INTO search_fts(item_id, item_type, text) SELECT id, 'song', title FROM media_files`,
	}
	for _, q := range inserts {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// ---- Leitura ----

// GetSongPath devolve o caminho em disco de uma faixa (para streaming).
func (s *Store) GetSongPath(id string) (path, suffix string, err error) {
	err = s.db.QueryRow(`SELECT path, suffix FROM media_files WHERE id = ?`, id).Scan(&path, &suffix)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return path, suffix, err
}

const songSelect = `
SELECT m.id, m.title, m.album_id, al.name, m.artist_id, ar.name,
       m.track, m.disc, m.duration, m.bit_rate, m.sample_rate, m.suffix, m.size,
       m.album_id,
       COALESCE(an.starred_at IS NOT NULL, 0), COALESCE(an.rating,0), COALESCE(an.play_count,0)
FROM media_files m
JOIN albums al ON al.id = m.album_id
JOIN artists ar ON ar.id = m.artist_id
LEFT JOIN annotations an ON an.item_id = m.id AND an.item_type = 'song' AND an.user_id = ?`

func (s *Store) scanSong(rows *sql.Rows) (model.Song, error) {
	var so model.Song
	var starred int
	if err := rows.Scan(&so.ID, &so.Title, &so.AlbumID, &so.AlbumName, &so.ArtistID, &so.ArtistName,
		&so.Track, &so.Disc, &so.Duration, &so.BitRate, &so.SampleRate, &so.Suffix, &so.Size,
		&so.CoverArtID, &starred, &so.Rating, &so.PlayCount); err != nil {
		return so, err
	}
	so.Starred = starred == 1
	return so, nil
}

// SongsByAlbum devolve as faixas de um álbum, ordenadas por disco/faixa.
func (s *Store) SongsByAlbum(userID, albumID string) ([]model.Song, error) {
	rows, err := s.db.Query(songSelect+` WHERE m.album_id = ? ORDER BY m.disc, m.track, m.title`, userID, albumID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Song
	for rows.Next() {
		so, err := s.scanSong(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, so)
	}
	return out, rows.Err()
}

// GetSong devolve uma faixa por id.
func (s *Store) GetSong(userID, id string) (model.Song, error) {
	rows, err := s.db.Query(songSelect+` WHERE m.id = ?`, userID, id)
	if err != nil {
		return model.Song{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return model.Song{}, ErrNotFound
	}
	return s.scanSong(rows)
}

// ---- Álbuns ----

type AlbumQuery struct {
	Filter   string // all|recent|frequent|random|starred
	Genre    string
	ArtistID string
	Sort     string // name|year|recentlyAdded|playCount
	Order    string // asc|desc
	Offset   int
	Limit    int
}

const albumSelect = `
SELECT al.id, al.name, al.artist_id, ar.name, al.year, al.genre, al.song_count, al.duration,
       al.id, COALESCE(an.starred_at IS NOT NULL, 0), al.play_count, al.created_at
FROM albums al
JOIN artists ar ON ar.id = al.artist_id
LEFT JOIN annotations an ON an.item_id = al.id AND an.item_type = 'album' AND an.user_id = ?`

func (s *Store) scanAlbum(rows *sql.Rows) (model.Album, error) {
	var a model.Album
	var starred int
	var created string
	if err := rows.Scan(&a.ID, &a.Name, &a.ArtistID, &a.ArtistName, &a.Year, &a.Genre, &a.SongCount, &a.Duration,
		&a.CoverArtID, &starred, &a.PlayCount, &created); err != nil {
		return a, err
	}
	a.Starred = starred == 1
	a.CreatedAt = parseTime(created)
	return a, nil
}

// ListAlbums devolve uma página de álbuns conforme filtros/ordenação.
func (s *Store) ListAlbums(userID string, q AlbumQuery) (model.Page[model.Album], error) {
	var page model.Page[model.Album]
	page.Offset, page.Limit = q.Offset, q.Limit

	where := []string{}
	args := []any{userID}
	if q.Genre != "" {
		where = append(where, "al.genre = ?")
		args = append(args, q.Genre)
	}
	if q.ArtistID != "" {
		where = append(where, "al.artist_id = ?")
		args = append(args, q.ArtistID)
	}
	if q.Filter == "starred" {
		where = append(where, "an.starred_at IS NOT NULL")
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	// total
	countQ := `SELECT COUNT(*) FROM albums al JOIN artists ar ON ar.id = al.artist_id
        LEFT JOIN annotations an ON an.item_id = al.id AND an.item_type='album' AND an.user_id = ?` + whereSQL
	if err := s.db.QueryRow(countQ, args...).Scan(&page.Total); err != nil {
		return page, err
	}

	orderSQL := orderForAlbums(q)
	listQ := albumSelect + whereSQL + orderSQL + " LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)
	rows, err := s.db.Query(listQ, args...)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		a, err := s.scanAlbum(rows)
		if err != nil {
			return page, err
		}
		page.Items = append(page.Items, a)
	}
	return page, rows.Err()
}

func orderForAlbums(q AlbumQuery) string {
	dir := "ASC"
	if strings.EqualFold(q.Order, "desc") {
		dir = "DESC"
	}
	switch q.Filter {
	case "recent":
		return " ORDER BY al.created_at DESC"
	case "frequent":
		return " ORDER BY al.play_count DESC"
	case "random":
		return " ORDER BY RANDOM()"
	}
	switch q.Sort {
	case "year":
		return " ORDER BY al.year " + dir + ", al.name ASC"
	case "recentlyAdded":
		return " ORDER BY al.created_at " + dir
	case "playCount":
		return " ORDER BY al.play_count " + dir
	default:
		return " ORDER BY al.name " + dir
	}
}

// GetAlbum devolve um álbum por id.
func (s *Store) GetAlbum(userID, id string) (model.Album, error) {
	rows, err := s.db.Query(albumSelect+` WHERE al.id = ?`, userID, id)
	if err != nil {
		return model.Album{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return model.Album{}, ErrNotFound
	}
	return s.scanAlbum(rows)
}

// ---- Artistas ----

func (s *Store) ListArtists(userID, sort, order string, offset, limit int) (model.Page[model.Artist], error) {
	var page model.Page[model.Artist]
	page.Offset, page.Limit = offset, limit
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM artists`).Scan(&page.Total); err != nil {
		return page, err
	}
	dir := "ASC"
	if strings.EqualFold(order, "desc") {
		dir = "DESC"
	}
	orderSQL := " ORDER BY ar.name " + dir
	switch sort {
	case "albumCount":
		orderSQL = " ORDER BY ar.album_count " + dir
	case "recentlyAdded":
		orderSQL = " ORDER BY ar.id " + dir
	}
	q := `SELECT ar.id, ar.name, ar.album_count, ar.song_count,
            COALESCE(an.starred_at IS NOT NULL, 0)
        FROM artists ar
        LEFT JOIN annotations an ON an.item_id = ar.id AND an.item_type='artist' AND an.user_id = ?` +
		orderSQL + " LIMIT ? OFFSET ?"
	rows, err := s.db.Query(q, userID, limit, offset)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var a model.Artist
		var starred int
		if err := rows.Scan(&a.ID, &a.Name, &a.AlbumCount, &a.SongCount, &starred); err != nil {
			return page, err
		}
		a.Starred = starred == 1
		page.Items = append(page.Items, a)
	}
	return page, rows.Err()
}

func (s *Store) GetArtist(userID, id string) (model.Artist, error) {
	var a model.Artist
	var starred int
	err := s.db.QueryRow(`SELECT ar.id, ar.name, ar.album_count, ar.song_count,
            COALESCE(an.starred_at IS NOT NULL,0)
        FROM artists ar
        LEFT JOIN annotations an ON an.item_id = ar.id AND an.item_type='artist' AND an.user_id = ?
        WHERE ar.id = ?`, userID, id).Scan(&a.ID, &a.Name, &a.AlbumCount, &a.SongCount, &starred)
	if errors.Is(err, sql.ErrNoRows) {
		return a, ErrNotFound
	}
	a.Starred = starred == 1
	return a, err
}

func (s *Store) AlbumsByArtist(userID, artistID string) ([]model.Album, error) {
	rows, err := s.db.Query(albumSelect+` WHERE al.artist_id = ? ORDER BY al.year, al.name`, userID, artistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Album
	for rows.Next() {
		a, err := s.scanAlbum(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ---- Busca (FTS5) ----

type SearchResult struct {
	Artists []model.Artist `json:"artists"`
	Albums  []model.Album  `json:"albums"`
	Songs   []model.Song   `json:"songs"`
}

func (s *Store) Search(userID, query string, limit int) (SearchResult, error) {
	var res SearchResult
	res.Artists, res.Albums, res.Songs = []model.Artist{}, []model.Album{}, []model.Song{}
	match := ftsQuery(query)
	if match == "" {
		return res, nil
	}
	rows, err := s.db.Query(`SELECT item_id, item_type FROM search_fts WHERE search_fts MATCH ? LIMIT ?`, match, limit*3)
	if err != nil {
		return res, err
	}
	defer rows.Close()
	var artistIDs, albumIDs, songIDs []string
	for rows.Next() {
		var id, typ string
		if err := rows.Scan(&id, &typ); err != nil {
			return res, err
		}
		switch typ {
		case "artist":
			if len(artistIDs) < limit {
				artistIDs = append(artistIDs, id)
			}
		case "album":
			if len(albumIDs) < limit {
				albumIDs = append(albumIDs, id)
			}
		case "song":
			if len(songIDs) < limit {
				songIDs = append(songIDs, id)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return res, err
	}
	for _, id := range artistIDs {
		if a, err := s.GetArtist(userID, id); err == nil {
			res.Artists = append(res.Artists, a)
		}
	}
	for _, id := range albumIDs {
		if a, err := s.GetAlbum(userID, id); err == nil {
			res.Albums = append(res.Albums, a)
		}
	}
	for _, id := range songIDs {
		if so, err := s.GetSong(userID, id); err == nil {
			res.Songs = append(res.Songs, so)
		}
	}
	return res, nil
}

// ftsQuery transforma a query do usuário em um prefixo-match seguro para FTS5.
func ftsQuery(q string) string {
	fields := strings.Fields(q)
	var terms []string
	for _, f := range fields {
		clean := strings.Map(func(r rune) rune {
			if r == '"' || r == '*' {
				return -1
			}
			return r
		}, f)
		if clean == "" {
			continue
		}
		terms = append(terms, fmt.Sprintf("\"%s\"*", clean))
	}
	return strings.Join(terms, " ")
}
