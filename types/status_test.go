package types

import "testing"

func TestCommonStatus_IsHealthy(t *testing.T) {
	tests := []struct {
		status *CommonStatus
		name   string
		want   bool
	}{
		{
			name: "default is healthy",
			status: &CommonStatus{
				ID: 0,
			},
			want: true,
		},
		{
			name: "with raw fields",
			status: &CommonStatus{
				ID:        1,
				RawFields: make(RawFields),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsHealthy(); got != tt.want {
				t.Errorf("CommonStatus.IsHealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}
