// Package model contém as entidades de domínio do BerserkerPlayer.
// Os campos JSON espelham os schemas de openapi.yaml (fonte da verdade).
package model

import "time"

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	IsAdmin      bool      `json:"isAdmin"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Artist struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AlbumCount int    `json:"albumCount"`
	SongCount  int    `json:"songCount"`
	Starred    bool   `json:"starred"`
	CoverArtID string `json:"coverArtId,omitempty"`
}

type Album struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	ArtistID   string    `json:"artistId"`
	ArtistName string    `json:"artistName"`
	Year       int       `json:"year,omitempty"`
	Genre      string    `json:"genre,omitempty"`
	SongCount  int       `json:"songCount"`
	Duration   int       `json:"duration"`
	CoverArtID string    `json:"coverArtId,omitempty"`
	Starred    bool      `json:"starred"`
	PlayCount  int       `json:"playCount"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Song struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	AlbumID    string `json:"albumId"`
	AlbumName  string `json:"albumName"`
	ArtistID   string `json:"artistId"`
	ArtistName string `json:"artistName"`
	Track      int    `json:"track,omitempty"`
	Disc       int    `json:"disc,omitempty"`
	Duration   int    `json:"duration"`
	BitRate    int    `json:"bitRate,omitempty"`
	SampleRate int    `json:"sampleRate,omitempty"`
	Suffix     string `json:"suffix"`
	Size       int64  `json:"size"`
	CoverArtID string `json:"coverArtId,omitempty"`
	Starred    bool   `json:"starred"`
	Rating     int    `json:"rating,omitempty"`
	PlayCount  int    `json:"playCount"`
	// Path é interno (não serializado): usado para streaming/scan.
	Path string `json:"-"`
}

type Playlist struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	OwnerID   string    `json:"ownerId"`
	SongCount int       `json:"songCount"`
	Duration  int       `json:"duration"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Page é o envelope de paginação offset/limit usado nas listagens.
type Page[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// ItemType identifica o tipo de um item anotável (star/rating).
type ItemType string

const (
	ItemArtist ItemType = "artist"
	ItemAlbum  ItemType = "album"
	ItemSong   ItemType = "song"
)
