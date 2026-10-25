package canonical_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/codex/canonical"
)

type vectorFile struct {
	Profile string       `json:"profile"`
	Cases   []vectorCase `json:"cases"`
}

type vectorCase struct {
	Name      string `json:"name"`
	InputText string `json:"input_text"`
	Canonical string `json:"canonical"`
	Digest    string `json:"digest"`
	Error     string `json:"error"`
}

func loadVectorFile(t *testing.T, name string) vectorFile {
	t.Helper()
	path := filepath.Join("..", "conformance", "contract", "fixtures", "vectors", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read vectors %s: %v", name, err)
	}
	var file vectorFile
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("decode vectors %s: %v", name, err)
	}
	return file
}

func loadVectors(t *testing.T) vectorFile {
	t.Helper()
	return loadVectorFile(t, "canonical.json")
}

func TestOfficialCanonicalVectors(t *testing.T) {
	file := loadVectors(t)
	if file.Profile != canonical.EntryProfile {
		t.Fatalf("profile = %q, want %q", file.Profile, canonical.EntryProfile)
	}
	if len(file.Cases) != 24 {
		t.Fatalf("case count = %d, want 24", len(file.Cases))
	}
	ok, fail := 0, 0
	for _, c := range file.Cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			if c.Error != "" {
				fail++
				_, err := canonical.Canonicalize(c.InputText)
				if err == nil {
					t.Fatalf("Canonicalize() error = nil, want %s", c.Error)
				}
				if got := canonical.Code(err); got != c.Error {
					t.Fatalf("error code = %q, want %q (err = %v)", got, c.Error, err)
				}
				if _, err := canonical.Digest(c.InputText); err == nil {
					t.Fatal("Digest() error = nil")
				}
				return
			}
			ok++
			out, err := canonical.Canonicalize(c.InputText)
			if err != nil {
				t.Fatalf("Canonicalize() error = %v", err)
			}
			if out != c.Canonical {
				t.Fatalf("canonical mismatch\n got %s\nwant %s", out, c.Canonical)
			}
			digest, err := canonical.Digest(c.InputText)
			if err != nil {
				t.Fatalf("Digest() error = %v", err)
			}
			if digest != c.Digest {
				t.Fatalf("digest = %q, want %q", digest, c.Digest)
			}
			again, err := canonical.Canonicalize(out)
			if err != nil {
				t.Fatalf("idempotent Canonicalize() error = %v", err)
			}
			if again != out {
				t.Fatal("canonical form is not idempotent")
			}
		})
	}
	_ = ok
	_ = fail
}

func TestOfficialManifestVectors(t *testing.T) {
	file := loadVectorFile(t, "manifest.json")
	if file.Profile != canonical.ManifestProfile {
		t.Fatalf("profile = %q, want %q", file.Profile, canonical.ManifestProfile)
	}
	if len(file.Cases) < 6 {
		t.Fatalf("case count = %d, want at least 6", len(file.Cases))
	}
	for _, c := range file.Cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			if c.Error != "" {
				_, err := canonical.CanonicalizeManifest(c.InputText)
				if err == nil {
					t.Fatalf("CanonicalizeManifest() error = nil, want %s", c.Error)
				}
				if got := canonical.Code(err); got != c.Error {
					t.Fatalf("error code = %q, want %q (err = %v)", got, c.Error, err)
				}
				if _, err := canonical.ManifestDigest(c.InputText); err == nil {
					t.Fatal("ManifestDigest() error = nil")
				}
				return
			}
			out, err := canonical.CanonicalizeManifest(c.InputText)
			if err != nil {
				t.Fatalf("CanonicalizeManifest() error = %v", err)
			}
			if out != c.Canonical {
				t.Fatalf("canonical mismatch\n got %s\nwant %s", out, c.Canonical)
			}
			digest, err := canonical.ManifestDigest(c.InputText)
			if err != nil {
				t.Fatalf("ManifestDigest() error = %v", err)
			}
			if digest != c.Digest {
				t.Fatalf("digest = %q, want %q", digest, c.Digest)
			}
			again, err := canonical.CanonicalizeManifest(out)
			if err != nil {
				t.Fatalf("idempotent CanonicalizeManifest() error = %v", err)
			}
			if again != out {
				t.Fatal("canonical form is not idempotent")
			}
		})
	}
}

func TestOfficialTermSettingsVectorsPending(t *testing.T) {
	for _, name := range []string{"term.json", "settings.json"} {
		path := filepath.Join("..", "conformance", "contract", "fixtures", "vectors", name)
		if _, err := os.Stat(path); err != nil {
			t.Logf("TODO: %s is not in the contract pin yet", name)
			continue
		}
		file := loadVectorFile(t, name)
		canonicalize := canonical.CanonicalizeTerm
		digestFn := canonical.TermDigest
		profile := canonical.TermProfile
		if name == "settings.json" {
			canonicalize = canonical.CanonicalizeSettings
			digestFn = canonical.SettingsDigest
			profile = canonical.SettingsProfile
		}
		if file.Profile != profile {
			t.Fatalf("%s profile = %q, want %q", name, file.Profile, profile)
		}
		for _, c := range file.Cases {
			c := c
			t.Run(name+"/"+c.Name, func(t *testing.T) {
				if c.Error != "" {
					_, err := canonicalize(c.InputText)
					if err == nil {
						t.Fatalf("error = nil, want %s", c.Error)
					}
					if got := canonical.Code(err); got != c.Error {
						t.Fatalf("error code = %q, want %q", got, c.Error)
					}
					return
				}
				out, err := canonicalize(c.InputText)
				if err != nil {
					t.Fatalf("canonicalize error = %v", err)
				}
				if out != c.Canonical {
					t.Fatalf("canonical mismatch\n got %s\nwant %s", out, c.Canonical)
				}
				digest, err := digestFn(c.InputText)
				if err != nil {
					t.Fatalf("digest error = %v", err)
				}
				if digest != c.Digest {
					t.Fatalf("digest = %q, want %q", digest, c.Digest)
				}
			})
		}
	}
}

func TestESNumberFormat(t *testing.T) {
	// Matches the numbers vector: 1250 4.5 0.1 12.50 1e21 1e-7 -0 -1250 max-safe 1E2 100e-2
	input := `{"id":"post-1","locale":"en-US","kind":"post","status":"published","visibility":"public","slug":"hello","title":"Hello","version":1,"created_at":"2026-10-25T11:09:00Z","updated_at":"2026-10-25T11:09:00Z","metadata":{"x":{"value":[1250,4.5,0.1,12.50,1e21,1e-7,-0,-0.0,-1250,9007199254740991,-9007199254740991,1E2,100e-2]}}}`
	out, err := canonical.Canonicalize(input)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"created_at":"2026-10-25T11:09:00Z","id":"post-1","kind":"post","locale":"en-US","metadata":{"x":{"value":[1250,4.5,0.1,12.5,1e+21,1e-7,0,0,-1250,9007199254740991,-9007199254740991,100,1]}},"slug":"hello","status":"published","title":"Hello","updated_at":"2026-10-25T11:09:00Z","version":1,"visibility":"public"}`; out != want {
		t.Fatalf("got %s\nwant %s", out, want)
	}
}
