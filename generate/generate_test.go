package generate

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNumericToFloatDomainParams(t *testing.T) {
	const eps = 0.0001

	tests := []struct {
		precision int
		scale     int
		wantTop   float64
		wantStep  float64
	}{
		{
			0, 0,
			10.0, 0.1,
		},
		{
			3, 0,
			999.0, 1,
		},
		{
			3, 2,
			9.99, 0.01,
		},
		{
			3, 3,
			0.999, 0.001,
		},
		{
			3, 4,
			0.0999, 0.0001,
		},
		{
			1, -1,
			90, 10,
		},
		{
			5, -1,
			999990, 10,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("precision:%d,scale:%d", tt.precision, tt.scale),
			func(t *testing.T) {
				top, step := NumericToFloatDomainParams(tt.precision, tt.scale)
				assert.InDelta(t, tt.wantTop, top, eps)
				assert.InDelta(t, tt.wantStep, step, eps)
			})
	}
}
