package psi

import (
	"database/sql"
	"os"
	"testing"

	"github.com/SanthoshCheemala/LE-PSI/internal/storage"
	lepkg "github.com/SanthoshCheemala/LE-PSI/pkg/LE"
	"github.com/SanthoshCheemala/LE-PSI/pkg/matrix"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tuneinsight/lattigo/v3/ring"
)

func assertPolyEqual(t *testing.T, label string, got, want *ring.Poly) {
	t.Helper()
	if got == nil || want == nil {
		if got != want {
			t.Fatalf("%s nil mismatch: got=%v want=%v", label, got == nil, want == nil)
		}
		return
	}
	if len(got.Coeffs) != len(want.Coeffs) {
		t.Fatalf("%s coefficient limb count mismatch: got=%d want=%d", label, len(got.Coeffs), len(want.Coeffs))
	}
	for limb := range got.Coeffs {
		if len(got.Coeffs[limb]) != len(want.Coeffs[limb]) {
			t.Fatalf("%s coefficient length mismatch at limb %d: got=%d want=%d", label, limb, len(got.Coeffs[limb]), len(want.Coeffs[limb]))
		}
		for i := range got.Coeffs[limb] {
			if got.Coeffs[limb][i] != want.Coeffs[limb][i] {
				t.Fatalf("%s coefficient mismatch at limb=%d index=%d: got=%d want=%d", label, limb, i, got.Coeffs[limb][i], want.Coeffs[limb][i])
			}
		}
	}
}

func assertVectorEqual(t *testing.T, label string, got, want *matrix.Vector) {
	t.Helper()
	if got == nil || want == nil {
		if got != want {
			t.Fatalf("%s nil mismatch: got=%v want=%v", label, got == nil, want == nil)
		}
		return
	}
	if len(got.Elements) != len(want.Elements) {
		t.Fatalf("%s vector length mismatch: got=%d want=%d", label, len(got.Elements), len(want.Elements))
	}
	for i := range got.Elements {
		assertPolyEqual(t, label, got.Elements[i], want.Elements[i])
	}
}

func TestBuildMemoryTreeBottomUpMatchesSQLitePath(t *testing.T) {
	t.Setenv("PSI_SECURITY_LEVEL", "")

	serverSet := make([]uint64, 64)
	for i := range serverSet {
		serverSet[i] = uint64(i + 1)
	}

	leParams, err := SetupLEParameters(len(serverSet))
	if err != nil {
		t.Fatalf("SetupLEParameters: %v", err)
	}

	publicKeys := make([]*matrix.Vector, len(serverSet))
	for i := range serverSet {
		publicKeys[i], _ = leParams.KeyGen()
	}

	placement, _, err := placeCuckooLeaves(serverSet, leParams.Layers)
	if err != nil {
		t.Fatalf("placeCuckooLeaves: %v", err)
	}

	dbPath := t.TempDir() + "/tree.db"
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	if err := configureTreeBuildDB(db); err != nil {
		t.Fatalf("configureTreeBuildDB: %v", err)
	}
	if err := storage.InitializeTreeDB(db, leParams.Layers); err != nil {
		t.Fatalf("InitializeTreeDB: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	for i := range serverSet {
		lepkg.Upd(tx, placement[i], leParams.Layers, publicKeys[i], leParams)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	sqliteTree, err := lepkg.LoadTreeFromDB(db, leParams.Layers, leParams)
	if err != nil {
		t.Fatalf("LoadTreeFromDB: %v", err)
	}

	memoryTree, err := buildMemoryTreeBottomUp(leParams, placement, publicKeys)
	if err != nil {
		t.Fatalf("buildMemoryTreeBottomUp: %v", err)
	}

	assertVectorEqual(t, "root", memoryTree.Layers[0][0], sqliteTree.Layers[0][0])
	for i, leaf := range placement {
		gotLeft, gotRight := lepkg.WitGenMemory(memoryTree, leParams, leaf)
		wantLeft, wantRight := lepkg.WitGenMemory(sqliteTree, leParams, leaf)
		for layer := range gotLeft {
			assertVectorEqual(t, "left witness", gotLeft[layer], wantLeft[layer])
			assertVectorEqual(t, "right witness", gotRight[layer], wantRight[layer])
		}
		if i >= 7 {
			break
		}
	}
}
