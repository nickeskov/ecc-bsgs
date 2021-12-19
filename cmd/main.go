package main

import (
	"context"
	"flag"
	"log"
	"math/big"

	"github.com/nickeskov/ecc-bsgs/pkg"
)

var (
	threads = flag.Int("threads", 1, "Threads count for parallel calculation of giant steps")
)

func main() {
	flag.Parse()

	curve := pkg.TinyCurve
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
	x := big.NewInt(8684)

	p := pkg.Point{X: params.Gx, Y: params.Gy}
	qX, qY := curve.ScalarMult(p.X, p.Y, x.Bytes())
	q := pkg.Point{X: qX, Y: qY}

	log.Printf("curve order: %d", params.N)
	log.Printf("p = %s", p.String())
	log.Printf("q = %s", q.String())
	log.Printf("%d * p = q", x)

	ctx := context.TODO()
	logarithm, steps, err := pkg.EccLogBSGS(ctx, *threads, curve, p, q)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("log(p, q) = %d", logarithm)
	log.Printf("Took %d steps", steps)
}
