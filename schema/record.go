package schema

import (
	"fmt"
	"strings"

	"github.com/fastygo/codex/validation"
)

type RecordTypeID string
type FieldID string
type RelationID string
type CapabilityID string
type SchemaVersion string
type Scope string
type FieldType string
type RelationCardinality string
type DeleteBehavior string
type CrossWorkspaceMode string

const (
	ScopeGlobal    Scope = "global"
	ScopeWorkspace Scope = "workspace"
	ScopeTenant    Scope = "tenant"
	ScopeUser      Scope = "user"
)

const (
	FieldText       FieldType = "text"
	FieldString     FieldType = "string"
	FieldTextarea   FieldType = "textarea"
	FieldRichText   FieldType = "richtext"
	FieldMarkdown   FieldType = "markdown"
	FieldNumber     FieldType = "number"
	FieldBoolean    FieldType = "boolean"
	FieldSelect     FieldType = "select"
	FieldDateTime   FieldType = "datetime"
	FieldJSON       FieldType = "json"
	FieldObject     FieldType = "object"
	FieldRelation   FieldType = "relation"
	FieldCollection FieldType = "collection"
	FieldComputed   FieldType = "computed"
	FieldEncrypted  FieldType = "encrypted"
)

const (
	RelationOneToOne   RelationCardinality = "one-to-one"
	RelationOneToMany  RelationCardinality = "one-to-many"
	RelationManyToMany RelationCardinality = "many-to-many"
)

const (
	DeleteRestrict DeleteBehavior = "restrict"
	DeleteCascade  DeleteBehavior = "cascade"
	DeleteNullify  DeleteBehavior = "nullify"
)

type RelationPolicy struct {
	CrossWorkspaceMode CrossWorkspaceMode `json:"cross_workspace_mode,omitempty"`
	AllowedTargets     []string           `json:"allowed_targets,omitempty"`
	Capability         CapabilityID       `json:"capability,omitempty"`
	ReadOnly           bool               `json:"read_only,omitempty"`
}

type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type ValidationRule struct {
	Name     string            `json:"name"`
	Message  string            `json:"message,omitempty"`
	Args     map[string]string `json:"args,omitempty"`
	Severity string            `json:"severity,omitempty"`
}

// Field is a content field. Presentation widgets belong to a form projection.
type Field struct {
	ID           FieldID          `json:"id"`
	Label        string           `json:"label"`
	Type         FieldType        `json:"type"`
	Namespace    string           `json:"namespace,omitempty"`
	OwnerModule  string           `json:"owner_module,omitempty"`
	Description  string           `json:"description,omitempty"`
	Required     bool             `json:"required,omitempty"`
	Localized    bool             `json:"localized,omitempty"`
	DefaultValue string           `json:"default_value,omitempty"`
	Options      []Option         `json:"options,omitempty"`
	Rules        []ValidationRule `json:"rules,omitempty"`
	Items        *Field           `json:"items,omitempty"`
	Fields       []Field          `json:"fields,omitempty"`
	Searchable   bool             `json:"searchable,omitempty"`
	Indexed      bool             `json:"indexed,omitempty"`
	Unique       bool             `json:"unique,omitempty"`
	Sensitive    bool             `json:"sensitive,omitempty"`
	Encrypted    bool             `json:"encrypted,omitempty"`
	UIHint       string           `json:"ui_hint,omitempty"`
	StorageHint  string           `json:"storage_hint,omitempty"`
}

type Relation struct {
	ID                   RelationID          `json:"id"`
	Label                string              `json:"label,omitempty"`
	Source               RecordTypeID        `json:"source"`
	Target               RecordTypeID        `json:"target"`
	Cardinality          RelationCardinality `json:"cardinality"`
	InverseName          string              `json:"inverse_name,omitempty"`
	CrossWorkspacePolicy string              `json:"cross_workspace_policy,omitempty"`
	Policy               RelationPolicy      `json:"policy,omitempty"`
	DeleteBehavior       DeleteBehavior      `json:"delete_behavior,omitempty"`
}

