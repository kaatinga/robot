package job

import "testing"

func Test_resultAction_add(t *testing.T) {
	tests := []struct {
		name   string
		a      resultAction
		result resultAction
		want   resultAction
	}{
		{"add resultUpdated to resultNoAction", resultNoAction, resultUpdated, resultUpdated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.a.add(tt.result); tt.a != tt.want {
				t.Errorf("add() = %v, want %v", tt.a, tt.want)
			}
			t.Logf("tt.a: %b", tt.a)
			t.Logf("tt.result: %b", tt.result)
		})
	}
}
