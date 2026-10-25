package validation

// Stable v0.4 issue codes. Codes are a compatibility contract; messages are not.
const (
	DecodeInvalid       = "decode.invalid"
	DecodeUnknownField  = "decode.unknown_field"
	DecodeDuplicateKey  = "decode.duplicate_key"
	DecodeMultipleValue = "decode.multiple_value"

	CoreIDRequired          = "core.id_required"
	CoreIDInvalid           = "core.id_invalid"
	CoreLocaleInvalid       = "core.locale_invalid"
	CoreVersionRequired     = "core.version_required"
	CoreTimestampsRequired  = "core.timestamps_required"
	CoreTimestampsOrder     = "core.timestamps_order"
	CoreOriginInvalid       = "core.origin_invalid"
	CoreMetadataInvalid     = "core.metadata_invalid"
	CoreMetadataKeyRequired = "core.metadata_key_required"
	CoreMetadataKeyShadowed = "core.metadata_key_shadowed"

	CoreEntryTypeInvalid         = "core.entry.type_invalid"
	CoreEntryKindInvalid         = "core.entry.kind_invalid"
	CoreEntryStatusInvalid       = "core.entry.status_invalid"
	CoreEntryVisibilityInvalid   = "core.entry.visibility_invalid"
	CoreEntrySlugRequired        = "core.entry.slug_required"
	CoreEntryTitleRequired       = "core.entry.title_required"
	CoreEntryTimeInvalid         = "core.entry.time_invalid"
	CoreEntryPublishedAtRequired = "core.entry.published_at_required"
	CoreEntryDeletedAtRequired   = "core.entry.deleted_at_required"
	CoreEntryTermsInvalid        = "core.entry.terms_invalid"
	CoreEntryKeyDuplicated       = "core.entry.key_duplicated"
	CoreEntrySlugDuplicated      = "core.entry.slug_duplicated"
	CoreEntryParentMissing       = "core.entry.parent_missing"
	CoreEntryParentSelf          = "core.entry.parent_self"
	CoreEntryParentKindInvalid   = "core.entry.parent_kind_invalid"
	CoreEntryParentCycle         = "core.entry.parent_cycle"

	CoreTermRefIncomplete = "core.term_ref.incomplete"
	CoreTermRefDuplicated = "core.term_ref.duplicated"
	CoreTermRefMissing    = "core.term_ref.missing"

	CoreTermTypeInvalid     = "core.term.type_invalid"
	CoreTermTaxonomyInvalid = "core.term.taxonomy_invalid"
	CoreTermNameRequired    = "core.term.name_required"
	CoreTermSlugRequired    = "core.term.slug_required"
	CoreTermKeyDuplicated   = "core.term.key_duplicated"
	CoreTermSlugDuplicated  = "core.term.slug_duplicated"
	CoreTermParentMissing   = "core.term.parent_missing"
	CoreTermParentSelf      = "core.term.parent_self"
	CoreTermParentCycle     = "core.term.parent_cycle"

	CoreCommentEntryRequired   = "core.comment.entry_required"
	CoreCommentContentRequired = "core.comment.content_required"
	CoreCommentStatusInvalid   = "core.comment.status_invalid"
	CoreCommentKeyDuplicated   = "core.comment.key_duplicated"
	CoreCommentEntryMissing    = "core.comment.entry_missing"

	CoreMenuLocationRequired = "core.menu.location_required"
	CoreMenuItemsRequired    = "core.menu.items_required"
	CoreMenuItemIncomplete   = "core.menu.item_incomplete"
	CoreMenuItemDuplicated   = "core.menu.item_duplicated"
	CoreMenuParentMissing    = "core.menu.parent_missing"
	CoreMenuParentCycle      = "core.menu.parent_cycle"
	CoreMenuTargetInvalid    = "core.menu.target_invalid"
	CoreMenuKeyDuplicated    = "core.menu.key_duplicated"

	CoreDictionaryMessagesRequired = "core.dictionary.messages_required"
	CoreDictionaryValueInvalid     = "core.dictionary.value_invalid"

	CoreSettingsTypeInvalid     = "core.settings.type_invalid"
	CoreSettingsTitleRequired   = "core.settings.title_required"
	CoreSettingsLanguageInvalid = "core.settings.language_invalid"
	CoreSettingsTimezoneInvalid = "core.settings.timezone_invalid"
	CoreSettingsCurrencyInvalid = "core.settings.currency_invalid"
	CoreSettingsKeyDuplicated   = "core.settings.key_duplicated"

	ManifestInvalid         = "manifest.invalid"
	ManifestKindUnknown     = "manifest.kind_unknown"
	ManifestMetadataUnknown = "manifest.metadata_unknown"
	ManifestFieldRequired   = "manifest.field_required"
	ManifestFieldInvalid    = "manifest.field_invalid"
	ManifestCurrencyMissing = "manifest.currency_missing"
	ManifestCurrencyInvalid = "manifest.currency_invalid"

	// Q7: a non-localized field differs across locale records that share an id.
	EntryLocalizedMismatch = "entry.localized_mismatch"

	// Canonical profile errors (codex-entry/v1, codex-term/v1, codex-manifest/v2).
	CanonicalUnsafeInteger     = "canonical.unsafe_integer"
	CanonicalFractionalSeconds = "canonical.fractional_seconds"
	CanonicalTimeInvalid       = "canonical.time_invalid"
	CanonicalNonFiniteNumber   = "canonical.non_finite_number"

	SchemaFieldRuleUnknown = "schema.field.rule_unknown"
)

// WarningCodes are reported with severity "warning" and do not make a record invalid.
var WarningCodes = map[string]struct{}{
	CoreMetadataKeyShadowed: {},
	ManifestKindUnknown:     {},
	ManifestMetadataUnknown: {},
}
