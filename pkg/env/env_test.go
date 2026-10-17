package env

import "testing"

func TestCurrent(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "unset defaults to dev", value: "", want: Dev},
		{name: "known value", value: "prod", want: Prod},
		{name: "unknown value", value: "qa", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(Key, tt.value)

			got, err := Current()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Current() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Current() = %q, want %q", got, tt.want)
			}
		})
	}
}
