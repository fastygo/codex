package canonical

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

const maxSafeInteger = int64(1<<53 - 1)

var (
	bigMaxSafe = big.NewInt(maxSafeInteger)
	bigMinSafe = big.NewInt(-maxSafeInteger)
)

type parser struct {
	text string
	i    int
}

func parse(text string) (any, error) {
	p := parser{text: text}
	v, err := p.value(0)
	if err != nil {
		return nil, err
	}
	p.ws()
	if p.i != len(p.text) {
		return nil, p.fail(ErrInvalidJSON, "trailing data after the JSON value", p.i)
	}
	return v, nil
}

func (p *parser) fail(code, msg string, at int) error {
	if at < 0 {
		at = p.i
	}
	return fail(code, "@"+strconv.Itoa(at), msg)
}

func (p *parser) ws() {
	for p.i < len(p.text) {
		c := p.text[p.i]
		if c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d {
			p.i++
			continue
		}
		return
	}
}

func (p *parser) lit(word string, v any) (any, error) {
	if strings.HasPrefix(p.text[p.i:], word) {
		p.i += len(word)
		return v, nil
	}
	return nil, p.fail(ErrInvalidJSON, "unexpected token", p.i)
}

func (p *parser) str() (string, error) {
	start := p.i
	p.i++ // opening quote already verified by caller
	var units []uint16
	for {
		if p.i >= len(p.text) {
			return "", p.fail(ErrInvalidJSON, "unterminated string", start)
		}
		c := p.text[p.i]
		if c == 0x22 {
			p.i++
			break
		}
		if c < 0x20 {
			return "", p.fail(ErrInvalidJSON, "raw control character in string", p.i)
		}
		if c != 0x5c {
			r, size := utf8.DecodeRuneInString(p.text[p.i:])
			if r == utf8.RuneError && size == 1 {
				return "", p.fail(ErrInvalidJSON, "invalid UTF-8 in string", p.i)
			}
			units = utf16.AppendRune(units, r)
			p.i += size
			continue
		}
		if p.i+1 >= len(p.text) {
			return "", p.fail(ErrInvalidJSON, "unterminated string", start)
		}
		e := p.text[p.i+1]
		escAt := p.i
		p.i += 2
		switch e {
		case '"':
			units = append(units, '"')
		case '\\':
			units = append(units, '\\')
		case '/':
			units = append(units, '/')
		case 'b':
			units = append(units, '\b')
		case 'f':
			units = append(units, '\f')
		case 'n':
			units = append(units, '\n')
		case 'r':
			units = append(units, '\r')
		case 't':
			units = append(units, '\t')
		case 'u':
			if p.i+4 > len(p.text) {
				return "", p.fail(ErrInvalidJSON, "bad \\u escape", escAt)
			}
			h := p.text[p.i : p.i+4]
			n, err := strconv.ParseUint(h, 16, 16)
			if err != nil {
				return "", p.fail(ErrInvalidJSON, "bad \\u escape", escAt)
			}
			units = append(units, uint16(n))
			p.i += 4
		default:
			return "", p.fail(ErrInvalidJSON, "bad escape", escAt)
		}
	}
	for k := 0; k < len(units); k++ {
		c := units[k]
		if c >= 0xd800 && c <= 0xdbff {
			if k+1 < len(units) {
				d := units[k+1]
				if d >= 0xdc00 && d <= 0xdfff {
					k++
					continue
				}
			}
			return "", p.fail(ErrInvalidString, "lone surrogate in string", start)
		}
		if c >= 0xd800 && c <= 0xdfff {
			return "", p.fail(ErrInvalidString, "lone surrogate in string", start)
		}
	}
	return string(utf16.Decode(units)), nil
}

func (p *parser) num() (float64, error) {
	at := p.i
	tok, frac, exp := p.scanNumber()
	if tok == "" {
		return 0, p.fail(ErrInvalidJSON, "bad number", at)
	}
	if !frac && !exp {
		n := new(big.Int)
		if _, ok := n.SetString(tok, 10); !ok {
			return 0, p.fail(ErrInvalidJSON, "bad number", at)
		}
		if n.Cmp(bigMaxSafe) > 0 || n.Cmp(bigMinSafe) < 0 {
			return 0, p.fail(ErrUnsafeInteger, fmt.Sprintf("integer %s is outside ±(2^53-1)", tok), at)
		}
	}
	f, err := strconv.ParseFloat(tok, 64)
	if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
		return 0, p.fail(ErrInvalidNumber, fmt.Sprintf("number %s is not finite", tok), at)
	}
	if f == math.Trunc(f) && math.Abs(f) > float64(maxSafeInteger) && !strings.ContainsAny(tok, "eE") {
		return 0, p.fail(ErrUnsafeInteger, fmt.Sprintf("integer %s is outside ±(2^53-1)", tok), at)
	}
	return f, nil
}

