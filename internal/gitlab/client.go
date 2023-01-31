package gitlab

import (
	"fmt"

	git "github.com/xanzy/go-gitlab"
)

type Client struct {
	gitlabURL string
	token     string
	c         *git.Client
}

func NewInstance(gitlabURL, token string) (*Client, error) {
	client, err := git.NewClient(token, git.WithBaseURL(fmt.Sprintf(`%s/api/v4`, gitlabURL)))
	if err != nil {
		return nil, err
	}

	s := &Client{
		gitlabURL: gitlabURL,
		token:     token,
		c:         client,
	}

	return s, nil
}
