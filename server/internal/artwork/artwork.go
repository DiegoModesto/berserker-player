// Package artwork resolve e serve capas de álbum (arquivo na pasta ou capa
// embutida nas tags), com resize/cache opcional por tamanho.
package artwork

import (
	"bytes"
	"database/sql"
	"errors"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	"github.com/dhowden/tag"
	"golang.org/x/image/draw"
)

var ErrNoCover = errors.New("sem capa")

type Resolver struct {
	db       *sql.DB
	cacheDir string
}

func New(db *sql.DB, cacheDir string) *Resolver {
	_ = os.MkdirAll(cacheDir, 0o755)
	return &Resolver{db: db, cacheDir: cacheDir}
}

// Cover devolve os bytes da capa do item (id de álbum) no tamanho pedido
// (size=0 → original). Resultados redimensionados são cacheados em disco.
func (r *Resolver) Cover(id string, size int) ([]byte, string, error) {
	raw, ct, err := r.rawCover(id)
	if err != nil {
		return nil, "", err
	}
	if size <= 0 {
		return raw, ct, nil
	}
	// Cache por (id, size).
	cachePath := filepath.Join(r.cacheDir, id+"_"+itoa(size)+".jpg")
	if b, err := os.ReadFile(cachePath); err == nil {
		return b, "image/jpeg", nil
	}
	resized, err := resizeJPEG(raw, size)
	if err != nil {
		return raw, ct, nil // fallback: original
	}
	_ = os.WriteFile(cachePath, resized, 0o644)
	return resized, "image/jpeg", nil
}

// rawCover obtém a capa original: arquivo cover.* da pasta ou capa embutida.
func (r *Resolver) rawCover(albumID string) ([]byte, string, error) {
	var coverPath string
	err := r.db.QueryRow(`SELECT cover_path FROM albums WHERE id = ?`, albumID).Scan(&coverPath)
	if err != nil {
		return nil, "", ErrNoCover
	}
	if coverPath != "" {
		if b, err := os.ReadFile(coverPath); err == nil {
			return b, contentTypeFor(coverPath), nil
		}
	}
	// Capa embutida em alguma faixa do álbum.
	var path string
	err = r.db.QueryRow(`SELECT path FROM media_files WHERE album_id = ? AND has_embedded_cover = 1 LIMIT 1`, albumID).Scan(&path)
	if err == nil {
		if pic := embeddedPicture(path); pic != nil {
			ct := pic.MIMEType
			if ct == "" {
				ct = "image/jpeg"
			}
			return pic.Data, ct, nil
		}
	}
	return nil, "", ErrNoCover
}

func embeddedPicture(path string) *tag.Picture {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	m, err := tag.ReadFrom(f)
	if err != nil {
		return nil
	}
	return m.Picture()
}

func resizeJPEG(raw []byte, size int) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	// Mantém proporção, lado maior = size.
	nw, nh := size, size
	if w > h {
		nh = size * h / w
	} else {
		nw = size * w / h
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func contentTypeFor(path string) string {
	switch filepath.Ext(path) {
	case ".png":
		return "image/png"
	default:
		return "image/jpeg"
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
