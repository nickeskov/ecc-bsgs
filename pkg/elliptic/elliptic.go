// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package elliptic implements several standard elliptic curves over prime
// fields.
package elliptic

// This package operates, internally, on Jacobian coordinates. For a given
// (x, y) position on the curve, the Jacobian coordinates are (x1, y1, z1)
// where x = x1/z1² and y = y1/z1³. The greatest speedups come when the whole
// calculation can be performed within the transform (as in ScalarMult and
// ScalarBaseMult). But even for Add and Double, it's faster to apply and
// reverse the transform than to operate in affine coordinates.

import (
	"crypto/elliptic"
	"io"
	"math/big"
)

// A EnhancedCurve represents a short-form Weierstrass curve with variable 'a' and 'b' polynomial parameters.
//
// Note that the point at infinity (0, 0) is not considered on the curve, and
// although it can be returned by Add, Double, ScalarMult, or ScalarBaseMult, it
// can't be marshaled or unmarshaled, and IsOnCurve will return false for it.
type EnhancedCurve interface {
	elliptic.Curve
	CustomParams() *EnhancedCurveParams
}

// EnhancedCurveParams contains the parameters of an elliptic curve and also provides
// a generic, non-constant time implementation of EnhancedCurve.
type EnhancedCurveParams struct {
	elliptic.CurveParams
	A *big.Int // the constant of the curve equation: x³ + ax + b.
}

func (curve *EnhancedCurveParams) Params() *elliptic.CurveParams {
	return &curve.CurveParams
}

func (curve *EnhancedCurveParams) CustomParams() *EnhancedCurveParams {
	return curve
}

// polynomial returns x³ + ax + b.
func (curve *EnhancedCurveParams) polynomial(x *big.Int) *big.Int {
	// x³
	x3 := new(big.Int).Mul(x, x)
	x3.Mul(x3, x)
	// ax
	aX := new(big.Int).Mul(x, curve.A)
	// b
	b := curve.B

	// compute result
	x3.Add(x3, aX)
	x3.Add(x3, b)
	// perform modulo
	x3.Mod(x3, curve.P)

	return x3
}

func (curve *EnhancedCurveParams) IsOnCurve(x, y *big.Int) bool {
	// y² = x³ + ax + b
	y2 := new(big.Int).Mul(y, y)
	y2.Mod(y2, curve.P)

	return curve.polynomial(x).Cmp(y2) == 0
}

func (curve *EnhancedCurveParams) Add(x1, y1, x2, y2 *big.Int) (*big.Int, *big.Int) {
	// TODO(nickeskov): implement me
	panic("implement me")
}

func (curve *EnhancedCurveParams) Double(x1, y1 *big.Int) (*big.Int, *big.Int) {
	return curve.Add(x1, y1, x1, y1)
}

func (curve *EnhancedCurveParams) ScalarMult(Bx, By *big.Int, k []byte) (*big.Int, *big.Int) {
	// TODO(nickeskov): implement me
	panic("implement me")
}

func (curve *EnhancedCurveParams) ScalarBaseMult(k []byte) (*big.Int, *big.Int) {
	return curve.ScalarMult(curve.Gx, curve.Gy, k)
}

var mask = []byte{0xff, 0x1, 0x3, 0x7, 0xf, 0x1f, 0x3f, 0x7f}

// GenerateKey returns a public/private key pair. The private key is
// generated using the given reader, which must return random data.
func GenerateKey(curve elliptic.Curve, rand io.Reader) (priv []byte, x, y *big.Int, err error) {
	N := curve.Params().N
	bitSize := N.BitLen()
	byteLen := (bitSize + 7) / 8
	priv = make([]byte, byteLen)

	for x == nil {
		_, err = io.ReadFull(rand, priv)
		if err != nil {
			return
		}
		// We have to mask off any excess bits in the case that the size of the
		// underlying field is not a whole number of bytes.
		priv[0] &= mask[bitSize%8]
		// This is because, in tests, rand will return all zeros and we don't
		// want to get the point at infinity and loop forever.
		priv[1] ^= 0x42

		// If the scalar is out of range, sample another random number.
		if new(big.Int).SetBytes(priv).Cmp(N) >= 0 {
			continue
		}

		x, y = curve.ScalarBaseMult(priv)
	}
	return
}

// Marshal converts a point on the curve into the uncompressed form specified in
// section 4.3.6 of ANSI X9.62.
func Marshal(curve elliptic.Curve, x, y *big.Int) []byte {
	byteLen := (curve.Params().BitSize + 7) / 8

	ret := make([]byte, 1+2*byteLen)
	ret[0] = 4 // uncompressed point

	x.FillBytes(ret[1 : 1+byteLen])
	y.FillBytes(ret[1+byteLen : 1+2*byteLen])

	return ret
}

// MarshalCompressed converts a point on the curve into the compressed form
// specified in section 4.3.6 of ANSI X9.62.
func MarshalCompressed(curve elliptic.Curve, x, y *big.Int) []byte {
	byteLen := (curve.Params().BitSize + 7) / 8
	compressed := make([]byte, 1+byteLen)
	compressed[0] = byte(y.Bit(0)) | 2
	x.FillBytes(compressed[1:])
	return compressed
}

// Unmarshal converts a point, serialized by Marshal, into an x, y pair.
// It is an error if the point is not in uncompressed form or is not on the curve.
// On error, x = nil.
func Unmarshal(curve elliptic.Curve, data []byte) (x, y *big.Int) {
	byteLen := (curve.Params().BitSize + 7) / 8
	if len(data) != 1+2*byteLen {
		return nil, nil
	}
	if data[0] != 4 { // uncompressed form
		return nil, nil
	}
	p := curve.Params().P
	x = new(big.Int).SetBytes(data[1 : 1+byteLen])
	y = new(big.Int).SetBytes(data[1+byteLen:])
	if x.Cmp(p) >= 0 || y.Cmp(p) >= 0 {
		return nil, nil
	}
	if !curve.IsOnCurve(x, y) {
		return nil, nil
	}
	return
}

// UnmarshalCompressed converts a point, serialized by MarshalCompressed, into an x, y pair.
// It is an error if the point is not in compressed form or is not on the curve.
// On error, x = nil.
func UnmarshalCompressed(curve EnhancedCurve, data []byte) (x, y *big.Int) {
	byteLen := (curve.Params().BitSize + 7) / 8
	if len(data) != 1+byteLen {
		return nil, nil
	}
	if data[0] != 2 && data[0] != 3 { // compressed form
		return nil, nil
	}
	p := curve.Params().P
	x = new(big.Int).SetBytes(data[1:])
	if x.Cmp(p) >= 0 {
		return nil, nil
	}
	// y² = x³ + ax + b
	y = curve.CustomParams().polynomial(x)
	y = y.ModSqrt(y, p)
	if y == nil {
		return nil, nil
	}
	if byte(y.Bit(0)) != data[0]&1 {
		y.Neg(y).Mod(y, p)
	}
	if !curve.IsOnCurve(x, y) {
		return nil, nil
	}
	return
}
