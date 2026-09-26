package cmd

import (
	"testing"

	"github.com/bluescorpian/tendactl/tenda/tendatest"
)

func TestClientsCmd(t *testing.T) {
	t.Parallel()
	r := tendatest.New(t)
	text := mustRun(t, r, "clients").Stdout
	golden(t, "clients", text)
	for _, args := range [][]string{{"clients", "list"}, {"online"}, {"online", "list"}} {
		if got := mustRun(t, r, args...).Stdout; got != text {
			t.Fatalf("%v output differs from clients:\n%s", args, got)
		}
	}
	golden(t, "clients_json", mustRun(t, r, "clients", "-o", "json").Stdout)
}
