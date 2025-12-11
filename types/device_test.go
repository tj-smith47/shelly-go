package types

import "testing"

func TestDeviceType_String(t *testing.T) {
	tests := []struct {
		name string
		dt   DeviceType
		want string
	}{
		{
			name: "switch",
			dt:   DeviceTypeSwitch,
			want: "switch",
		},
		{
			name: "cover",
			dt:   DeviceTypeCover,
			want: "cover",
		},
		{
			name: "light",
			dt:   DeviceTypeLight,
			want: "light",
		},
		{
			name: "dimmer",
			dt:   DeviceTypeDimmer,
			want: "dimmer",
		},
		{
			name: "plug",
			dt:   DeviceTypePlug,
			want: "plug",
		},
		{
			name: "relay",
			dt:   DeviceTypeRelay,
			want: "relay",
		},
		{
			name: "sensor",
			dt:   DeviceTypeSensor,
			want: "sensor",
		},
		{
			name: "power_meter",
			dt:   DeviceTypePowerMeter,
			want: "power_meter",
		},
		{
			name: "gateway",
			dt:   DeviceTypeGateway,
			want: "gateway",
		},
		{
			name: "custom",
			dt:   DeviceType("custom_device"),
			want: "custom_device",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.dt.String(); got != tt.want {
				t.Errorf("DeviceType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
