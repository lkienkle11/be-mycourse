package infra

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	taxpkg "mycourse-io-be/pkg/taxonomy"
)

type treeNodesJSONB []taxpkg.TreeNode

func (m treeNodesJSONB) Value() (driver.Value, error) {
	if m == nil {
		return "[]", nil
	}
	b, err := json.Marshal([]taxpkg.TreeNode(m))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (m *treeNodesJSONB) Scan(src any) error {
	if src == nil {
		*m = treeNodesJSONB{}
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("unsupported jsonb type %T", src)
	}
	if len(b) == 0 || string(b) == "null" {
		*m = treeNodesJSONB{}
		return nil
	}
	var out []taxpkg.TreeNode
	if err := json.Unmarshal(b, &out); err != nil {
		return err
	}
	*m = treeNodesJSONB(out)
	return nil
}

type descriptionJSONB []string

func (m descriptionJSONB) Value() (driver.Value, error) {
	if m == nil {
		return "[]", nil
	}
	b, err := json.Marshal([]string(m))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (m *descriptionJSONB) Scan(src any) error {
	if src == nil {
		*m = descriptionJSONB{}
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("unsupported jsonb type %T", src)
	}
	if len(b) == 0 || string(b) == "null" {
		*m = descriptionJSONB{}
		return nil
	}
	var out []string
	if err := json.Unmarshal(b, &out); err != nil {
		return err
	}
	*m = descriptionJSONB(out)
	return nil
}
