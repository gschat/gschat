package gschat

import (
	"testing"

	"github.com/gsdocker/gsactor"
)

var agentsysm = gsactor.BuildAgentSystem(NewIMServer())

func TestLogin(t *testing.T) {

	context, err := agentsysm.Run("test")

	if err != nil {
		t.Fatal(err)
	}

	defer context.Close()
}