func (p *parser) scanNumber() (tok string, frac, exp bool) {
	start := p.i
	if p.i < len(p.text) && p.text[p.i] == '-' {
		p.i++
	}
	if p.i >= len(p.text) {
		p.i = start
		return "", false, false
	}
	if p.text[p.i] == '0' {
		p.i++
	} else if p.text[p.i] >= '1' && p.text[p.i] <= '9' {
		p.i++
		for p.i < len(p.text) && p.text[p.i] >= '0' && p.text[p.i] <= '9' {
			p.i++
		}
	} else {
		p.i = start
		return "", false, false
	}
	if p.i < len(p.text) && p.text[p.i] == '.' {
		frac = true
		p.i++
		if p.i >= len(p.text) || p.text[p.i] < '0' || p.text[p.i] > '9' {
			p.i = start
			return "", false, false
		}
		for p.i < len(p.text) && p.text[p.i] >= '0' && p.text[p.i] <= '9' {
			p.i++
		}
	}
	if p.i < len(p.text) && (p.text[p.i] == 'e' || p.text[p.i] == 'E') {
		exp = true
		p.i++
		if p.i < len(p.text) && (p.text[p.i] == '+' || p.text[p.i] == '-') {
			p.i++
		}
		if p.i >= len(p.text) || p.text[p.i] < '0' || p.text[p.i] > '9' {
			p.i = start
			return "", false, false
		}
		for p.i < len(p.text) && p.text[p.i] >= '0' && p.text[p.i] <= '9' {
			p.i++
		}
	}
	return p.text[start:p.i], frac, exp
}

func (p *parser) value(depth int) (any, error) {
	if depth > 512 {
		return nil, p.fail(ErrInvalidJSON, "nesting too deep", p.i)
	}
	p.ws()
	if p.i >= len(p.text) {
		return nil, p.fail(ErrInvalidJSON, "unexpected end of input", p.i)
	}
	c := p.text[p.i]
	if c == '{' {
		p.i++
		obj := map[string]any{}
		p.ws()
		if p.i < len(p.text) && p.text[p.i] == '}' {
			p.i++
			return obj, nil
		}
		for {
			p.ws()
			if p.i >= len(p.text) || p.text[p.i] != '"' {
				return nil, p.fail(ErrInvalidJSON, "expected a key", p.i)
			}
			kAt := p.i
			k, err := p.str()
			if err != nil {
				return nil, err
			}
			if _, exists := obj[k]; exists {
				return nil, p.fail(ErrDuplicateKey, fmt.Sprintf("duplicate key %s", strconv.Quote(k)), kAt)
			}
			p.ws()
			if p.i >= len(p.text) || p.text[p.i] != ':' {
				return nil, p.fail(ErrInvalidJSON, "expected ':'", p.i)
			}
			p.i++
			v, err := p.value(depth + 1)
			if err != nil {
				return nil, err
			}
			obj[k] = v
			p.ws()
			if p.i < len(p.text) && p.text[p.i] == ',' {
				p.i++
				continue
			}
			if p.i < len(p.text) && p.text[p.i] == '}' {
				p.i++
				return obj, nil
			}
			return nil, p.fail(ErrInvalidJSON, "expected ',' or '}'", p.i)
		}
	}
	if c == '[' {
		p.i++
		arr := []any{}
		p.ws()
		if p.i < len(p.text) && p.text[p.i] == ']' {
			p.i++
			return arr, nil
		}
		for {
			v, err := p.value(depth + 1)
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
			p.ws()
			if p.i < len(p.text) && p.text[p.i] == ',' {
				p.i++
				continue
			}
			if p.i < len(p.text) && p.text[p.i] == ']' {
				p.i++
				return arr, nil
			}
			return nil, p.fail(ErrInvalidJSON, "expected ',' or ']'", p.i)
		}
	}
	if c == '"' {
		return p.str()
	}
	if c == 't' {
		return p.lit("true", true)
	}
	if c == 'f' {
		return p.lit("false", false)
	}
	if c == 'n' {
		return p.lit("null", nil)
	}
	if c == '-' || (c >= '0' && c <= '9') {
		return p.num()
	}
	return nil, p.fail(ErrInvalidJSON, "unexpected token", p.i)
}
