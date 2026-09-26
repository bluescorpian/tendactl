package tenda

import (
	"context"
	"io"
)

// SysLogEntry is one row of "System Log" (GetSySLogCfg).
type SysLogEntry struct {
	Index int    `json:"index"`
	Time  string `json:"time"`
	Type  string `json:"type"`
	Log   string `json:"log"`
}

type sysLogEntryWire struct {
	Index int    `json:"index"`
	Time  string `json:"time"`
	Type  string `json:"type"`
	Log   string `json:"log"`
}

// SysLog reads GetSySLogCfg, in the order the router sends it. The result is
// never nil.
func (c *Client) SysLog(ctx context.Context) ([]SysLogEntry, error) {
	var w []sysLogEntryWire
	if err := c.get(ctx, "GetSySLogCfg", nil, &w); err != nil {
		return nil, err
	}
	entries := make([]SysLogEntry, len(w))
	for i, e := range w {
		entries[i] = SysLogEntry{Index: e.Index, Time: e.Time, Type: e.Type, Log: e.Log}
	}
	return entries, nil
}

// DownloadSysLog streams the system log archive (cgi-bin/DownloadLog/syslog.tar) to w.
func (c *Client) DownloadSysLog(ctx context.Context, w io.Writer) (int64, error) {
	return c.download(ctx, "cgi-bin/DownloadLog/syslog.tar", w)
}
