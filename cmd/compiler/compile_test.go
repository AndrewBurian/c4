package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type mockLoader struct {
	sources     map[string]string
	fileSources map[string][]byte
}

func (ml *mockLoader) Load(_ context.Context, n string) ([]byte, error) {
	if s, has := ml.sources[n]; has {
		return []byte(s), nil
	}

	if s, has := ml.fileSources[n]; has {
		return s, nil
	}

	fileData, err := os.ReadFile(filepath.Join("testdata", n))
	if err != nil {
		return nil, err
	}
	ml.fileSources[n] = fileData

	return fileData, nil
}

type compileOutput struct {
	toMatchFile  string
	toBe         []byte
	toBeAnything bool
	toError      bool
}

var expectFileCache map[string][]byte

func expectedStr(s string) []byte {
	if !strings.HasSuffix(s, "\n") {
		return []byte(s + "\n")
	}
	return []byte(s)
}

func (expect *compileOutput) check(t *testing.T, actual []byte, err error) {
	if err != nil {
		if expect.toError {
			t.Logf("expected error occurred\n%s", err)
			return
		}
		t.Errorf("compile error occured: %s", err)
		return
	}
	if expect.toMatchFile != "" && expect.toBe == nil {
		if err := expect.fromFile(); err != nil {
			t.Fatalf("error reading expected test output from file: %s", err)
		}
	}

	if expect.toBe != nil {
		if !bytes.Equal(expect.toBe, actual) {
			t.Errorf("expected and actual bytes don't match")
			t.Logf("Data for comparison:\nexp: %q\nact: %q", string(expect.toBe), string(actual))
			return
		}
		t.Log("output matches expected")
		return
	}

	if expect.toBeAnything {
		if len(actual) > 0 {
			t.Log("output was present")
		}
		t.Error("no output present")
	}

	t.Fatal("nothing to compare to")

}

func (expect *compileOutput) fromFile() error {
	if exptBytes, cached := expectFileCache[expect.toMatchFile]; cached {
		expect.toBe = exptBytes
		return nil
	}

	fileBytes, err := os.ReadFile(filepath.Join("testdata", "outputs", expect.toMatchFile))
	if err != nil {
		return err
	}

	expectFileCache[expect.toMatchFile] = fileBytes
	expect.toBe = fileBytes
	return nil
}

func Test_compiler_Run(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		config    compileConfig
		mockFs    map[string]string
		expectErr bool
		expect    *compileOutput
	}{
		{
			name: "basic compile",
			config: compileConfig{
				jsonPretty: false,
			},
			mockFs: map[string]string{
				"main.c4": `
					workspace 'foo' { }
				`,
			},
			expect: &compileOutput{toBe: expectedStr(`{"name":"foo"}`)},
		},
		{
			name: "basic error",
			mockFs: map[string]string{
				"main.c4": `
					workspace 'foo' {
						model {
							identifier = 'unexpected string'
						}
					}
				`,
			},
			expect: &compileOutput{toError: true},
		},
		{
			name: "basic include pragma",
			mockFs: map[string]string{
				"main.c4": `
					workspace 'foo' {
						model {
							a = person 'bob' {
								#include props.c4
							}
						}
					}
				`,
				"props.c4": `properties { "proptype" "common"; }`,
			},
			expect: &compileOutput{toMatchFile: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := &compiler{
				loader:        &mockLoader{sources: tt.mockFs},
				context:       context.Background(),
				compileConfig: tt.config,
			}

			if comp.outputFile == "" {
				comp.outputFile = fmt.Sprintf("%s/testout.c4c", t.TempDir())
			}
			t.Logf("writting to file: %s", comp.outputFile)
			if tt.target == "" {
				tt.target = "main.c4"
			}

			err := comp.Run(tt.target)
			compiledData, readErr := os.ReadFile(comp.outputFile)
			if readErr != nil {
				if !errors.Is(readErr, os.ErrNotExist) {
					t.Errorf("error reading output file: %s", readErr)
				}
			}

			tt.expect.check(t, compiledData, err)
		})
	}
}
