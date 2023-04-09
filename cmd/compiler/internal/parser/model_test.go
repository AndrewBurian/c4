package parser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"go.burian.dev/c4/cmd/compiler/internal/lexer"
)

func jsonMust(o any) []byte {
	b, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		panic(err)
	}
	return b
}

func visualCompare(t *testing.T, want, got any, contextLines int) {
	t.Helper()

	wantLines := bufio.NewScanner(bytes.NewBuffer(jsonMust(want)))
	wantLines.Split(bufio.ScanLines)
	gotLines := bufio.NewScanner(bytes.NewBuffer(jsonMust(got)))
	gotLines.Split(bufio.ScanLines)

	out := new(strings.Builder)
	out.WriteString("Comparison of objects:\n")
	for wantLines.Scan() {
		if !gotLines.Scan() {
			return
		}
		indicator := "    "
		if wantLines.Text() != gotLines.Text() {
			indicator = "!! >"
		}

		fmt.Fprintf(out, "\nwant %s %s\ngot  %s %s", indicator, wantLines.Text(), indicator, gotLines.Text())

	}

	t.Log(out.String())
}

func TestParser_runParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Workspace
		wantErr bool
	}{
		{
			name: "empty workspace",
			input: `
				workspace "foo" {}
			`,
			want: &Workspace{
				baseEntity: baseEntity{
					Name: "foo",
				},
			},
		},
		{
			name:  "multi-line description of workspace",
			input: "workspace 'foo' {\n\tdescription `this\n\t\tis very hard\n\t\tto test`\n}",
			want: &Workspace{
				baseEntity: baseEntity{
					Name:        "foo",
					Description: "this\nis very hard\nto test",
				},
			},
		},
		{
			name:  "multi-line description of workspace 2",
			input: "workspace 'foo' {\n\tdescription `\n\t\tthis\n\t\tis very hard\n\t\tto test\n\t`\n}",
			want: &Workspace{
				baseEntity: baseEntity{
					Name:        "foo",
					Description: "this\nis very hard\nto test",
				},
			},
		},
		{
			name: "model declaration and children",
			input: `
				workspace 'foo' {
					model {
						a -> b
					}
				}
			`,
			want: &Workspace{
				baseEntity: baseEntity{
					Name: "foo",
				},
				Model: &Model{
					baseEntity: baseEntity{
						relationshipEntity: relationshipEntity{
							Relationships: []*Relationship{
								{
									SourceId:      "a",
									DestinationId: "b",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "relationship with details",
			input: `
				workspace 'foo' {
					model {
						a -> b 'magic' 'pixie-dust' {
							tags 'foo' 'bar'
						}
						b -> c 'but how?' {
							technology 'space-mana'
							tags 'blinding,powerful'
						}
					}
				}
			`,
			want: &Workspace{
				baseEntity: baseEntity{
					Name: "foo",
				},
				Model: &Model{
					baseEntity: baseEntity{
						relationshipEntity: relationshipEntity{
							[]*Relationship{
								{
									SourceId:      "a",
									DestinationId: "b",
									baseEntity: baseEntity{
										Description: "magic",
										Technology:  "pixie-dust",
										Tags:        []string{"foo", "bar"},
									},
								},
								{
									SourceId:      "b",
									DestinationId: "c",
									baseEntity: baseEntity{
										Description: "but how?",
										Technology:  "space-mana",
										Tags:        []string{"blinding", "powerful"},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "properties",
			input: `
				workspace 'foo' {
					properties {
						'foo' 'bar'
					}
					model {
						softwareSystem 'my system' {
							properties {
								'bar' 'baz'
							}
						}
					}
				}
			`,
			want: &Workspace{
				baseEntity: baseEntity{
					Name: "foo",
					Properties: map[string]string{
						"foo": "bar",
					},
				},
				Model: &Model{
					baseEntity: baseEntity{
						childEntities: childEntities{
							NamedEntities: map[IdentifierString]Entity{
								"_softwaresystem00_my_system": &SoftwareSystem{
									baseEntity{
										Name:    "my system",
										LocalId: "_softwaresystem00_my_system",
										Properties: map[string]string{
											"bar": "baz",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "container parser",
			input: `
				workspace 'foo' {
					model {
						a = softwareSystem 'my system' {
							b = container "container name" "description" "tech" "tag1,tag2"
						}
					}
				}
			`,
			want: &Workspace{
				baseEntity: baseEntity{
					Name: "foo",
				},
				Model: &Model{
					baseEntity: baseEntity{

						childEntities: childEntities{
							NamedEntities: map[IdentifierString]Entity{

								"a": &SoftwareSystem{
									baseEntity: baseEntity{
										Name:    "my system",
										LocalId: "a",

										childEntities: childEntities{
											NamedEntities: map[IdentifierString]Entity{

												"b": &Container{
													baseEntity: baseEntity{
														LocalId:     "b",
														Name:        "container name",
														Description: "description",
														Technology:  "tech",
														Tags:        []string{"tag1", "tag2"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "short description declaration",
			input: "workspace 'foo' 'description of workspace' {}",
			want: &Workspace{
				baseEntity: baseEntity{
					Name:        "foo",
					Description: "description of workspace",
				},
			},
		},
		{
			name:    "multiline unacceptable description",
			input:   "workspace 'foo' {\nproperties {\n'key' `values are\nnot allowed to be\nmulti-line`\n}\n}",
			wantErr: true,
		},
		{
			name: "illegal identifier",
			input: `workspace {
				model {
					_anon = softwaresystem 'my sys'
				}
			}`,
			wantErr: true,
		},
		{
			name: "surprise terminator",
			input: `workspace {
				model
				{
					a -> b
				}
			}`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := new(Parser)

			// l := new(lexer.Lexer)
			// if err := l.Run(bytes.NewReader([]byte(tt.input))); err != nil {
			// 	t.Fatalf("lexer error on test input: %s", err)
			// }

			mts := &mockDependencies{lexers: make(map[string]*lexer.Lexer), sources: map[string]string{"test": tt.input}}
			got, err := p.Run("test", mts)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Parser.runParse() error = %v", err)
				return
			}
			if tt.wantErr {
				t.Log(err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Returned objects don't match")
				visualCompare(t, tt.want, got, 3)
			}
		})
	}
}

func TestParseInclude(t *testing.T) {

	sources := map[string]string{
		"base.c4": `
			workspace {
				model{
					a = softwaresystem 'a' {
						properties {
							"foo" "bar"
						}
					}
					b = #include 'include.c4'
					#include 'include.c4'
				}
			}
		`,
		"include.c4": `softwaresystem 'remoteB' { properties { "bar" "baz"; } }`,
	}

	want := &Workspace{
		Model: &Model{
			baseEntity: baseEntity{
				childEntities: childEntities{
					NamedEntities: map[IdentifierString]Entity{
						"a": &SoftwareSystem{
							baseEntity{
								Name:    "a",
								LocalId: "a",
								Properties: map[string]string{
									"foo": "bar",
								},
							},
						},
						"b": &SoftwareSystem{
							baseEntity{
								Name:    "remoteB",
								LocalId: "b",
								Properties: map[string]string{
									"bar": "baz",
								},
							},
						},
						"_softwaresystem00_remoteb": &SoftwareSystem{
							baseEntity{
								Name:    "remoteB",
								LocalId: "_softwaresystem00_remoteb",
								Properties: map[string]string{
									"bar": "baz",
								},
							},
						},
					},
				},
			},
		},
	}

	p := new(Parser)
	_ = 1

	mts := &mockDependencies{lexers: make(map[string]*lexer.Lexer), sources: sources}
	got, err := p.Run("base.c4", mts)

	if err != nil {
		var e *ExpectationError
		if errors.As(err, &e) {
			t.Logf("at token: %s", e.gotToken.String())
		}
		t.Fatalf("Parser.runParse() error = %v", err)
		return
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Returned objects don't match")
		visualCompare(t, want, got, 3)
	}

}
