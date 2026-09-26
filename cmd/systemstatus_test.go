package cmd

import (
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestSystemStatusShow(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	golden(t, "systemstatus", mustRun(t, r, "system", "status").Stdout)
	golden(t, "systemstatus_json", mustRun(t, r, "system", "status", "-o", "json").Stdout)
}
