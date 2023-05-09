package generate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeDomain(t *testing.T) {
	td := &TimeDomain{}
	td.ResetWith(time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC), 5)

	// Test Next and Value methods for the first five values
	expectedValues := []string{
		"2023-01-03T00:00:00Z",
		"2023-01-04T00:00:01Z",
		"2023-01-01T23:59:59Z", // отнимается один день и одна секунда
		"2023-01-05T00:00:02Z",
		"2022-12-31T23:59:58Z",
	}
	for i, expected := range expectedValues {
		require.True(t, td.Next(), "Expected Next to return true for value %d", i)
		actual := td.Value()
		assert.Equal(t, expected, actual,
			"Expected Value to return %q for value %d, but got %q", expected, i, actual)
	}

	// Test that Next returns false after reaching the top of the domain
	require.False(t, td.Next(), "Expected Next to return false after reaching the top of the domain")

	// Test Reset method
	td.Reset()
	assert.Equal(t, -1, td.index, "Expected Reset to reset the TimeDomain struct")
	assert.Equal(t, 1000, td.top, "Expected Reset to reset the TimeDomain struct")
	assert.False(t, td.value.IsZero(), "Expected Reset to reset the TimeDomain struct")
}

func TestFloatDomain(t *testing.T) {
	fd := &FloatDomain{}
	fd.ResetWith(0.5, 3.0)

	// Test Next and Value methods for the first few values
	expectedValues := []string{
		"0",
		"0.5", "-0.5", "1", "-1",
		"1.5", "-1.5", "2", "-2",
		"2.5", "-2.5", "3", "-3",
	}
	for i, expected := range expectedValues {
		require.True(t, fd.Next(), "Expected Next to return true for value %d", i)
		actual := fd.Value()
		assert.Equal(t, expected, actual,
			"Expected Value to return %q for value %d, but got %q", expected, i, actual)
	}

	// Test that Next returns false after reaching the top of the domain
	require.False(t, fd.Next(), "Expected Next to return false after reaching the top of the domain")

	// Test Reset method
	fd.Reset()
	assert.Equal(t, -1, fd.index, "Expected Reset to reset the FloatDomain struct")
	assert.Equal(t, 10.0, fd.top, "Expected Reset to reset the FloatDomain struct")
	assert.Equal(t, 0.1, fd.step, "Expected Reset to reset the FloatDomain struct")
	assert.Equal(t, 0.0, fd.value, "Expected Reset to reset the FloatDomain struct")
}
