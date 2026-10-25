package canonical

import "fmt"

// MANIFEST_SHAPE from @ui8kit/contract src/canonical.ts (commit a70c243),
// matching schemas/manifest.schema.json property names exactly.
type manifestShape struct {
	keys     []string
	required []string
}

var manifestShapes = map[string]manifestShape{
	"manifest": {[]string{"name", "version", "resources"}, []string{"name", "version", "resources"}},
	"resource": {[]string{"record", "taxonomies"}, []string{"record"}},
	"record":   {[]string{"id", "label", "description", "schema_version", "owner_module", "scope", "fields", "relations", "capabilities", "visibility"}, []string{"id", "label", "scope"}},
	"field":    {[]string{"id", "label", "type", "namespace", "owner_module", "description", "required", "localized", "default_value", "options", "rules", "items", "fields", "searchable", "indexed", "unique", "sensitive", "encrypted", "ui_hint", "storage_hint"}, []string{"id", "label", "type"}},
	"option":   {[]string{"value", "label"}, []string{"value", "label"}},
	"rule":     {[]string{"name", "message", "args", "severity"}, []string{"name"}},
	"relation": {[]string{"id", "label", "source", "target", "cardinality", "inverse_name", "cross_workspace_policy", "policy", "delete_behavior"}, []string{"id", "source", "target", "cardinality"}},
	"policy":   {[]string{"cross_workspace_mode", "allowed_targets", "capability", "read_only"}, nil},
}

func shapeObj(v any, shape, path string) (map[string]any, error) {
	o, ok := isObj(v)
	if !ok {
		return nil, fail(ErrInvalidType, path, shape+" must be a JSON object")
	}
	spec := manifestShapes[shape]
	if err := closed(o, spec.keys, path); err != nil {
		return nil, err
	}
	for _, k := range spec.required {
		if !own(o, k) || o[k] == nil {
			return nil, fail(ErrMissingField, path+"."+k, "required field "+k+" is missing")
		}
	}
	return o, nil
}

func shapeArr(v any, path string, each func(any, string) error) error {
	arr, ok := v.([]any)
	if !ok {
		return fail(ErrInvalidType, path, "must be an array")
	}
	if each != nil {
		for n, x := range arr {
			if err := each(x, fmt.Sprintf("%s[%d]", path, n)); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkField(f any, path string) error {
	o, err := shapeObj(f, "field", path)
	if err != nil {
		return err
	}
	if own(o, "options") {
		if err := shapeArr(o["options"], path+".options", func(x any, p string) error {
			_, err := shapeObj(x, "option", p)
			return err
		}); err != nil {
			return err
		}
	}
	if own(o, "rules") {
		if err := shapeArr(o["rules"], path+".rules", func(x any, p string) error {
			r, err := shapeObj(x, "rule", p)
			if err != nil {
				return err
			}
			if own(r, "args") {
				if _, ok := isObj(r["args"]); !ok {
					return fail(ErrInvalidType, p+".args", "args must be an object")
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	if own(o, "items") {
		if err := checkField(o["items"], path+".items"); err != nil {
			return err
		}
	}
	if own(o, "fields") {
		if err := shapeArr(o["fields"], path+".fields", checkField); err != nil {
			return err
		}
	}
	return nil
}

// projectManifest is the Go port of src/canonical.ts projectManifest.
// Nested values are kept as parsed (including empty policy:{} and fields:[]).
// No array is sorted. Empty optionals are omitted only at the top level.
func projectManifest(doc any) (map[string]any, error) {
	m, err := shapeObj(doc, "manifest", "$")
	if err != nil {
		return nil, err
	}
	if err := shapeArr(m["resources"], "$.resources", func(x any, p string) error {
		r, err := shapeObj(x, "resource", p)
		if err != nil {
			return err
		}
		if own(r, "taxonomies") {
			if err := shapeArr(r["taxonomies"], p+".taxonomies", nil); err != nil {
				return err
			}
		}
		rec, err := shapeObj(r["record"], "record", p+".record")
		if err != nil {
			return err
		}
		if own(rec, "fields") {
			if err := shapeArr(rec["fields"], p+".record.fields", checkField); err != nil {
				return err
			}
		}
		if own(rec, "capabilities") {
			if err := shapeArr(rec["capabilities"], p+".record.capabilities", nil); err != nil {
				return err
			}
		}
		if own(rec, "relations") {
			return shapeArr(rec["relations"], p+".record.relations", func(y any, q string) error {
				rel, err := shapeObj(y, "relation", q)
				if err != nil {
					return err
				}
				if own(rel, "policy") {
					pol, err := shapeObj(rel["policy"], "policy", q+".policy")
					if err != nil {
						return err
					}
					if own(pol, "allowed_targets") {
						return shapeArr(pol["allowed_targets"], q+".policy.allowed_targets", nil)
					}
				}
				return nil
			})
		}
		return nil
	}); err != nil {
		return nil, err
	}
	required := make(map[string]struct{}, len(manifestShapes["manifest"].required))
	for _, k := range manifestShapes["manifest"].required {
		required[k] = struct{}{}
	}
	out := map[string]any{}
	for k, v := range m {
		if _, req := required[k]; req || !isEmpty(v) {
			out[k] = v
		}
	}
	return out, nil
}

// CanonicalizeManifest returns the codex-manifest/v2 JCS text of a full manifest JSON document.
// Source array order is significant; Canonical() sorting must not be applied.
func CanonicalizeManifest(inputText string) (string, error) {
	doc, err := parse(inputText)
	if err != nil {
		return "", err
	}
	projected, err := projectManifest(doc)
	if err != nil {
		return "", err
	}
	return jcs(projected), nil
}

// ManifestDigest returns "codex-manifest/v2:sha256:" plus hex(sha256(UTF-8 bytes of CanonicalizeManifest(inputText))).
func ManifestDigest(inputText string) (string, error) {
	text, err := CanonicalizeManifest(inputText)
	if err != nil {
		return "", err
	}
	return digestPrefixed(ManifestDigestV2Prefix, text), nil
}
