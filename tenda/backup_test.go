package tenda

import (
	"bytes"
	"context"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestDownloadBackup(t *testing.T) {
	r := tendatest.New(t)
	var buf bytes.Buffer
	n, err := newTestClient(t, r).DownloadBackup(context.Background(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(tendatest.ConfigBytes)) || !bytes.Equal(buf.Bytes(), tendatest.ConfigBytes) {
		t.Fatalf("n = %d, body = %q", n, buf.Bytes())
	}
}
