package tmux

import "testing"

func TestTranslateShiftEnter(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  []byte
	}{
		{
			name:  "Shift+Enter CSI u replaced with newline",
			input: []byte("\x1b[13;2u"),
			want:  []byte("\n"),
		},
		{
			name:  "plain Enter untouched",
			input: []byte("\r"),
			want:  []byte("\r"),
		},
		{
			name:  "text around Shift+Enter",
			input: []byte("hello\x1b[13;2uworld"),
			want:  []byte("hello\nworld"),
		},
		{
			name:  "no CSI u passes through",
			input: []byte("just text"),
			want:  []byte("just text"),
		},
		{
			name:  "multiple Shift+Enter in one chunk",
			input: []byte("a\x1b[13;2ub\x1b[13;2uc"),
			want:  []byte("a\nb\nc"),
		},
		{
			name:  "other CSI u sequences untouched",
			input: []byte("\x1b[109;2u"),
			want:  []byte("\x1b[109;2u"),
		},
		{
			name:  "unmodified Enter CSI u untouched",
			input: []byte("\x1b[13u"),
			want:  []byte("\x1b[13u"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TranslateShiftEnter(tt.input)
			if string(got) != string(tt.want) {
				t.Errorf("TranslateShiftEnter(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTranslateEsc(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  []byte
	}{
		{
			name:  "ESC CSI u replaced with plain ESC",
			input: []byte("\x1b[27u"),
			want:  []byte("\x1b"),
		},
		{
			name:  "plain ESC untouched",
			input: []byte("\x1b"),
			want:  []byte("\x1b"),
		},
		{
			name:  "text around ESC CSI u",
			input: []byte("hello\x1b[27uworld"),
			want:  []byte("hello\x1bworld"),
		},
		{
			name:  "no ESC CSI u passes through",
			input: []byte("just text"),
			want:  []byte("just text"),
		},
		{
			name:  "multiple ESC CSI u in one chunk",
			input: []byte("\x1b[27ua\x1b[27u"),
			want:  []byte("\x1ba\x1b"),
		},
		{
			name:  "other CSI u sequences untouched",
			input: []byte("\x1b[109;2u"),
			want:  []byte("\x1b[109;2u"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TranslateEsc(tt.input)
			if string(got) != string(tt.want) {
				t.Errorf("TranslateEsc(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
