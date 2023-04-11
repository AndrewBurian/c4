package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/textproto"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
)

type archiveLoader struct {
	archive *txtar.Archive
}

func (al *archiveLoader) Load(_ context.Context, target string) ([]byte, error) {
	for i := range al.archive.Files {
		if al.archive.Files[i].Name == target {
			return al.archive.Files[i].Data, nil
		}
	}

	return nil, fmt.Errorf("mock loader: no file for %s", target)
}

type testCase struct {
	target      string
	outFile     string
	expectErr   string
	matchFile   string
	compareWith string
	jsonPretty  string

	archive *txtar.Archive
}

func SetupTest(t *testing.T, directives textproto.MIMEHeader) (*testCase, error) {
	tt := &testCase{
		target: directives.Get("Target"),

		outFile:     directives.Get("Output-File"),
		matchFile:   directives.Get("Output-Match"),
		compareWith: directives.Get("Compare-With"),

		expectErr:  directives.Get("Should-Error"),
		jsonPretty: directives.Get("Json-Pretty"),
	}

	if tt.outFile == "" {
		tt.outFile = "_out.c4c"
	}

	tt.outFile = filepath.Join(t.TempDir(), tt.outFile)

	if tt.compareWith == "" {
		tt.compareWith = "json"
	}

	return tt, nil
}

func (tc *testCase) Check(t *testing.T, err error) {
	if err != nil {
		if tc.expectErr != "" {
			t.Logf("expected error occurred\n%s", err)
			return
		}
		t.Errorf("compile error occured: %s", err)
		return
	}

	if tc.matchFile != "" {
		var matchBytes []byte
		for _, file := range tc.archive.Files {
			if file.Name == tc.matchFile {
				matchBytes = file.Data
				break
			}
		}
		if matchBytes == nil {
			t.Errorf("Specified output match %s not present in archive", tc.matchFile)
		}

		var gotBytes []byte
		gotBytes, err = os.ReadFile(tc.outFile)
		if err != nil {
			t.Fatalf("unable to read compiled output: %s", err)
		}

		switch tc.compareWith {
		case "json":
			var want, got map[string]any
			err = json.Unmarshal(matchBytes, &want)
			if err != nil {
				t.Fatalf("error interpreting expected output as JSON: %s", err)
			}
			err = json.Unmarshal(gotBytes, &got)
			if err != nil {
				t.Fatalf("error interpreting compile output as JSON: %s", err)
			}

			compareObjects(t, nil, want, got)
			// if !reflect.DeepEqual(want, got) {
			// 	t.Error("compiled output JSON does not match expected")
			// }

		default:
			t.Errorf("unknown comparison method: %s", tc.compareWith)
		}
	}

}

func Test_CompileScripts(t *testing.T) {
	runScriptDir(t, "testdata/scripts")
}

func runScriptDir(t *testing.T, dirName string) {
	testFiles, err := os.ReadDir(dirName)
	if err != nil {
		t.Fatalf("unable to read test scripts directory: %s", err)
	}

	for _, entry := range testFiles {
		fullPath := filepath.Join(dirName, entry.Name())
		if entry.Type().IsDir() {
			t.Run(filepath.Base(fullPath), func(t *testing.T) {
				runScriptDir(t, fullPath)
			})
			continue
		}

		if entry.Type().IsRegular() {
			t.Run(filepath.Base(fullPath), func(t *testing.T) {
				runScript(t, fullPath)
			})
			continue
		}

		t.Logf("Skipping irregular file %s", fullPath)
	}
}

func runScript(t *testing.T, file string) {

	archive, err := txtar.ParseFile(file)
	if err != nil {
		t.Fatalf("error reading test script archive: %s", err)
	}
	read := textproto.NewReader(bufio.NewReader(bytes.NewReader(archive.Comment)))
	header, err := read.ReadMIMEHeader()
	if err != nil {
		t.Fatalf("error reading script directives: %s", err)
	}

	tt, err := SetupTest(t, header)
	if err != nil {
		t.Fatalf("invalid test configuration: %s", err)
	}
	tt.archive = archive

	c := new(compiler)
	c.loader = &archiveLoader{archive}
	c.context = context.Background()
	c.outputFile = tt.outFile

	if tt.jsonPretty != "" {
		c.jsonPretty = true
	}

	err = c.Run(tt.target)

	tt.Check(t, err)
}

func compareObjects(t *testing.T, path []string, a, b any) {
	if reflect.TypeOf(a) != reflect.TypeOf(b) {
		t.Errorf("types mismatch at %s", strings.Join(path, "."))
		t.Logf("a = %s", reflect.TypeOf(a))
		t.Logf("b = %s", reflect.TypeOf(b))
		return
	}

	if aMap, ok := a.(map[string]any); ok {
		bMap := b.(map[string]any)

		numKeys := 0
		for key, aMapVal := range aMap {
			numKeys++

			bMapVal, has := bMap[key]
			if !has {
				t.Errorf("b at %s does not contain key %s", strings.Join(path, "."), key)
				continue
			}

			compareObjects(t, append(path, key), aMapVal, bMapVal)
		}
	}

	if aStr, ok := a.(string); ok {
		bStr := b.(string)

		if aStr[0] == '_' || bStr[0] == '_' {
			t.Logf("not checking match of anonymous value at %s", strings.Join(path, "."))
			return
		}

		if aStr != bStr {
			t.Errorf("at %s string values don't match", strings.Join(path, "."))
			t.Logf("a = %s", aStr)
			t.Logf("b = %s", bStr)
		}
	}
}
