package pkg

import (
	stdelliptic "crypto/elliptic"
	"fmt"
	"math/big"
)

type precomputedStepsMap map[string]*big.Int

func EccLogBSGS(curve stdelliptic.Curve, p Point, q Point) (*big.Int, *big.Int, error) {
	if !curve.IsOnCurve(p.X, p.Y) {
		return nil, nil, fmt.Errorf("point 'p' %s is not on curve %s", p.String(), curve.Params().Name)
	}
	if !curve.IsOnCurve(q.X, q.Y) {
		return nil, nil, fmt.Errorf("point 'q' %s is not on curve %s", q.String(), curve.Params().Name)
	}

	sqrtN := new(big.Int).Sqrt(curve.Params().N)
	sqrtN.Add(sqrtN, one)

	// Compute the baby steps and store them in the 'precomputed' hash table.
	r := Point{X: zero, Y: zero}
	precomputed := precomputedStepsMap{
		r.Key(): new(big.Int).Set(zero),
	}

	for a := big.NewInt(1); a.Cmp(sqrtN) < 0; a = a.Add(a, one) {
		x, y := curve.Add(r.X, r.Y, p.X, p.Y)
		r = Point{X: x, Y: y}
		precomputed[r.Key()] = new(big.Int).Set(a)
	}

	// Now compute the giant steps and check the hash table for any matching point.
	negP := negPoint(p, curve.Params().P)                     // compute -P
	sX, sY := curve.ScalarMult(negP.X, negP.Y, sqrtN.Bytes()) // compute -mP
	s := Point{X: sX, Y: sY}                                  // s == -mP

	a, b, err := giantSteps(curve, precomputed, q, s, zero, sqrtN)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find log for p=%s and q=%s: %w", p, q, err)
	}
	log := new(big.Int).Add(new(big.Int).Mul(sqrtN, b), a) // log = mb + a
	steps := new(big.Int).Add(sqrtN, b)
	return log, steps, nil
}

// giantSteps computes giant steps and returns a and b coefficients for equation 'log = bm + a' where m == sqrt(N)
func giantSteps(
	curve stdelliptic.Curve,
	precomputed precomputedStepsMap,
	q, s Point,
	sqrtNStart, sqrtNEnd *big.Int,
) (*big.Int, *big.Int, error) {
	if sqrtNEnd.Cmp(sqrtNStart) == -1 {
		panic(fmt.Sprintf("sqrtNEnd=%d < sqrtNStart=%d", sqrtNEnd, sqrtNStart))
	}

	r := q // Q - 0 * mP
	sqrtNSub := new(big.Int).Sub(sqrtNEnd, sqrtNStart)

	if sqrtNSub.IsUint64() {
		sqrtNSubNative := sqrtNSub.Uint64()
		for b := uint64(0); b < sqrtNSubNative; b++ {
			if a, ok := precomputed[r.Key()]; ok {
				bigB := new(big.Int).SetUint64(b)
				bigB.Add(bigB, sqrtNStart) // restore b
				return a, bigB, nil
			}
			rX, rY := curve.Add(r.X, r.Y, s.X, s.Y) // Q - b*mP; remember, that s == -mP
			r = Point{X: rX, Y: rY}
		}
	} else { // doesn't fit into native uint64 type
		for b := new(big.Int).Set(sqrtNStart); b.Cmp(sqrtNEnd) < 0; b = b.Add(b, one) {
			if a, ok := precomputed[r.Key()]; ok {
				return a, b, nil
			}
			rX, rY := curve.Add(r.X, r.Y, s.X, s.Y) // Q - b*mP; remember, that s == -mP
			r = Point{X: rX, Y: rY}
		}
	}
	return nil, nil, fmt.Errorf("failed to find a and b coefficients")
}

func negPoint(point Point, p *big.Int) Point {
	if point.IsZero() {
		return point
	}
	y := new(big.Int).Set(point.Y)
	point.Y = y.Neg(y).Mod(y, p)
	return point
}
