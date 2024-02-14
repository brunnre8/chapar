package domain

type Environment struct {
	ApiVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind"`
	Meta       EnvMeta    `yaml:"meta"`
	Values     []EnvValue `yaml:"values"`

	FilePath string `yaml:"-"`
}

type EnvMeta struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

type EnvValue struct {
	ID     string `yaml:"id"`
	Key    string `yaml:"key"`
	Value  string `yaml:"value"`
	Enable bool   `yaml:"enable"`
}

func CompareEnvValue(a, b EnvValue) bool {
	if a.ID != b.ID {
		return false
	}

	if a.Key != b.Key {
		return false
	}

	if a.Value != b.Value {
		return false
	}

	if a.Enable != b.Enable {
		return false
	}

	return true
}

func (e *Environment) Clone() *Environment {
	clone := &Environment{
		ApiVersion: e.ApiVersion,
		Kind:       e.Kind,
		Meta:       e.Meta,
		Values:     make([]EnvValue, len(e.Values)),
		FilePath:   e.FilePath,
	}

	for i, v := range e.Values {
		clone.Values[i] = v
	}

	return clone
}

func CompareEnvValues(a, b []EnvValue) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if !CompareEnvValue(v, b[i]) {
			return false
		}
	}

	return true
}
