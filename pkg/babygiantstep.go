package pkg

import (
	"context"
	stdelliptic "crypto/elliptic"
	"fmt"
	"math/big"
)

var ErcCtxDone = fmt.Errorf("context done")

type precomputedStepsMap map[string]*big.Int

func EccLogBSGS(ctx context.Context, threads int, curve stdelliptic.Curve, p Point, q Point) (*big.Int, *big.Int, error) {
	if !curve.IsOnCurve(p.X, p.Y) {
		return nil, nil, fmt.Errorf("point 'p' %s is not on curve %s", p.String(), curve.Params().Name)
	}
	if !curve.IsOnCurve(q.X, q.Y) {
		return nil, nil, fmt.Errorf("point 'q' %s is not on curve %s", q.String(), curve.Params().Name)
	}
	if threads < 1 || threads > 128 {
		return nil, nil, fmt.Errorf("threads=%d must be 1 <= theads <= 128", threads)
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

	if threads == 1 { // fast path if activated singlethread mode
		a, b, err := giantSteps(ctx, curve, precomputed, q, s, zero, sqrtN)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to find log for p=%s and q=%s: %w", p, q, err)
		}
		log := new(big.Int).Add(new(big.Int).Mul(sqrtN, b), a) // log = mb + a
		steps := new(big.Int).Add(sqrtN, b)
		return log, steps, nil
	}

	// process in multithread mode

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tasks := giantStepsTasks(sqrtN, threads)
	out := make(chan giantStepsTaskResult, len(tasks))
	runningThreads := len(tasks)

	for i := range tasks {
		task := tasks[i]
		go func() {
			a, b, err := giantSteps(ctx, curve, precomputed, q, s, task.sqrtNStart, task.sqrtNEnd)
			out <- giantStepsTaskResult{
				a:    a,
				b:    b,
				task: task,
				err:  err,
			}
		}()
	}
	var res giantStepsTaskResult
	for {
		select {
		case res = <-out:
			runningThreads -= 1
			if res.err == nil {
				a, b := res.a, res.b
				log := new(big.Int).Add(new(big.Int).Mul(sqrtN, b), a) // log = mb + a
				steps := new(big.Int).Add(sqrtN, b)
				return log, steps, nil
			}
			if runningThreads == 0 {
				return nil, nil, fmt.Errorf("failed to find log for p=%s and q=%s: %w", p, q, res.err)
			}
		case <-ctx.Done():
			return nil, nil, ErcCtxDone
		}
	}
}

type giantStepsTaskResult struct {
	a, b *big.Int
	task giantStepsTask
	err  error
}

type giantStepsTask struct {
	taskNum              int
	sqrtNStart, sqrtNEnd *big.Int
}

// giantSteps computes giant steps and returns a and b coefficients for equation 'log = bm + a' where m == sqrt(N)
func giantSteps(
	ctx context.Context,
	curve stdelliptic.Curve,
	precomputed precomputedStepsMap,
	q, s Point, // s == -mP
	sqrtNStart, sqrtNEnd *big.Int,
) (*big.Int, *big.Int, error) {
	if sqrtNEnd.Cmp(sqrtNStart) == -1 {
		panic(fmt.Sprintf("sqrtNEnd=%d < sqrtNStart=%d", sqrtNEnd, sqrtNStart))
	}

	bmPX, bmPY := curve.ScalarMult(s.X, s.Y, sqrtNStart.Bytes())
	rX, rY := curve.Add(q.X, q.Y, bmPX, bmPY)
	r := Point{X: rX, Y: rY} // Q - sqrtNStart * mP

	sqrtNSub := new(big.Int).Sub(sqrtNEnd, sqrtNStart)

	if sqrtNSub.IsUint64() {
		sqrtNSubNative := sqrtNSub.Uint64()
		for b := uint64(0); b < sqrtNSubNative; b++ {
			select {
			case <-ctx.Done():
				return nil, nil, ErcCtxDone
			default:
				// continue
			}
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
			select {
			case <-ctx.Done():
				return nil, nil, ErcCtxDone
			default:
				// continue
			}
			if a, ok := precomputed[r.Key()]; ok {
				return a, b, nil
			}
			rX, rY := curve.Add(r.X, r.Y, s.X, s.Y) // Q - b*mP; remember, that s == -mP
			r = Point{X: rX, Y: rY}
		}
	}
	return nil, nil, fmt.Errorf("failed to find a and b coefficients")
}

func giantStepsTasks(sqrtN *big.Int, threads int) []giantStepsTask {
	bigIntThreads := big.NewInt(int64(threads))
	perThread := new(big.Int).Div(sqrtN, bigIntThreads)

	start := new(big.Int).Set(zero)
	end := new(big.Int).Sub(sqrtN, new(big.Int).Mul(perThread, bigIntThreads))
	end.Add(end, perThread)

	tasks := make([]giantStepsTask, threads)

	for i := 0; i < threads; i++ {
		tasks[i] = giantStepsTask{
			taskNum:    i + 1,
			sqrtNStart: new(big.Int).Set(start),
			sqrtNEnd:   new(big.Int).Set(end),
		}
		start.Add(start, perThread)
		end.Add(end, perThread)
	}
	return tasks
}

func negPoint(point Point, p *big.Int) Point {
	if point.IsZero() {
		return point
	}
	y := new(big.Int).Set(point.Y)
	point.Y = y.Neg(y).Mod(y, p)
	return point
}
