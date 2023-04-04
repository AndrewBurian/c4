package parser

import (
	"encoding/json"
	"reflect"
	"testing"

	"go.burian.dev/c4arch/internal/lexer"
)

func jsonMust(t *testing.T, o any) string {
	t.Helper()
	b, err := json.Marshal(o)
	if err != nil {
		panic(err)
	}
	return string(b)
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
			input: "workspace 'foo' { description `this\n\tis very hard\n\tto test`}",
			want: &Workspace{
				baseEntity: baseEntity{
					Name:        "foo",
					Description: "this\nis very hard\nto test",
				},
			},
		},
		{
			name:  "multi-line description of workspace 2",
			input: "workspace 'foo' { description `\n\tthis\n\tis very hard\n\tto test\n\n`}",
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
					relationshipEntity: relationshipEntity{
						Relationships: []*Relationship{
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
		{
			name: "container parser",
			input: `
				workspace 'foo' {
					model {
						a = softwareSystem {
							b = container "help" "desc" "tech" "tag1,tag2"
						}
					}
				}
			`,
			want: &Workspace{
				baseEntity: baseEntity{
					Name: "foo",
				},
			},
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

			got, err := p.runParse()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.runParse() error = %v", err)
				return
			}
			if !reflect.DeepEqual(got[0], tt.want) {
				t.Errorf("Parser.runParse() \ngot = %s\nwant = %s", jsonMust(t, got[0]), jsonMust(t, tt.want))

			}
		})
	}
}
