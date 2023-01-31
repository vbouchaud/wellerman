package gitlab

import (
	"errors"
	"fmt"
	"path"

	git "github.com/xanzy/go-gitlab"

	appv1 "github.com/vbouchaud/wellerman/api/v1"
)

func findInGroups(p string, groups []*git.Group) *git.Group {
	for _, group := range groups {
		if group.FullPath == p {
			return group
		}
	}
	return nil
}

func (s *Client) FindProjects(p appv1.ProjectPath) (*git.Project, error) {

	projects, _, err := s.c.Projects.ListProjects(&git.ListProjectsOptions{Search: git.String(path.Base(p.Path))})
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not list projects: %s", err.Error()))
	}

	for _, project := range projects {
		if project.PathWithNamespace == p.Path {
			return project, nil
		}
	}

	return nil, nil
}

func (s *Client) ensurePathExists(p string) (int, error) {
	groups, _, err := s.c.Groups.ListGroups(&git.ListGroupsOptions{Search: git.String(p)})
	if err != nil {
		return -1, errors.New(fmt.Sprintf("Could not list group: %s", err.Error()))
	}

	if group := findInGroups(p, groups); group != nil {
		return group.ID, nil
	}

	var (
		newPath  = path.Dir(p)
		parentId = -1
	)

	if newPath != "." {
		parentId, err = s.ensurePathExists(newPath)
		if err != nil {
			return parentId, err
		}
	}

	var group *git.Group

	groupOptions := &git.CreateGroupOptions{
		Name:       git.String(path.Base(p)),
		Path:       git.String(path.Base(p)),
		Visibility: git.Visibility(git.PrivateVisibility),
	}

	if parentId != -1 {
		groupOptions.ParentID = git.Int(parentId)
	}
	group, _, err = s.c.Groups.CreateGroup(groupOptions)

	if err != nil {
		return -1, errors.New(fmt.Sprintf("Could not create group: %s", err.Error()))
	}

	return group.ID, nil
}

func (s *Client) ReconcileProject(p appv1.ProjectPath) (error, bool) {
	project, err := s.FindProjects(p)
	if err != nil {
		return err, false
	}

	if project != nil {
		if project.Name == p.Name && project.Description == p.Description {
			return nil, false
		}

		if _, _, err = s.c.Projects.EditProject(project.ID, &git.EditProjectOptions{
			Name:        git.String(p.Name),
			Description: git.String(p.Description),
		}); err != nil {
			return errors.New(fmt.Sprintf("Could not edit project: %s", err.Error())), false
		}
	} else {
		parentId, err := s.ensurePathExists(path.Dir(p.Path))
		if err != nil {
			return err, false
		}

		if _, _, err = s.c.Projects.CreateProject(&git.CreateProjectOptions{
			Name:        git.String(p.Name),
			Description: git.String(p.Description),
			Visibility:  git.Visibility(git.PrivateVisibility),
			Path:        git.String(path.Base(p.Path)),
			NamespaceID: git.Int(parentId),
		}); err != nil {
			return errors.New(fmt.Sprintf("Could not create project: %s", err.Error())), false
		}
	}

	return nil, true
}

func (s *Client) DeleteProject(p appv1.ProjectPath) (error, bool) {
	project, err := s.FindProjects(p)
	if err != nil {
		return err, false
	}

	if project == nil {
		return errors.New(fmt.Sprintf("Could not find project.")), false
	}

	if p.ArchiveOnDelete {
		_, _, err = s.c.Projects.ArchiveProject(project.ID)
	} else {
		_, err = s.c.Projects.DeleteProject(project.ID)
	}

	return err, true
}
