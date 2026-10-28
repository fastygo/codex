package canonical

func settingsSpec() recordSpec {
	return recordSpec{
		kind:     "settings",
		keys:     settingsKeys,
		required: requiredSettings,
		dates:    stampDates,
		special: map[string]fieldHandler{
			"metadata": handleMetadata,
			"origin":   handleOrigin,
		},
		nested: map[string]nestedShape{
			"contact": {
				keys: contactKeys,
				inner: map[string]nestedShape{
					"links": {keys: contactLinkKeys, array: true},
				},
			},
			"hours":            {keys: hoursRuleKeys, array: true},
			"hours_exceptions": {keys: hoursExceptionKeys, array: true},
			"map":              {keys: mapKeys},
		},
	}
}

// CanonicalizeSettings returns the codex-settings/v1 JCS text of one SettingsV04 JSON document.
func CanonicalizeSettings(inputText string) (string, error) {
	return canonicalizeRecord(inputText, settingsSpec())
}

// SettingsDigest returns "codex-settings/v1:sha256:" plus hex(sha256(UTF-8 bytes of CanonicalizeSettings(inputText))).
func SettingsDigest(inputText string) (string, error) {
	text, err := CanonicalizeSettings(inputText)
	if err != nil {
		return "", err
	}
	return digestPrefixed(SettingsDigestPrefix, text), nil
}
