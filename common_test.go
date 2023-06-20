package liner

import (
	"reflect"
	"testing"
)

func TestState_getHistory(t *testing.T) {
	tests := []struct {
		name        string
		historyMode HistoryMode
		line        string
		history     []string
		want        []string
	}{
		{
			name:    "no specified mode uses default prefix matching mode",
			line:    "foo",
			history: []string{"food", "foot", "tool"},
			want:    []string{"food", "foot"},
		},
		{
			name:        "explicit prefix mode matches",
			line:        "foo",
			historyMode: HistoryModePrefix,
			history:     []string{"food", "foot", "tool"},
			want:        []string{"food", "foot"},
		},
		{
			name:        "pattern mode matches history substrings",
			line:        "oo",
			historyMode: HistoryModePattern,
			history:     []string{"food", "foot", "tool"},
			want:        []string{"food", "foot", "tool"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewLiner()
			s.SetHistoryMode(tt.historyMode)
			for _, line := range tt.history {
				s.AppendHistory(line)
			}

			got := s.getHistory(tt.line)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getHistory() got = %v, want %v", got, tt.want)
			}
		})
	}
}
