package pkg

import (
	stdelliptic "crypto/elliptic"
	"fmt"
	"math/big"
)

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
	precomputed := map[string]*big.Int{
		r.Key(): new(big.Int).Set(zero),
	}

	for a := big.NewInt(1); a.Cmp(sqrtN) < 0; a = a.Add(a, one) {
		x, y := curve.Add(r.X, r.Y, p.X, p.Y)
		r = Point{X: x, Y: y}
		precomputed[r.Key()] = new(big.Int).Set(a)
	}

	// Now compute the giant steps and check the hash table for any matching point.
	negP := negPoint(p, curve.Params().P)
	sX, sY := curve.ScalarMult(negP.X, negP.Y, sqrtN.Bytes())

	r = q
	s := Point{X: sX, Y: sY}

	for b := big.NewInt(0); b.Cmp(sqrtN) < 0; b = b.Add(b, one) {
		if a, ok := precomputed[r.Key()]; ok {
			log := new(big.Int).Add(a, new(big.Int).Mul(sqrtN, b))
			steps := new(big.Int).Add(sqrtN, b)
			return log, steps, nil
		}
		rX, rY := curve.Add(r.X, r.Y, s.X, s.Y)
		r = Point{X: rX, Y: rY}
	}
	return nil, nil, fmt.Errorf("failed to find log")
}

func negPoint(point Point, p *big.Int) Point {
	if point.IsZero() {
		return point
	}
	y := new(big.Int).Set(point.Y)
	point.Y = y.Neg(y).Mod(y, p)
	return point
}
