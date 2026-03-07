package service

import "testing"

func TestBizAutoIntervalDuration(t *testing.T) {
	tests := []struct {
		name string
		sec  int
		want string
	}{
		{name: "default", sec: 0, want: "30m"},
		{name: "minute", sec: 900, want: "15m"},
		{name: "hour", sec: 7200, want: "2h"},
		{name: "seconds", sec: 75, want: "75s"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BizAutoIntervalDuration(tc.sec)
			if got != tc.want {
				t.Fatalf("want %q, got %q", tc.want, got)
			}
		})
	}
}

