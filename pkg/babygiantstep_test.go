package pkg

import (
	"context"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"testing"
)

type benchTimer interface {
	StartTimer()
	StopTimer()
}

type noopBenchTimer struct{}

func (n noopBenchTimer) StartTimer() {
	// noop
}

func (n noopBenchTimer) StopTimer() {
	// noop
}

func runEccLogBSGS(curve elliptic.Curve, bt benchTimer, threads int) error {
	params := curve.Params()

	var (
		x   = new(big.Int)
		p   = Point{X: params.Gx, Y: params.Gy}
		q   = Point{}
		err error
	)
	for x.IsUint64() && x.Uint64() == 0 {
		x, err = rand.Int(rand.Reader, params.N)
		if err != nil {
			log.Fatalln(err)
		}
		qX, qY := curve.ScalarMult(p.X, p.Y, x.Bytes())
		q = Point{X: qX, Y: qY} // q point should be zero
		if q.IsZero() {
			x = new(big.Int)
		}
	}

	bt.StartTimer()
	logarithm, _, err := EccLogBSGS(context.TODO(), threads, curve, p, q)
	bt.StopTimer()

	if err != nil {
		return err
	}
	actualQX, actualQy := curve.ScalarMult(p.X, p.Y, logarithm.Bytes())
	actualQ := Point{X: actualQX, Y: actualQy}

	if !q.Equals(actualQ) {
		return fmt.Errorf("actual logarithm=%d is not valid for p=%s and q=%s", logarithm, p, q)
	}
	return nil
}

func BenchmarkEccLogBSGS_SingleThread(b *testing.B) {
	curve := TinyCurve

	b.ReportAllocs()
	b.StopTimer()
	for i := 0; i < b.N; i++ {
		if err := runEccLogBSGS(curve, b, 1); err != nil {
			b.Fatal(err)
		}
	}
	b.StartTimer()
}

func BenchmarkEccLogBSGS_MultiThread(b *testing.B) {
	curve := TinyCurve

	b.ReportAllocs()
	b.StopTimer()
	for i := 0; i < b.N; i++ {
		if err := runEccLogBSGS(curve, b, 4); err != nil {
			b.Fatal(err)
		}
	}
	b.StartTimer()
}

func TestEccLogBSGS_SingleThread(t *testing.T) {
	const iterations = 2048
	curve := TinyCurve

	for i := 0; i < iterations; i++ {
		if err := runEccLogBSGS(curve, noopBenchTimer{}, 1); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEccLogBSGS_MultiThread(t *testing.T) {
	const iterations = 2048
	curve := TinyCurve

	for i := 0; i < iterations; i++ {
		if err := runEccLogBSGS(curve, noopBenchTimer{}, 4); err != nil {
			t.Fatal(err)
		}
	}
}
