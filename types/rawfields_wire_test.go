package types_test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/tj-smith47/shelly-go/gen2/components"
)

// An embedded RawFields without a `json:"-"` tag is encoded by encoding/json as
// a field literally named "RawFields", so every SetConfig call would carry
// "RawFields":null to the device. The tag keeps it off the wire; a struct that
// wants to capture unknown fields does so in its own UnmarshalJSON.
var untaggedRawFields = regexp.MustCompile(`^\s+(types\.)?RawFields\s*$`)

func TestRawFields_EveryEmbedIsTaggedOffTheWire(t *testing.T) {
	root := moduleRoot(t)
	var offenders []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == ".git" || name == "vendor" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		for line := 1; sc.Scan(); line++ {
			if untaggedRawFields.MatchString(sc.Text()) {
				rel, _ := filepath.Rel(root, path)
				offenders = append(offenders, rel+":"+strconv.Itoa(line))
			}
		}
		return sc.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(offenders) > 0 {
		t.Errorf("embedded RawFields without `json:\"-\"` (it would be sent to the device as \"RawFields\":null):\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

func TestRawFields_StayOffTheWire(t *testing.T) {
	name := "guest bath"
	enable := false
	for _, v := range []any{
		&components.SysConfig{Device: &components.SysDeviceConfig{Name: &name}},
		&components.CloudConfig{Enable: &enable},
	} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "RawFields") {
			t.Errorf("%T marshals %s; RawFields must not reach the device", v, b)
		}
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test directory")
		}
		dir = parent
	}
}
