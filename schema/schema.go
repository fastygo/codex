// Package schema defines manifest-backed Codex resource kinds.
package schema

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"

	"github.com/fastygo/codex/content"
	"github.com/fastygo/codex/validation"
	"github.com/fastygo/formset"
)

const ManifestDigestPrefix = "codex-manifest/v1:sha256:"

type Resource struct {
	Record     formset.RecordType `json:"record"`
	Taxonomies []string           `json:"taxonomies,omitempty"`
}

type Manifest struct {
	Name      string     `json:"name"`
	Version   string     `json:"version"`
	Resources []Resource `json:"resources"`
}

func (manifest Manifest) Validate() error {
	if strings.TrimSpace(manifest.Name) == "" {
		return validation.New("schema.manifest.name_required", "name", "manifest name is required")
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return validation.New("schema.manifest.version_required", "version", "manifest version is required")
	}

	records := make([]formset.RecordType, 0, len(manifest.Resources))
	resourceIDs := make(map[formset.RecordTypeID]struct{}, len(manifest.Resources))
	relations := map[formset.RelationID]formset.Relation{}
	for _, resource := range manifest.Resources {
		if err := resource.Validate(); err != nil {
			return validation.Wrap("schema.manifest.resource_invalid", string(resource.Record.ID), err)
		}
		if _, exists := resourceIDs[resource.Record.ID]; exists {
			return validation.New("schema.manifest.resource_duplicated", string(resource.Record.ID), "resource identifier is duplicated")
		}
		resourceIDs[resource.Record.ID] = struct{}{}
		records = append(records, resource.Record)
		for _, relation := range resource.Record.Relations {
			if _, exists := relations[relation.ID]; exists {
				return validation.New("schema.manifest.relation_duplicated", string(relation.ID), "relation identifier is duplicated")
			}
			relations[relation.ID] = relation
		}
	}

	report := formset.ReviewSchema(records, nil)
	if report.HasErrors() {
		return validation.New("schema.manifest.formset_invalid", "resources", report.Summary())
	}
	for _, resource := range manifest.Resources {
		if err := validateRelationFields(resource, relations); err != nil {
			return err
		}
	}
	return nil
}

func (resource Resource) Validate() error {
	if !content.ValidIdentifier(string(resource.Record.ID)) {
		return validation.New("schema.resource.id_invalid", "record.id", "resource identifier is invalid")
	}
	if err := resource.Record.Validate(); err != nil {
		return validation.Wrap("schema.resource.formset_invalid", "record", err)
	}
	for _, field := range resource.Record.Fields {
		if isEntryChromeField(field.ID) && !field.Localized {
			return validation.New(
				"schema.field.chrome_not_localized",
				string(field.ID),
				"entry chrome field must be localized",
			)
		}
		if err := ValidateFieldProfile(field); err != nil {
			return err
		}
	}
	capabilities := make(map[formset.CapabilityID]struct{}, len(resource.Record.Capabilities))
	for _, capability := range resource.Record.Capabilities {
		if strings.TrimSpace(string(capability)) == "" {
			return validation.New("schema.resource.capability_invalid", "record.capabilities", "resource capability is invalid")
		}
		if _, exists := capabilities[capability]; exists {
			return validation.New("schema.resource.capability_duplicated", string(capability), "resource capability is duplicated")
		}
		capabilities[capability] = struct{}{}
	}
	taxonomies := make(map[string]struct{}, len(resource.Taxonomies))
	for _, taxonomy := range resource.Taxonomies {
		if !content.ValidIdentifier(taxonomy) {
			return validation.New("schema.resource.taxonomy_invalid", taxonomy, "taxonomy identifier is invalid")
		}
		if _, exists := taxonomies[taxonomy]; exists {
			return validation.New("schema.resource.taxonomy_duplicated", taxonomy, "resource taxonomy is duplicated")
		}
		taxonomies[taxonomy] = struct{}{}
	}
	for _, relation := range resource.Record.Relations {
		allowedTargets := make(map[string]struct{}, len(relation.Policy.AllowedTargets))
		for _, target := range relation.Policy.AllowedTargets {
			if strings.TrimSpace(target) == "" {
				return validation.New("schema.relation.allowed_target_invalid", string(relation.ID), "relation allowed target is invalid")
			}
			if _, exists := allowedTargets[target]; exists {
				return validation.New("schema.relation.allowed_target_duplicated", string(relation.ID), "relation allowed target is duplicated")
			}
			allowedTargets[target] = struct{}{}
		}
	}
	relations := make(map[formset.RelationID]formset.Relation, len(resource.Record.Relations))
	for _, relation := range resource.Record.Relations {
		if _, exists := relations[relation.ID]; exists {
			return validation.New("schema.resource.relation_duplicated", string(relation.ID), "relation identifier is duplicated")
		}
		relations[relation.ID] = relation
	}
	return validateRelationFields(resource, relations)
}

