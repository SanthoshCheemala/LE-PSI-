package psi

import (
	"sort"
	"testing"
)

func sortedCopy(values []uint64) []uint64 {
	out := append([]uint64(nil), values...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func assertUint64SetEqual(t *testing.T, got, want []uint64) {
	t.Helper()
	got = sortedCopy(got)
	want = sortedCopy(want)
	if len(got) != len(want) {
		t.Fatalf("set length mismatch: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("set mismatch: got %v want %v", got, want)
		}
	}
}

func TestAuditToyChunkedPSIHonestFlow(t *testing.T) {
	serverSet := []uint64{10, 20, 30, 40}
	clientSet := []uint64{20, 40, 99}

	dbPath := t.TempDir() + "/tree.db"
	ctx, err := ServerInitializeChunked(serverSet, dbPath)
	if err != nil {
		t.Fatalf("ServerInitializeChunked: %v", err)
	}

	pp, msg, le := GetPublicParameters(ctx)
	ciphertexts := ClientEncrypt(clientSet, pp, msg, le)
	intersection, stats, err := DetectIntersectionChunkedWithContext(ctx, ciphertexts, ChunkedDetectionOptions{
		ChunkSize:   2,
		WorkerCount: 1,
		ForceGC:     true,
	})
	if err != nil {
		t.Fatalf("DetectIntersectionChunkedWithContext: %v", err)
	}

	assertUint64SetEqual(t, intersection, []uint64{20, 40})
	if stats.ActualDecCalls != 2 {
		t.Fatalf("expected only the two matching leaves to be decrypted, got %d", stats.ActualDecCalls)
	}
}

func TestAuditSerializedParametersPreserveEncryptionSettings(t *testing.T) {
	ctx, err := ServerInitializeChunked([]uint64{10, 20}, t.TempDir()+"/tree.db")
	if err != nil {
		t.Fatalf("ServerInitializeChunked: %v", err)
	}

	pp, msg, le := GetPublicParameters(ctx)
	_, _, decoded, err := DeserializeParameters(SerializeParameters(pp, msg, le))
	if err != nil {
		t.Fatalf("DeserializeParameters: %v", err)
	}

	if decoded.QBits != le.QBits {
		t.Fatalf("QBits mismatch after serialization: got %d want %d", decoded.QBits, le.QBits)
	}
	if decoded.Sigma != le.Sigma {
		t.Fatalf("Sigma mismatch after serialization: got %v want %v", decoded.Sigma, le.Sigma)
	}
	if decoded.Bound != le.Bound {
		t.Fatalf("Bound mismatch after serialization: got %d want %d", decoded.Bound, le.Bound)
	}
}

func TestAuditCuckooSecondLeafStillMatches(t *testing.T) {
	serverSet := []uint64{1, 33}

	dbPath := t.TempDir() + "/tree.db"
	ctx, err := ServerInitializeChunked(serverSet, dbPath)
	if err != nil {
		t.Fatalf("ServerInitializeChunked: %v", err)
	}

	foundSecondLeafPlacement := false
	for i, value := range serverSet {
		if ctx.TreeIndices[i] != ReduceToTreeIndex(value, ctx.LEParams.Layers) {
			foundSecondLeafPlacement = true
			break
		}
	}
	if !foundSecondLeafPlacement {
		t.Fatalf("test setup did not force a second-choice cuckoo placement; placements=%v", ctx.TreeIndices)
	}

	pp, msg, le := GetPublicParameters(ctx)
	ciphertexts := ClientEncrypt(serverSet, pp, msg, le)
	intersection, _, err := DetectIntersectionChunkedWithContext(ctx, ciphertexts, ChunkedDetectionOptions{
		ChunkSize:   1,
		WorkerCount: 1,
	})
	if err != nil {
		t.Fatalf("DetectIntersectionChunkedWithContext: %v", err)
	}

	assertUint64SetEqual(t, intersection, serverSet)
}

func TestAuditTargetLeafMetadataTamperingCanHideRealMatch(t *testing.T) {
	serverSet := []uint64{10, 20, 30}
	clientSet := []uint64{20}

	dbPath := t.TempDir() + "/tree.db"
	ctx, err := ServerInitializeChunked(serverSet, dbPath)
	if err != nil {
		t.Fatalf("ServerInitializeChunked: %v", err)
	}

	pp, msg, le := GetPublicParameters(ctx)
	ciphertexts := ClientEncrypt(clientSet, pp, msg, le)
	occupiedLeaves := make(map[uint64]bool, len(ctx.TreeIndices))
	for _, leaf := range ctx.TreeIndices {
		occupiedLeaves[leaf] = true
	}

	for i := range ciphertexts {
		for leaf := uint64(0); ; leaf++ {
			if !occupiedLeaves[leaf] {
				ciphertexts[i].TargetLeaf = leaf
				break
			}
		}
	}

	intersection, stats, err := DetectIntersectionChunkedWithContext(ctx, ciphertexts, ChunkedDetectionOptions{
		ChunkSize:   2,
		WorkerCount: 1,
	})
	if err != nil {
		t.Fatalf("DetectIntersectionChunkedWithContext: %v", err)
	}
	if len(intersection) != 0 {
		t.Fatalf("expected metadata tampering to hide the match in current implementation, got %v", intersection)
	}
	if stats.ActualDecCalls != 0 {
		t.Fatalf("expected no decryptions after retargeting to empty leaves, got %d", stats.ActualDecCalls)
	}
}

func TestAuditDuplicateServerInputsReturnDuplicateMatches(t *testing.T) {
	serverSet := []uint64{7, 7}
	clientSet := []uint64{7}

	dbPath := t.TempDir() + "/tree.db"
	ctx, err := ServerInitializeChunked(serverSet, dbPath)
	if err != nil {
		t.Fatalf("ServerInitializeChunked: %v", err)
	}

	pp, msg, le := GetPublicParameters(ctx)
	ciphertexts := ClientEncrypt(clientSet, pp, msg, le)
	intersection, _, err := DetectIntersectionChunkedWithContext(ctx, ciphertexts, ChunkedDetectionOptions{
		ChunkSize:   1,
		WorkerCount: 1,
	})
	if err != nil {
		t.Fatalf("DetectIntersectionChunkedWithContext: %v", err)
	}

	if len(intersection) != 2 || intersection[0] != 7 || intersection[1] != 7 {
		t.Fatalf("expected current implementation to return duplicate server matches, got %v", intersection)
	}
}

func TestAuditDifferentItemsWithSameLeafCanFalsePositive(t *testing.T) {
	serverSet := []uint64{1}
	clientSet := []uint64{17}

	dbPath := t.TempDir() + "/tree.db"
	ctx, err := ServerInitializeChunked(serverSet, dbPath)
	if err != nil {
		t.Fatalf("ServerInitializeChunked: %v", err)
	}

	if ReduceToTreeIndex(serverSet[0], ctx.LEParams.Layers) != ReduceToTreeIndex(clientSet[0], ctx.LEParams.Layers) {
		t.Fatalf("test setup expected server and client to share first leaf")
	}
	if serverSet[0] == clientSet[0] {
		t.Fatalf("test setup requires different raw items")
	}

	pp, msg, le := GetPublicParameters(ctx)
	ciphertexts := ClientEncrypt(clientSet, pp, msg, le)
	intersection, _, err := DetectIntersectionChunkedWithContext(ctx, ciphertexts, ChunkedDetectionOptions{
		ChunkSize:   1,
		WorkerCount: 1,
	})
	if err != nil {
		t.Fatalf("DetectIntersectionChunkedWithContext: %v", err)
	}

	if len(intersection) != 1 || intersection[0] != serverSet[0] {
		t.Fatalf("expected current leaf-level implementation to false-positive on %v, got %v", serverSet, intersection)
	}
}

func TestAuditDistributedStylePreReducedClientCanReturnWrongCollisionRecord(t *testing.T) {
	serverSet := []uint64{33, 65}
	clientSet := []uint64{33}

	dbPath := t.TempDir() + "/tree.db"
	ctx, err := ServerInitializeChunked(serverSet, dbPath)
	if err != nil {
		t.Fatalf("ServerInitializeChunked: %v", err)
	}

	if ReduceToTreeIndex(serverSet[0], ctx.LEParams.Layers) != ReduceToTreeIndex(serverSet[1], ctx.LEParams.Layers) {
		t.Fatalf("test setup expected server records to share first leaf")
	}
	if ctx.TreeIndices[0] == ReduceToTreeIndex(serverSet[0], ctx.LEParams.Layers) {
		t.Fatalf("test setup expected first record to be displaced to its second cuckoo leaf; placements=%v", ctx.TreeIndices)
	}

	pp, msg, le := GetPublicParameters(ctx)

	honestCiphertexts := ClientEncrypt(clientSet, pp, msg, le)
	honestIntersection, _, err := DetectIntersectionChunkedWithContext(ctx, honestCiphertexts, ChunkedDetectionOptions{
		ChunkSize:   1,
		WorkerCount: 1,
	})
	if err != nil {
		t.Fatalf("honest DetectIntersectionChunkedWithContext: %v", err)
	}
	assertUint64SetEqual(t, honestIntersection, []uint64{33, 65})

	preReducedClientSet := []uint64{ReduceToTreeIndex(clientSet[0], ctx.LEParams.Layers)}
	distributedStyleCiphertexts := ClientEncrypt(preReducedClientSet, pp, msg, le)
	distributedStyleIntersection, _, err := DetectIntersectionChunkedWithContext(ctx, distributedStyleCiphertexts, ChunkedDetectionOptions{
		ChunkSize:   1,
		WorkerCount: 1,
	})
	if err != nil {
		t.Fatalf("distributed-style DetectIntersectionChunkedWithContext: %v", err)
	}

	assertUint64SetEqual(t, distributedStyleIntersection, []uint64{65})
}
