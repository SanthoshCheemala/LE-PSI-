package psi

// cxtx_serial.go — JSON-serializable wrappers for Cxtx
// Needed for network transport between coordinator and shard VMs.

import (
	"fmt"

	"github.com/SanthoshCheemala/LE-PSI/pkg/LE"
	"github.com/SanthoshCheemala/LE-PSI/pkg/matrix"
	"github.com/tuneinsight/lattigo/v3/ring"
)

// SerializableCxtx is the JSON-serializable form of Cxtx.
// All ring.Poly and matrix.Vector fields are flattened to [][]uint64.
type SerializableCxtx struct {
	C0 [][][]uint64 `json:"c0"` // [layers+1][N][D] coefficients
	C1 [][][]uint64 `json:"c1"`
	C  [][]uint64   `json:"c"`  // [N][D]
	D  []uint64     `json:"d"`  // [D]
}

// SerializeCxtx converts a Cxtx to its JSON-serializable form.
func SerializeCxtx(ct Cxtx) SerializableCxtx {
	serVecSlice := func(vecs []*matrix.Vector) [][][]uint64 {
		out := make([][][]uint64, len(vecs))
		for i, v := range vecs {
			out[i] = make([][]uint64, len(v.Elements))
			for j, p := range v.Elements {
				out[i][j] = append([]uint64{}, p.Coeffs[0]...)
			}
		}
		return out
	}

	serVec := func(v *matrix.Vector) [][]uint64 {
		out := make([][]uint64, len(v.Elements))
		for i, p := range v.Elements {
			out[i] = append([]uint64{}, p.Coeffs[0]...)
		}
		return out
	}

	return SerializableCxtx{
		C0: serVecSlice(ct.C0),
		C1: serVecSlice(ct.C1),
		C:  serVec(ct.C),
		D:  append([]uint64{}, ct.D.Coeffs[0]...),
	}
}

// DeserializeCxtx reconstructs a Cxtx from its serialized form.
// Requires the LE params to reconstruct the ring.
func DeserializeCxtx(s SerializableCxtx, le *LE.LE) (Cxtx, error) {
	r := le.R

	deVecSlice := func(raw [][][]uint64) ([]*matrix.Vector, error) {
		vecs := make([]*matrix.Vector, len(raw))
		for i, elems := range raw {
			v := &matrix.Vector{Elements: make([]*ring.Poly, len(elems))}
			for j, coeffs := range elems {
				p := r.NewPoly()
				if len(coeffs) != len(p.Coeffs[0]) {
					return nil, fmt.Errorf("coeff len mismatch at [%d][%d]: got %d want %d",
						i, j, len(coeffs), len(p.Coeffs[0]))
				}
				copy(p.Coeffs[0], coeffs)
				v.Elements[j] = p
			}
			vecs[i] = v
		}
		return vecs, nil
	}

	c0, err := deVecSlice(s.C0)
	if err != nil {
		return Cxtx{}, fmt.Errorf("C0: %w", err)
	}
	c1, err := deVecSlice(s.C1)
	if err != nil {
		return Cxtx{}, fmt.Errorf("C1: %w", err)
	}

	cVec := &matrix.Vector{Elements: make([]*ring.Poly, len(s.C))}
	for i, coeffs := range s.C {
		p := r.NewPoly()
		copy(p.Coeffs[0], coeffs)
		cVec.Elements[i] = p
	}

	dPoly := r.NewPoly()
	copy(dPoly.Coeffs[0], s.D)

	return Cxtx{C0: c0, C1: c1, C: cVec, D: dPoly}, nil
}
