// Package gconfig loads conf/config.yaml merged with conf/application-<env>.yaml.
package gconfig

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// Load reads dir/config.yaml, deep-merges dir/application-<environment>.yaml over it,
// and decodes the result into out, which must be a pointer to a struct with yaml tags.
//
// The base file is required; the environment file is optional. ${VAR} references in
// either file are replaced with environment variables before parsing. Keys in the files
// that have no matching struct field are an error, so typos fail loudly.
func Load(dir, environment string, out any) error {
	base, err := readFile(filepath.Join(dir, "config.yaml"), true)
	if err != nil {
		return err
	}
	override, err := readFile(filepath.Join(dir, "application-"+environment+".yaml"), false)
	if err != nil {
		return err
	}

	merged, err := yaml.Marshal(merge(base, override))
	if err != nil {
		return fmt.Errorf("gconfig: encode merged config: %w", err)
	}

	dec := yaml.NewDecoder(bytes.NewReader(merged))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("gconfig: decode config: %w", err)
	}
	return nil
}

// readFile parses one YAML file into a map. A missing file is an error only when required.
// A file holding only comments yields an empty map.
func readFile(path string, required bool) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) && !required {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gconfig: read %s: %w", path, err)
	}

	m := map[string]any{}
	if err := yaml.Unmarshal([]byte(os.ExpandEnv(string(raw))), &m); err != nil {
		return nil, fmt.Errorf("gconfig: parse %s: %w", path, err)
	}
	return m, nil
}

// merge returns base with override applied on top. Nested maps are merged key by key;
// any other value in override (scalars, lists) replaces the base value.
func merge(base, override map[string]any) map[string]any {
	out := make(map[string]any, len(base))
	for k, v := range base {
		out[k] = v
	}
	for k, ov := range override {
		bm, baseIsMap := out[k].(map[string]any)
		om, overrideIsMap := ov.(map[string]any)
		if baseIsMap && overrideIsMap {
			out[k] = merge(bm, om)
			continue
		}
		out[k] = ov
	}
	return out
}