func isEntryChromeField(fieldID formset.FieldID) bool {
	switch fieldID {
	case "slug", "title", "content", "excerpt":
		return true
	default:
		return false
	}
}

func (manifest Manifest) Resource(kind content.Kind) (Resource, bool) {
	for _, resource := range manifest.Resources {
		if resource.Record.ID == formset.RecordTypeID(kind) {
			return resource.Clone(), true
		}
	}
	return Resource{}, false
}

// Canonical returns a clone with deterministic ordering for set-like values.
// Field and option order is preserved because FormSet renderers may use it.
func (manifest Manifest) Canonical() Manifest {
	canonical := manifest.Clone()
	slices.SortFunc(canonical.Resources, func(left, right Resource) int {
		return strings.Compare(string(left.Record.ID), string(right.Record.ID))
	})
	for index := range canonical.Resources {
		resource := &canonical.Resources[index]
		slices.Sort(resource.Taxonomies)
		slices.Sort(resource.Record.Capabilities)
		slices.SortFunc(resource.Record.Relations, func(left, right formset.Relation) int {
			return strings.Compare(string(left.ID), string(right.ID))
		})
		for relationIndex := range resource.Record.Relations {
			slices.Sort(resource.Record.Relations[relationIndex].Policy.AllowedTargets)
		}
	}
	return canonical
}

func (manifest Manifest) Digest() (string, error) {
	if err := manifest.Validate(); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(manifest.Canonical())
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return ManifestDigestPrefix + hex.EncodeToString(sum[:]), nil
}

func (manifest Manifest) Clone() Manifest {
	cloned := manifest
	cloned.Resources = make([]Resource, len(manifest.Resources))
	for index, resource := range manifest.Resources {
		cloned.Resources[index] = resource.Clone()
	}
	return cloned
}

func (resource Resource) Clone() Resource {
	cloned := resource
	cloned.Record = resource.Record
	cloned.Record.Fields = cloneFields(resource.Record.Fields)
	cloned.Record.Relations = cloneRelations(resource.Record.Relations)
	cloned.Record.Capabilities = append([]formset.CapabilityID(nil), resource.Record.Capabilities...)
	cloned.Taxonomies = append([]string(nil), resource.Taxonomies...)
	return cloned
}

func cloneFields(fields []formset.Field) []formset.Field {
	if fields == nil {
		return nil
	}
	cloned := make([]formset.Field, len(fields))
	for index, field := range fields {
		cloned[index] = cloneField(field)
	}
	return cloned
}

func cloneField(field formset.Field) formset.Field {
	cloned := field
	cloned.Options = append([]formset.Option(nil), field.Options...)
	if field.Rules != nil {
		cloned.Rules = make([]formset.ValidationRule, len(field.Rules))
		for index, rule := range field.Rules {
			cloned.Rules[index] = rule
			if rule.Args != nil {
				cloned.Rules[index].Args = make(map[string]string, len(rule.Args))
				for key, value := range rule.Args {
					cloned.Rules[index].Args[key] = value
				}
			}
		}
	}
	if field.Items != nil {
		item := cloneField(*field.Items)
		cloned.Items = &item
	}
	cloned.Fields = cloneFields(field.Fields)
	return cloned
}

func cloneRelations(relations []formset.Relation) []formset.Relation {
	if relations == nil {
		return nil
	}
	cloned := make([]formset.Relation, len(relations))
	copy(cloned, relations)
	for index, relation := range relations {
		cloned[index].Policy.AllowedTargets = append([]string(nil), relation.Policy.AllowedTargets...)
	}
	return cloned
}

func validateRelationFields(resource Resource, relations map[formset.RelationID]formset.Relation) error {
	fields := make(map[formset.FieldID]formset.Field, len(resource.Record.Fields))
	for _, field := range resource.Record.Fields {
		fields[field.ID] = field
	}
	for _, relation := range resource.Record.Relations {
		if relation.Source != resource.Record.ID {
			return validation.New("schema.relation.source_mismatch", string(relation.ID), "relation source does not match owning resource")
		}
		field, exists := fields[formset.FieldID(relation.ID)]
		if !exists {
			return validation.New("schema.relation.field_missing", string(relation.ID), "relation requires a field with the same id")
		}
		if field.Type != formset.FieldRelation {
			return validation.New("schema.relation.field_type", string(relation.ID), "relation field has invalid type")
		}
		if field.Localized {
			return validation.New("schema.relation.localized", string(relation.ID), "relation field cannot be localized")
		}
	}
	for _, field := range resource.Record.Fields {
		if field.Type != formset.FieldRelation {
			continue
		}
		relation, exists := relations[formset.RelationID(field.ID)]
		if !exists {
			return validation.New("schema.relation.missing", string(field.ID), "relation field has no relation")
		}
		if relation.Source != resource.Record.ID {
			return validation.New("schema.relation.source_mismatch", string(field.ID), "relation field source does not match resource")
		}
	}
	return nil
}
