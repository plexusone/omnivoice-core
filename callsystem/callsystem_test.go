package callsystem

import "testing"

func TestWithSendDigits(t *testing.T) {
	opts := &CallOptions{}
	WithSendDigits("WW85012345678#")(opts)

	if opts.SendDigits != "WW85012345678#" {
		t.Fatalf("SendDigits = %q, want %q", opts.SendDigits, "WW85012345678#")
	}
}

func TestValidateSendDigits(t *testing.T) {
	tests := []struct {
		name    string
		digits  string
		wantErr bool
	}{
		{name: "empty", digits: ""},
		{name: "meeting ID and passcode", digits: "WW85012345678#ww#w123456#"},
		{name: "all tones", digits: "0123456789*#ABCD"},
		{name: "pauses only", digits: "wW"},
		{name: "space", digits: "123 456#", wantErr: true},
		{name: "comma pause notation", digits: "123,456#", wantErr: true},
		{name: "lowercase tone letter", digits: "a", wantErr: true},
		{name: "letter outside A-D", digits: "E", wantErr: true},
		{name: "plus sign", digits: "+1", wantErr: true},
		{name: "non-ASCII", digits: "１２３", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSendDigits(tt.digits)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateSendDigits(%q) error = %v, wantErr %v", tt.digits, err, tt.wantErr)
			}
		})
	}
}
