package git

import (
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestGit_GetToken(t *testing.T) {
	g, _ := New(log.StandardLogger(), "http://20.81.14.4:3000", "gitea_admin", "gitea_admin")
	defer g.client.DeleteAccessToken("gitea_admin")

	err := g.GetToken()
	if err != nil {
		t.Error(err)
	}
	if g.token == nil {
		t.Error("token is empty")
	}

	t.Logf("Got Token %#v", g.token)
}

func TestGit_CreateRepo(t *testing.T) {
	g, _ := New(log.StandardLogger(), "http://20.81.14.4:3000", "gitea_admin", "gitea_admin")
	defer g.client.DeleteAccessToken("gitea_admin")
	defer g.client.DeleteRepo("gitea_admin", "sample_app")

	err := g.GetToken()
	if err != nil {
		t.Error(err)
	}
	if g.token == nil {
		t.Error("token is empty")
	}

	r, err := g.CreateRepo("sample_app")
	if err != nil {
		t.Error(err)
	}

	t.Logf("Got Repo %#v", r)
}
