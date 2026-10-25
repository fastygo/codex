package canonical

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Format serializes v as RFC 8785 JSON Canonicalization Scheme.
// Numbers follow ECMAScript Number.prototype.toString; -0 becomes 0.
func Format(v any) string {
	return jcs(v)
}

// CanonicalJSON parses any JSON text with the strict parser and returns RFC 8785 JCS.
func CanonicalJSON(inputText string) (string, error) {
	v, err := parse(inputText)
	if err != nil {
		return "", err
	}
	return jcs(v), nil
}

func jcs(v any) string {
	var b strings.Builder
	writeJCS(&b, v)
	return b.String()
}

func writeJCS(b *strings.Builder, v any) {
	switch t := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if t {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case float64:
		b.WriteString(esNumber(t))
	case json.Number:
		f, err := strconv.ParseFloat(string(t), 64)
		if err != nil {
			b.WriteString("null")
			return
		}
		writeJCS(b, f)
	case int:
		writeJCS(b, float64(t))
	case int64:
		writeJCS(b, float64(t))
	case int32:
		writeJCS(b, float64(t))
	case uint64:
		writeJCS(b, float64(t))
	case string:
		writeJCSString(b, t)
	case []any:
		b.WriteByte('[')
		for i, item := range t {
			if i > 0 {
				b.WriteByte(',')
			}
			writeJCS(b, item)
		}
		b.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return utf16Less(keys[i], keys[j]) })
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			writeJCSString(b, k)
			b.WriteByte(':')
			writeJCS(b, t[k])
		}
		b.WriteByte('}')
	default:
		// Should not arise from parse(); keep deterministic rather than panic.
		b.WriteString("null")
	}
}

func writeJCSString(b *strings.Builder, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case 0x08:
			b.WriteString(`\b`)
		case 0x09:
			b.WriteString(`\t`)
		case 0x0a:
			b.WriteString(`\n`)
		case 0x0c:
			b.WriteString(`\f`)
		case 0x0d:
			b.WriteString(`\r`)
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		default:
			if r < 0x20 {
				b.WriteString(`\u00`)
				const hexDigits = "0123456789abcdef"
				b.WriteByte(hexDigits[r>>4])
				b.WriteByte(hexDigits[r&0xf])
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
}

func utf16Less(a, b string) bool {
	ua := utf16.Encode([]rune(a))
	ub := utf16.Encode([]rune(b))
	n := len(ua)
	if len(ub) < n {
		n = len(ub)
	}
	for i := 0; i < n; i++ {
		if ua[i] != ub[i] {
			return ua[i] < ub[i]
		}
	}
	return len(ua) < len(ub)
}

func codePointLess(a, b string) bool {
	ra := []rune(a)
	rb := []rune(b)
	n := len(ra)
	if len(rb) < n {
		n = len(rb)
	}
	for i := 0; i < n; i++ {
		if ra[i] != rb[i] {
			return ra[i] < rb[i]
		}
	}
	return len(ra) < len(rb)
}

// esNumber is ECMAScript Number.prototype.toString for a finite float64.
// 1e21 → "1e+21", 1e-7 → "1e-7", -0 → "0".
func esNumber(f float64) string {
	if f == 0 {
		return "0"
	}
	abs := math.Abs(f)
	if abs >= 1e21 || abs < 1e-6 {
		s := strconv.FormatFloat(f, 'e', -1, 64)
		return stripExpZeros(s)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func stripExpZeros(s string) string {
	i := strings.LastIndexByte(s, 'e')
	if i < 0 {
		return s
	}
	exp := s[i+1:]
	sign := "+"
	if strings.HasPrefix(exp, "+") {
		exp = exp[1:]
	} else if strings.HasPrefix(exp, "-") {
		sign = "-"
		exp = exp[1:]
	}
	exp = strings.TrimLeft(exp, "0")
	if exp == "" {
		exp = "0"
	}
	return s[:i] + "e" + sign + exp
}
