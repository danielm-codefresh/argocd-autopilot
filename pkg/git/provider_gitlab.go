package git

import (
	"context"
	"fmt"

	gl "github.com/xanzy/go-gitlab"
)

//go:generate mockery --name gitlabClient --output gitlab/mocks --case snake

type (
	gitlabClient interface {
		CurrentUser(options ...gl.RequestOptionFunc) (*gl.User, *gl.Response, error)
		CreateProject(opt *gl.CreateProjectOptions, options ...gl.RequestOptionFunc) (*gl.Project, *gl.Response, error)
		CreateProjectForUser(user int, opt *gl.CreateProjectForUserOptions, options ...gl.RequestOptionFunc) (*gl.Project, *gl.Response, error)
	}

	clientImpl struct {
		gl.ProjectsService
		gl.UsersService
	}

	gitlab struct {
		opts   *ProviderOptions
		client gitlabClient
	}
)

func newGitlab(opts *ProviderOptions) (Provider, error) {
	c, err := gl.NewClient(opts.Auth.Password)
	if err != nil {
		return nil, err
	}

	g := &gitlab{
		opts: opts,
		client: &clientImpl{
			*c.Projects, *c.Users,
		},
	}

	return g, nil
}

func (g *gitlab) CreateRepository(ctx context.Context, opts *CreateRepoOptions) (string, error) {
	authUser, res, err := g.client.CurrentUser()

	if err != nil {
		if res.StatusCode == 401 {
			return "", ErrAuthenticationFailed(err)
		}

		return "", err
	}

	createOpts := gl.CreateProjectOptions{
		Name:       &opts.Name,
		Visibility: gl.Visibility(gl.PublicVisibility),
	}

	if opts.Private {
		createOpts.Visibility = gl.Visibility(gl.PrivateVisibility)
	}

	p, res, err := g.client.CreateProject(&createOpts)
	if authUser.Username != opts.Owner {
		p.Owner.Organization = opts.Owner
	}
	if err != nil {
		if res.StatusCode == 404 {
			return "", fmt.Errorf("owner %s not found: %w", opts.Owner, err)
		}

		return "", err
	}

	if p.WebURL == "" {
		return "", fmt.Errorf("project url is empty")
	}

	return p.WebURL, err
}
