package db

import (
	"path/filepath"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.db")
	conn, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	// Tabelas criadas pelas migrações.
	for _, tbl := range []string{"users", "albums", "media_files", "playlists", "annotations", "search_fts"} {
		var name string
		err := conn.QueryRow(`SELECT name FROM sqlite_master WHERE name = ?`, tbl).Scan(&name)
		if err != nil {
			t.Fatalf("tabela %s ausente: %v", tbl, err)
		}
	}
	// Migrações registradas.
	var n int
	_ = conn.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n)
	if n < 2 {
		t.Fatalf("esperava >=2 migrações aplicadas, %d", n)
	}
	conn.Close()

	// Reabrir é idempotente (não reaplica).
	conn2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close()
}

func TestOpenInvalid(t *testing.T) {
	// Caminho dentro de diretório inexistente → erro ao abrir/migrar.
	if _, err := Open("/caminho/que/nao/existe/x.db"); err == nil {
		t.Fatal("esperava erro abrindo db em diretório inexistente")
	}
}
