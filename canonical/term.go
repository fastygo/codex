package canonical

func termSpec() recordSpec {
	return recordSpec{
		kind:     "term",
		keys:     termKeys,
		required: requiredTerm,
		dates:    stampDates,
		// count is a computed WordPress field and is not part of codex-term/v1.
		drop: []string{"count"},
		special: map[string]fieldHandler{
			"metadata": handleMetadata,
			"origin":   handleOrigin,
		},
	}
}

// CanonicalizeTerm returns the codex-term/v1 JCS text of one Term v0.4 JSON document.
// count is dropped and is not part of the canonical form.
func CanonicalizeTerm(inputText string) (string, error) {
	return canonicalizeRecord(inputText, termSpec())
}

// TermDigest returns "codex-term/v1:sha256:" plus hex(sha256(UTF-8 bytes of CanonicalizeTerm(inputText))).
func TermDigest(inputText string) (string, error) {
	text, err := CanonicalizeTerm(inputText)
	if err != nil {
		return "", err
	}
	return digestPrefixed(TermDigestPrefix, text), nil
}
