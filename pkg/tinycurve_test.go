package pkg

import (
	"math/big"
	"testing"
)

func TestPoint_Key(t *testing.T) {
	p := Point{
		X: big.NewInt(-10),
		Y: big.NewInt('S'),
	}
	expected := []byte{10, 83, 255, 1}
	actual := p.Key()

	if actual != string(expected) {
		t.Fail()
	}
}
