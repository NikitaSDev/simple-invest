package securities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_amortizationsNetPeriod(t *testing.T) {
	baseDate := time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		am         []Amortization
		settleDate time.Time
		want       float64
		wantErr    bool
	}{
		{
			name:       "empty Amortizations",
			am:         []Amortization{},
			settleDate: baseDate,
			want:       0,
			wantErr:    false,
		},

		{
			name: "one Amortization",
			am: []Amortization{
				{
					Amortdate: "2020-01-30",
					Facevalue: 1000,
					ValueRub:  100,
				},
			},
			settleDate: baseDate,
			want:       29,
			wantErr:    false,
		},

		{
			name: "two Amortizations",
			am: []Amortization{
				{
					Amortdate: "2020-01-31",
					Facevalue: 1000,
					ValueRub:  500,
				},
				{
					Amortdate: "2020-03-01",
					Facevalue: 1000,
					ValueRub:  500,
				},
			},
			settleDate: baseDate,
			want:       45,
			wantErr:    false,
		},

		{
			name: "positive simple case 1",
			am: []Amortization{
				{
					Amortdate: "2020-02-15",
					Facevalue: 1000,
					ValueRub:  200,
				},
				{
					Amortdate: "2020-03-16",
					Facevalue: 1000,
					ValueRub:  200,
				},
				{
					Amortdate: "2020-04-15",
					Facevalue: 1000,
					ValueRub:  200,
				},
				{
					Amortdate: "2020-05-15",
					Facevalue: 1000,
					ValueRub:  200,
				},
				{
					Amortdate: "2020-06-14",
					Facevalue: 1000,
					ValueRub:  200,
				},
			},
			settleDate: baseDate,
			want:       105,
			wantErr:    false,
		},

		{
			name: "positive simple case 2",
			am: []Amortization{
				{
					Amortdate: "2022-06-09",
					Facevalue: 800,
					ValueRub:  200,
				},
				{
					Amortdate: "2023-12-07",
					Facevalue: 800,
					ValueRub:  150,
				},
				{
					Amortdate: "2024-09-05",
					Facevalue: 800,
					ValueRub:  100,
				},
				{
					Amortdate: "2025-06-05",
					Facevalue: 800,
					ValueRub:  200,
				},
				{
					Amortdate: "2026-06-04",
					Facevalue: 800,
					ValueRub:  200,
				},
			},
			settleDate: time.Date(2020, time.November, 6, 0, 0, 0, 0, time.UTC),
			want:       1330.75,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := amortizationsNetPeriod(tt.am, tt.settleDate)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
