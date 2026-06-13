package config

import (
	"bytes"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

const ProjectFileVersion = 1

// ProjectFile is the declarative, persisted representation of a project.
type ProjectFile struct {
	Version       int `yaml:"version"`
	ProjectConfig `yaml:",inline"`
}

// ParseProjectFile decodes a project file and rejects unknown fields so that
// misspelled configuration cannot silently change the generated environment.
func ParseProjectFile(data []byte) (ProjectConfig, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var file ProjectFile
	if err := decoder.Decode(&file); err != nil {
		return ProjectConfig{}, fmt.Errorf("decoding project configuration: %w", err)
	}
	if file.Version != ProjectFileVersion {
		return ProjectConfig{}, fmt.Errorf("unsupported project configuration version %d (supported: %d)", file.Version, ProjectFileVersion)
	}
	if err := ensureSingleYAMLDocument(decoder); err != nil {
		return ProjectConfig{}, err
	}
	if err := file.ProjectConfig.Validate(); err != nil {
		return ProjectConfig{}, fmt.Errorf("validating project configuration: %w", err)
	}
	return file.ProjectConfig, nil
}

func ensureSingleYAMLDocument(decoder *yaml.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decoding project configuration: %w", err)
	}
	return fmt.Errorf("project configuration must contain exactly one YAML document")
}

// RenderProjectFile returns a stable YAML representation of the selected
// stack. Runtime-only options such as Force are intentionally omitted.
func RenderProjectFile(cfg ProjectConfig) (string, error) {
	cfg.Force = false
	data, err := yaml.Marshal(ProjectFile{Version: ProjectFileVersion, ProjectConfig: cfg})
	if err != nil {
		return "", fmt.Errorf("encoding project configuration: %w", err)
	}
	return string(data), nil
}
