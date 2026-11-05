package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
)

const ViewVersion = 1

type viewFile struct {
	Version int       `json:"version"`
	Spec    QuerySpec `json:"spec"`
}

// SaveView writes q as versioned JSON. Pivot keys are left out: they come
// from the data and are resolved again on load.
func SaveView(path string, q QuerySpec) error {
	q = q.Clone()
	if q.Pivot != nil {
		q.Pivot.Keys, q.Pivot.Null = nil, false
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(viewFile{ViewVersion, q}); err != nil {
		return err
	}
	return os.WriteFile(path, b.Bytes(), 0o644)
}

// LoadView reads a view saved by SaveView and checks it against this source.
func (s *Session) LoadView(ctx context.Context, path string) (QuerySpec, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return QuerySpec{}, fmt.Errorf("cannot open view: %w", err)
	}
	var v viewFile
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return QuerySpec{}, fmt.Errorf("bad view %s: %w", path, err)
	}
	switch {
	case v.Version == 0:
		return QuerySpec{}, fmt.Errorf("bad view %s: missing version", path)
	case v.Version > ViewVersion:
		return QuerySpec{}, fmt.Errorf("view %s has version %d, this pivotlens reads up to %d", path, v.Version, ViewVersion)
	}
	q := v.Spec
	have := make([]string, len(s.cols))
	for i, c := range s.cols {
		have[i] = c.Name
	}
	for _, c := range q.Cols {
		if !slices.Contains(have, c.Name) {
			return QuerySpec{}, fmt.Errorf("view %s: unknown column %q", path, c.Name)
		}
	}
	if len(q.Cols) != len(have) {
		return QuerySpec{}, fmt.Errorf("view %s lists %d columns, the source has %d", path, len(q.Cols), len(have))
	}
	if q.Pivot != nil {
		if q, err = s.ResolvePivot(ctx, q); err != nil {
			return QuerySpec{}, fmt.Errorf("view %s: %w", path, err)
		}
	}
	if _, err := s.Count(ctx, q); err != nil {
		return QuerySpec{}, fmt.Errorf("view %s does not fit this source: %w", path, err)
	}
	return q, nil
}
