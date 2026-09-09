// Package schema defines manifest-backed Codex resource kinds.
package schema

import (
	"errors"
	"slices"
	"strings"

	"github.com/fastygo/codex/content"
	"github.com/fastygo/formset"
)

type Resource struct {
	formset.RecordType `json:"record_type"`
	Collection         string   `json:"collection"`
	Taxonomies         []string `json:"taxonomies,omitempty"`
	Public             bool     `json:"public,omitempty"`
}

type Manifest struct {
	Name      string     `json:"name"`
	Version   string     `json:"version"`
	Resources []Resource `json:"resources"`
}

func CoreResources() []Resource {
	documentFields := []formset.Field{
		{ID: "title", Label: "Title", Type: formset.FieldString, Localized: true, Searchable: true},
		{ID: "excerpt", Label: "Excerpt", Type: formset.FieldText, Localized: true, Searchable: true},
		{ID: "content", Label: "Content", Type: formset.FieldText, Localized: true, Searchable: true},
	}
	return []Resource{
		newCoreResource(content.KindPost, "Posts", "posts", cloneFields(documentFields)),
		newCoreResource(content.KindPage, "Pages", "pages", cloneFields(documentFields)),
		newCoreResource(content.KindMenu, "Menus", "menus", []formset.Field{
			{ID: "items", Label: "Items", Type: formset.FieldJSON},
		}),
		newCoreResource(content.KindSetting, "Settings", "settings", []formset.Field{
			{ID: "value", Label: "Value", Type: formset.FieldJSON},
		}),
	}
}

func WithCoreResources(manifest Manifest) Manifest {
	merged := manifest.Clone()
	seen := make(map[formset.RecordTypeID]struct{}, len(merged.Resources))
	for _, resource := range merged.Resources {
		seen[resource.ID] = struct{}{}
	}
	for _, resource := range CoreResources() {
		if _, exists := seen[resource.ID]; !exists {
			merged.Resources = append(merged.Resources, resource)
		}
	}
	return merged
}

func (manifest Manifest) Validate() error {
	if strings.TrimSpace(manifest.Name) == "" {
		return errors.New("manifest name is required")
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return errors.New("manifest version is required")
	}

	records := make([]formset.RecordType, 0, len(manifest.Resources))
	resourceIDs := make(map[formset.RecordTypeID]struct{}, len(manifest.Resources))
	collections := make(map[string]struct{}, len(manifest.Resources))
	relationIDs := map[formset.RelationID]struct{}{}
	for _, resource := range manifest.Resources {
		if err := resource.Validate(); err != nil {
			return err
		}
		if _, exists := resourceIDs[resource.ID]; exists {
			return errors.New("resource identifier is duplicated")
		}
		if _, exists := collections[resource.Collection]; exists {
			return errors.New("resource collection is duplicated")
		}
		resourceIDs[resource.ID] = struct{}{}
		collections[resource.Collection] = struct{}{}
		records = append(records, resource.RecordType)
		for _, relation := range resource.Relations {
			if _, exists := relationIDs[relation.ID]; exists {
				return errors.New("relation identifier is duplicated")
			}
			relationIDs[relation.ID] = struct{}{}
		}
	}

	report := formset.ReviewSchema(records, nil)
	if report.HasErrors() {
		return errors.New(report.Summary())
	}
	return nil
}

func (resource Resource) Validate() error {
	if !content.ValidIdentifier(string(resource.ID)) {
		return errors.New("resource identifier is invalid")
	}
	if !content.ValidIdentifier(resource.Collection) {
		return errors.New("resource collection is invalid")
	}
	if err := resource.RecordType.Validate(); err != nil {
		return err
	}
	capabilities := make(map[formset.CapabilityID]struct{}, len(resource.Capabilities))
	for _, capability := range resource.Capabilities {
		if strings.TrimSpace(string(capability)) == "" {
			return errors.New("resource capability is invalid")
		}
		if _, exists := capabilities[capability]; exists {
			return errors.New("resource capability is duplicated")
		}
		capabilities[capability] = struct{}{}
	}
	taxonomies := make(map[string]struct{}, len(resource.Taxonomies))
	for _, taxonomy := range resource.Taxonomies {
		if !content.ValidIdentifier(taxonomy) {
			return errors.New("taxonomy identifier is invalid")
		}
		if _, exists := taxonomies[taxonomy]; exists {
			return errors.New("resource taxonomy is duplicated")
		}
		taxonomies[taxonomy] = struct{}{}
	}
	for _, relation := range resource.Relations {
		allowedTargets := make(map[string]struct{}, len(relation.Policy.AllowedTargets))
		for _, target := range relation.Policy.AllowedTargets {
			if strings.TrimSpace(target) == "" {
				return errors.New("relation allowed target is invalid")
			}
			if _, exists := allowedTargets[target]; exists {
				return errors.New("relation allowed target is duplicated")
			}
			allowedTargets[target] = struct{}{}
		}
	}
	if resource.Collection == "menus" && resource.ID != formset.RecordTypeID(content.KindMenu) {
		return errors.New("menus collection is reserved")
	}
	if resource.Collection == "settings" && resource.ID != formset.RecordTypeID(content.KindSetting) {
		return errors.New("settings collection is reserved")
	}
	return nil
}

func (manifest Manifest) Resource(kind content.Kind) (Resource, bool) {
	for _, resource := range manifest.Resources {
		if resource.ID == formset.RecordTypeID(kind) {
			return resource.Clone(), true
		}
	}
	return Resource{}, false
}

// Canonical returns a clone with deterministic ordering for set-like values.
// Field order is preserved because FormSet uses it as presentation order.
func (manifest Manifest) Canonical() Manifest {
	canonical := manifest.Clone()
	slices.SortFunc(canonical.Resources, func(left, right Resource) int {
		return strings.Compare(string(left.ID), string(right.ID))
	})
	for index := range canonical.Resources {
		resource := &canonical.Resources[index]
		slices.Sort(resource.Taxonomies)
		slices.Sort(resource.Capabilities)
		slices.SortFunc(resource.Relations, func(left, right formset.Relation) int {
			return strings.Compare(string(left.ID), string(right.ID))
		})
		for relationIndex := range resource.Relations {
			slices.Sort(resource.Relations[relationIndex].Policy.AllowedTargets)
		}
	}
	return canonical
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
	cloned.Fields = cloneFields(resource.Fields)
	cloned.Relations = cloneRelations(resource.Relations)
	cloned.Capabilities = append([]formset.CapabilityID(nil), resource.Capabilities...)
	cloned.Taxonomies = append([]string(nil), resource.Taxonomies...)
	return cloned
}

func newCoreResource(id content.Kind, label, collection string, fields []formset.Field) Resource {
	return Resource{
		RecordType: formset.RecordType{
			ID:            formset.RecordTypeID(id),
			Label:         label,
			SchemaVersion: "1",
			OwnerModule:   "codex",
			Scope:         formset.ScopeTenant,
			Fields:        fields,
		},
		Collection: collection,
		Public:     true,
	}
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
