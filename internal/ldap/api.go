package ldap

import (
	"errors"
	"fmt"
	appv1 "github.com/vbouchaud/wellerman/api/v1"
	"reflect"
)

func (s *Client) ReconcileGroup(team *appv1.Team) (string, error, bool) {
	groupDN := fmt.Sprintf("%s=%s,%s", s.groupNameProperty, team.Name, s.groupSearchBase)

	exists, err, entry := s.groupExists(team.Name)
	if err != nil {
		return groupDN, err, false
	}

	if exists {
		changed := entry.GetAttributeValue(description) != team.Spec.Comment || !reflect.DeepEqual(sanitize(entry.GetAttributeValues(uniqueMember)), sanitize(team.Spec.Subjects))
		if changed {
			return groupDN, s.modifyGroup(groupDN, team.Spec.Comment, team.Spec.Subjects), true
		}
		return groupDN, nil, false
	}

	return groupDN, s.createGroup(groupDN, team.Spec.Comment, team.Spec.Subjects), true
}

func (s *Client) DeleteGroup(name string) error {
	groupDN := fmt.Sprintf("%s=%s,%s", s.groupNameProperty, name, s.groupSearchBase)

	exists, err, _ := s.groupExists(name)
	if err != nil {
		return err
	}

	if !exists {
		return errors.New(errGroupNotFound)
	}

	return s.deleteGroup(groupDN)
}

func IsNotFound(err error) bool {
	return err.Error() == errGroupNotFound
}