// RecordType declares one manifest resource kind.
type RecordType struct {
	ID            RecordTypeID   `json:"id"`
	Label         string         `json:"label"`
	Description   string         `json:"description,omitempty"`
	SchemaVersion SchemaVersion  `json:"schema_version,omitempty"`
	OwnerModule   string         `json:"owner_module,omitempty"`
	Scope         Scope          `json:"scope"`
	Fields        []Field        `json:"fields,omitempty"`
	Relations     []Relation     `json:"relations,omitempty"`
	Capabilities  []CapabilityID `json:"capabilities,omitempty"`
	Visibility    string         `json:"visibility,omitempty"`
}

func (field Field) Validate() error {
	if strings.TrimSpace(string(field.ID)) == "" {
		return fmt.Errorf("field id is required")
	}
	if strings.TrimSpace(field.Label) == "" {
		return fmt.Errorf("field %q label is required", field.ID)
	}
	if strings.TrimSpace(string(field.Type)) == "" {
		return fmt.Errorf("field %q type is required", field.ID)
	}
	if field.Type == FieldCollection && field.Items == nil {
		return fmt.Errorf("field %q collection requires items", field.ID)
	}
	if field.Type == FieldObject && len(field.Fields) == 0 {
		return fmt.Errorf("field %q object requires fields", field.ID)
	}
	if field.Type != FieldCollection && field.Items != nil {
		return fmt.Errorf("field %q items require collection type", field.ID)
	}
	if field.Type != FieldObject && len(field.Fields) > 0 {
		return fmt.Errorf("field %q nested fields require object type", field.ID)
	}
	if field.Items != nil {
		if err := field.Items.Validate(); err != nil {
			return err
		}
	}
	seen := map[FieldID]struct{}{}
	for _, nested := range field.Fields {
		if _, exists := seen[nested.ID]; exists {
			return fmt.Errorf("field %q nested field %q is duplicated", field.ID, nested.ID)
		}
		seen[nested.ID] = struct{}{}
		if err := nested.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (relation Relation) Validate() error {
	if strings.TrimSpace(string(relation.ID)) == "" {
		return fmt.Errorf("relation id is required")
	}
	if relation.Source == "" || relation.Target == "" {
		return fmt.Errorf("relation %q source and target are required", relation.ID)
	}
	if strings.TrimSpace(string(relation.Cardinality)) == "" {
		return fmt.Errorf("relation %q cardinality is required", relation.ID)
	}
	return nil
}

func (record RecordType) Validate() error {
	if strings.TrimSpace(string(record.ID)) == "" {
		return fmt.Errorf("record type id is required")
	}
	if strings.TrimSpace(record.Label) == "" {
		return fmt.Errorf("record type %q label is required", record.ID)
	}
	if strings.TrimSpace(string(record.Scope)) == "" {
		return fmt.Errorf("record type %q scope is required", record.ID)
	}
	seen := map[FieldID]struct{}{}
	for _, field := range record.Fields {
		if _, exists := seen[field.ID]; exists {
			return fmt.Errorf("field %q is duplicated", field.ID)
		}
		seen[field.ID] = struct{}{}
		if err := field.Validate(); err != nil {
			return err
		}
	}
	for _, relation := range record.Relations {
		if err := relation.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func reviewRelations(resources []Resource) error {
	ids := make(map[RecordTypeID]struct{}, len(resources))
	for _, resource := range resources {
		ids[resource.Record.ID] = struct{}{}
	}
	for _, resource := range resources {
		for _, relation := range resource.Record.Relations {
			if _, ok := ids[relation.Source]; !ok {
				return validation.New("schema.relation.source_missing", string(relation.ID), "relation source record is missing")
			}
			if _, ok := ids[relation.Target]; !ok {
				return validation.New("schema.relation.target_missing", string(relation.ID), "relation target record is missing")
			}
		}
	}
	return nil
}
