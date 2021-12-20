package main

import (
	"context"
	"flag"
	"log"
	"math/big"
	"time"

	"github.com/nickeskov/ecc-bsgs/pkg"
)

var (
	threads = flag.Int("threads", 1, "Threads count for parallel calculation of giant steps")
)

func main() {
	flag.Parse()

	curve := pkg.NormalCurve
	params := curve.Params()

	log.Printf("threads count = %d", *threads)

	//var (
	//	x   = new(big.Int)
	//	err error
	//)
	//for x.IsUint64() && x.Uint64() == 0 {
	//	x, err = rand.Int(rand.Reader, params.N)
	//	if err != nil {
	//		log.Fatalln(err)
	//	}
	//}
	x := big.NewInt(446818759577)

	p := pkg.Point{X: params.Gx, Y: params.Gy}
	qX, qY := curve.ScalarMult(p.X, p.Y, x.Bytes())
	q := pkg.Point{X: qX, Y: qY}

	log.Printf("curve order: %d", params.N)
	log.Printf("p = %s", p.String())
	log.Printf("q = %s", q.String())
	log.Printf("%d * p = q", x)

	now := time.Now()

	ctx := context.TODO()
	logarithm, steps, err := pkg.EccLogBSGS(ctx, *threads, curve, p, q)
	if err != nil {
		log.Fatalln(err)
	}
	since := time.Since(now)

	log.Printf("log(p, q) = %d", logarithm)
	log.Printf("Took %d steps, duration %v", steps, since)
}
