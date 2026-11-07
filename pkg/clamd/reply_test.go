package clamd

import (
	"errors"
	"testing"
)

// The replies clamd's INSTREAM sends, including the ones the compose clamd cannot be made to send.
func TestParseScanReply(t *testing.T) {
	tests := []struct {
		wantErr error
		reply   string
		name    string
		want    Result
		anyErr  bool
	}{
		{name: "clean", reply: "stream: OK", want: Result{}},
		{name: "found", reply: "stream: Win.Test.EICAR_HDB-1 FOUND", want: Result{Found: true, Signature: "Win.Test.EICAR_HDB-1"}},
		{name: "stream limit", reply: "INSTREAM size limit exceeded. ERROR", wantErr: ErrStreamLimit},
		{name: "other error", reply: "stream: Can't allocate memory ERROR", anyErr: true},
		{name: "found without a name", reply: "stream:  FOUND", anyErr: true},
		{name: "unknown", reply: "UNKNOWN COMMAND", anyErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseScanReply(tt.reply)
			switch {
			case tt.wantErr != nil:
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
			case tt.anyErr:
				if err == nil || errors.Is(err, ErrStreamLimit) {
					t.Fatalf("err = %v, want an error other than the limit", err)
				}
			default:
				if err != nil || got != tt.want {
					t.Fatalf("parseScanReply = %+v, %v; want %+v", got, err, tt.want)
				}
			}
		})
	}
}
