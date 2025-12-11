package types

import "testing"

func TestCommonConfig_Validate(t *testing.T) {
	tests := []struct {
		config  *CommonConfig
		name    string
		wantErr bool
	}{
		{
			name: "valid config",
			config: &CommonConfig{
				ID: 0,
			},
			wantErr: false,
		},
		{
			name: "valid config with name",
			config: &CommonConfig{
				ID:   1,
				Name: stringPtr("test"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CommonConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
