package git

import (
	"errors"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"code.gitea.io/sdk/gitea"
)

var (
	ErrGitClientNil         = errors.New("git client is nil")
	ErrGitCreateAccessToken = errors.New("failed to create access token")
	ErrGitCreateRepo        = errors.New("failed to create repo")
)

type Git struct {
	l    log.Logger
	URL  string
	user string

	client *gitea.Client
	token  *gitea.AccessToken
}

func New(logger *log.Logger, url string, user string, pass string) (*Git, error) {
	client, err := gitea.NewClient(url, gitea.SetBasicAuth(user, pass))
	if err != nil {
		logger.Printf("failed to create git client for new user: %v", err)
		return nil, err
	}
	return &Git{
		URL:    url,
		user:   user,
		client: client,
	}, nil
}

func (g *Git) GetToken() error {
	if g.client == nil {
		log.Error(ErrGitClientNil)
		return ErrGitClientNil
	}

	t, resp, err := g.client.CreateAccessToken(gitea.CreateAccessTokenOption{
		Name: g.user,
	})
	if err != nil || resp.StatusCode != http.StatusCreated {
		g.l.Error(ErrGitCreateAccessToken)
		return ErrGitCreateAccessToken
	}

	g.token = t

	return nil
}

func (g *Git) CreateRepo(name string) (string, error) {
	if g.client == nil {
		g.l.Error(ErrGitClientNil)
		return "", ErrGitClientNil
	}

	r, resp, err := g.client.CreateRepo(gitea.CreateRepoOption{
		Name: name,
	})
	if err != nil || resp.StatusCode != http.StatusCreated {
		err = fmt.Errorf("%w", ErrGitCreateRepo)
		g.l.Error(err)
		return "", err
	}

	return g.URL + "/" + r.FullName, nil
}
