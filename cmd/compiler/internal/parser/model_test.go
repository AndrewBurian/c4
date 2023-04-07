package parser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"go.burian.dev/c4/compiler/lexer"
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

	for wantLines.Scan() {
		if !gotLines.Scan() {
			return
		}

		t.Logf("\nwant %s\ngot  %s", wantLines.Text(), gotLines.Text())

		if contextLines == 0 {
			return
		}
		if wantLines.Text() != gotLines.Text() {
			contextLines--
		}
	}
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
			p.reset()
			p.code = []byte(tt.input)
			p.lexer = lexer.NewLexer([]byte(tt.input))
			err := p.lexer.Run()
			if err != nil {
				t.Fatalf("Error running lexer: %s", err)
			}

			var got []*Workspace
			func() {
				defer func() {
					if panVal := recover(); panVal != nil {
						t.Fatalf("function panicked: %s", panVal)
					}
				}()
				got, err = p.runParse()
			}()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parser.runParse() error = %v", err)
				return
			}
			if tt.wantErr {
				t.Log(err)
				return
			}
			if !reflect.DeepEqual(got[0], tt.want) {
				t.Errorf("Returned objects don't match")
				visualCompare(t, tt.want, got[0], 3)
			}
		})
	}
}
