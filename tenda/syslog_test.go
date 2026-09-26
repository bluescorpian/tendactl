package tenda

import (
	"bytes"
	"context"
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestSysLog(t *testing.T) {
	r := tendatest.New(t)
	entries, err := newTestClient(t, r).SysLog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 25 {
		t.Fatalf("len = %d", len(entries))
	}
	if got := entries[0]; got != (SysLogEntry{Index: 1, Time: "2000-01-01 00:00:00", Type: "system", Log: "System Start Success"}) {
		t.Fatalf("entries[0] = %+v", got)
	}
}

func TestSysLogEmpty(t *testing.T) {
	r := tendatest.New(t)
	r.Reply("GetSySLogCfg", 200, `[]`)
	entries, err := newTestClient(t, r).SysLog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if entries == nil || len(entries) != 0 {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestDownloadSysLog(t *testing.T) {
	r := tendatest.New(t)
	var buf bytes.Buffer
	n, err := newTestClient(t, r).DownloadSysLog(context.Background(), &buf)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(tendatest.SyslogBytes)) || !bytes.Equal(buf.Bytes(), tendatest.SyslogBytes) {
		t.Fatalf("n = %d, body = %q", n, buf.Bytes())
	}
}
