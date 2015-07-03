package changelist

import (
	"encoding/json"

	"github.com/endophage/gotuf/data"
)

type tufChange struct {
	Actn       string `json:"action"`
	Role       string `json:"role"`
	ChangeType string `json:"type"`
	ChangePath string `json:"path"`
	Data       []byte `json:"data"`
}

func NewTufChange(action, role, changeType, changePath string, content []byte) *tufChange {
	return &tufChange{
		Actn:       action,
		Role:       role,
		ChangeType: changeType,
		ChangePath: changePath,
		Data:       content,
	}
}

func NewTufTargetChange(action, target string, hash map[string]data.HexBytes, size int64, custom map[string]interface{}) (*tufChange, error) {
	jsonCustom, err := json.Marshal(custom)
	if err != nil {
		return nil, err
	}
	customRaw := json.RawMessage{}
	err = customRaw.UnmarshalJSON(jsonCustom)
	if err != nil {
		return nil, err
	}
	content := &data.FileMeta{
		Length: size,
		Hashes: hash,
		Custom: &customRaw,
	}
	jsonContent, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	return &tufChange{
		Actn:       action,
		Role:       "targets",
		ChangeType: "target",
		ChangePath: target,
		Data:       jsonContent,
	}, nil
}

func (c tufChange) Action() string {
	return c.Actn
}

func (c tufChange) Scope() string {
	return c.Role
}

func (c tufChange) Type() string {
	return c.ChangeType
}

func (c tufChange) Path() string {
	return c.ChangePath
}

func (c tufChange) Content() []byte {
	return c.Data
}
