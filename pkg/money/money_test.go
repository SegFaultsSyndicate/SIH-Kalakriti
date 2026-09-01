// pkg/money/money_test.go
package money

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMoney_Add(t *testing.T) {
	m := New(1000)
	result := m.Add(New(500))
	assert.Equal(t, int64(1500), result.Paise())
}

func TestMoney_Sub(t *testing.T) {
	m := New(1000)
	result := m.Sub(New(300))
	assert.Equal(t, int64(700), result.Paise())
}

func TestMoney_MulPct(t *testing.T) {
	tests := []struct {
		name     string
		amount   Money
		pct      int
		expected int64
	}{
		{"10% of 1000", New(1000), 10, 100},
		{"25% of 8000", New(8000), 25, 2000},
		{"33% of 100 rounds down", New(100), 33, 33},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.amount.MulPct(tt.pct)
			assert.Equal(t, tt.expected, result.Paise())
		})
	}
}

func TestMoney_Split(t *testing.T) {
	tests := []struct {
		name     string
		amount   Money
		n        int
		expected []int64
	}{
		{
			"1000 split 3 ways",
			New(1000),
			3,
			[]int64{334, 333, 333},
		},
		{
			"100 split 3 ways",
			New(100),
			3,
			[]int64{34, 33, 33},
		},
		{
			"1000 split 1 way",
			New(1000),
			1,
			[]int64{1000},
		},
		{
			"10 split 4 ways",
			New(10),
			4,
			[]int64{3, 3, 2, 2},
		},
		{
			"500 split 2 ways",
			New(500),
			2,
			[]int64{250, 250},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := tt.amount.Split(tt.n)
			assert.Len(t, parts, tt.n)

			// Check each part
			for i, exp := range tt.expected {
				assert.Equal(t, exp, parts[i].Paise(), "part %d", i)
			}

			// Conservation: sum equals original
			sum := int64(0)
			for _, p := range parts {
				sum += p.Paise()
			}
			assert.Equal(t, tt.amount.Paise(), sum, "sum must equal original")
		})
	}
}

func TestMoney_Split_NoPaisaLost(t *testing.T) {
	// Property test: for any amount and split count, no paise is lost
	amounts := []int64{1, 10, 100, 999, 1000, 10000, 123456}
	splits := []int{2, 3, 5, 7, 11}

	for _, amt := range amounts {
		for _, n := range splits {
			m := New(amt)
			parts := m.Split(n)

			sum := int64(0)
			for _, p := range parts {
				sum += p.Paise()
			}

			assert.Equal(t, amt, sum, "Split(%d, %d) lost paise", amt, n)
		}
	}
}

func TestMoney_Format(t *testing.T) {
	tests := []struct {
		paise    int64
		expected string
	}{
		{0, "₹0.00"},
		{100, "₹1.00"},
		{150, "₹1.50"},
		{1234, "₹12.34"},
		{999999, "₹9999.99"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			m := New(tt.paise)
			assert.Equal(t, tt.expected, m.Format())
		})
	}
}

func TestMoney_Split_InvalidInput(t *testing.T) {
	m := New(1000)
	assert.Nil(t, m.Split(0))
	assert.Nil(t, m.Split(-1))
}
