package pkg

import (
	stdelliptic "crypto/elliptic"
	"fmt"
	"math/big"
)

var TinyCurve = &stdelliptic.CurveParams{
	P:       big.NewInt(10177),
	N:       big.NewInt(10331),
	B:       big.NewInt(-1),
	Gx:      big.NewInt(2),
	Gy:      big.NewInt(1),
	BitSize: 14,
	Name:    "TinyCurve",
}

type Point struct {
	X, Y *big.Int
}

func (p Point) IsZero() bool {
	return p.X.Cmp(zero) == 0 && p.Y.Cmp(zero) == 0
}

func (p Point) String() string {
	return fmt.Sprintf("(%x,%x)", p.X, p.Y)
}

func (p Point) Equals(other Point) bool {
	return p.X.Cmp(other.X) == 0 && p.Y.Cmp(other.Y) == 0
}

func (p Point) keyBytes() []byte {
	xBytes := p.X.Bytes()
	yBytes := p.Y.Bytes()

	buf := make([]byte, 0, len(xBytes)+len(yBytes)+2)
	buf = append(buf, xBytes...)
	buf = append(buf, yBytes...)
	buf = append(buf, byte(p.X.Sign()), byte(p.Y.Sign()))

	return buf
}

func (p Point) Key() string {
	return string(p.keyBytes())
}
