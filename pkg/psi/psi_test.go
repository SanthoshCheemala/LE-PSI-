package psi

import (
	"fmt"
	"os"
	"testing"
)

func TestCuckooReducesCollisions(t *testing.T) {
	// Test that 2-choice placement reduces collisions vs single-choice
	// Use a very small leaf space to force birthday-paradox collisions
	layers := 7  // 2^7 = 128 leaves
	m := 300     // ~234% load factor — heavy collisions guaranteed

	// Generate m random-looking hashes
	hashes := make([]uint64, m)
	for i := 0; i < m; i++ {
		// Simulate real hash output with good mixing
		v := uint64(i + 1)
		v ^= v << 13
		v ^= v >> 7
		v ^= v << 17
		v *= 0xbf58476d1ce4e5b9
		v ^= v >> 31
		hashes[i] = v
	}

	// Single-choice: count collisions with modulo mapping
	singleOccupied := make(map[uint64]bool)
	singleCollisions := 0
	for _, h := range hashes {
		leaf := ReduceToTreeIndex(h, layers)
		if singleOccupied[leaf] {
			singleCollisions++
		}
		singleOccupied[leaf] = true
	}

	// 2-choice: count collisions with Cuckoo placement
	cuckooOccupied := make(map[uint64]bool)
	cuckooCollisions := 0
	for _, h := range hashes {
		leaf1 := ReduceToTreeIndex(h, layers)
		leaf2 := ReduceToTreeIndex2(h, layers)
		if !cuckooOccupied[leaf1] {
			cuckooOccupied[leaf1] = true
		} else if !cuckooOccupied[leaf2] {
			cuckooOccupied[leaf2] = true
		} else {
			cuckooCollisions++
		}
	}

	t.Logf("Leaves: 2^%d = %d", layers, 1<<layers)
	t.Logf("Items: %d (load factor: %.1f%%)", m, float64(m)/float64(int(1)<<layers)*100)
	t.Logf("Single-choice collisions: %d (%.1f%%)", singleCollisions, float64(singleCollisions)/float64(m)*100)
	t.Logf("2-choice Cuckoo collisions: %d (%.1f%%)", cuckooCollisions, float64(cuckooCollisions)/float64(m)*100)

	if singleCollisions == 0 {
		t.Skip("No single-choice collisions with this hash set — test not meaningful")
	}
	if cuckooCollisions >= singleCollisions {
		t.Errorf("Cuckoo should reduce collisions: single=%d, cuckoo=%d", singleCollisions, cuckooCollisions)
	}
}

func TestHash2Independence(t *testing.T) {
	// Verify H1 and H2 produce different leaf indices
	layers := 10
	sameCount := 0
	total := 1000
	for i := 0; i < total; i++ {
		h := uint64(i*31337 + 42)
		l1 := ReduceToTreeIndex(h, layers)
		l2 := ReduceToTreeIndex2(h, layers)
		if l1 == l2 {
			sameCount++
		}
	}
	t.Logf("H1==H2 for %d/%d items (%.1f%%)", sameCount, total, float64(sameCount)/float64(total)*100)
	if float64(sameCount)/float64(total) > 0.05 {
		t.Errorf("H1 and H2 are not independent enough: %d/%d same", sameCount, total)
	}
}

func TestEndToEndWithCuckoo(t *testing.T) {
	// Small end-to-end PSI test with known intersection
	os.Setenv("PSI_VERBOSE", "false")

	serverSet := []uint64{100, 200, 300, 400, 500}
	clientSet := []uint64{200, 400, 600} // intersection = {200, 400}

	treePath := "/tmp/test_cuckoo_psi.db"
	os.Remove(treePath)
	defer os.Remove(treePath)

	ctx, err := ServerInitialize(serverSet, treePath)
	if err != nil {
		t.Fatalf("ServerInitialize failed: %v", err)
	}

	pp, msg, le := GetPublicParameters(ctx)
	ciphertexts := ClientEncrypt(clientSet, pp, msg, le)

	t.Logf("Client items: %d, Ciphertexts produced: %d (should be 2x)", len(clientSet), len(ciphertexts))
	if len(ciphertexts) != 2*len(clientSet) {
		t.Errorf("Expected %d ciphertexts (2 per item), got %d", 2*len(clientSet), len(ciphertexts))
	}

	intersection, err := DetectIntersectionWithContext(ctx, ciphertexts)
	if err != nil {
		t.Fatalf("DetectIntersection failed: %v", err)
	}

	t.Logf("Intersection found: %v", intersection)

	// Check that 200 and 400 are in the intersection
	found := make(map[uint64]bool)
	for _, v := range intersection {
		found[v] = true
	}

	for _, expected := range []uint64{200, 400} {
		if !found[expected] {
			t.Errorf("Expected %d in intersection, not found", expected)
		}
	}

	// 600 should NOT be in the intersection
	if found[600] {
		t.Errorf("600 should NOT be in intersection")
	}

	fmt.Printf("✅ End-to-end PSI with 2-choice Cuckoo: intersection=%v\n", intersection)
}
