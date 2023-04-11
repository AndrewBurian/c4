package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"go.burian.dev/c4/cmd/compiler/internal/lexer"
	"go.burian.dev/c4/cmd/compiler/internal/loader"
	"go.burian.dev/c4/cmd/compiler/internal/parser"
)

type compiler struct {
	sources    map[string][]byte
	tokens     map[string]*lexer.LexedSource
	workspaces map[string]*parser.Workspace

	loader loader.Loader
	lexer  *lexer.Lexer
	parser *parser.Parser

	context context.Context

	logger *log.Logger

	compileConfig
}

type compileConfig struct {
	outputFile string

	quiet      bool
	jsonPretty bool

	loader.SourceLoadConfig
}

var (
	defaultLoader = loader.NewLoader()
	defaultLexer  = new(lexer.Lexer)
	defaultParser = new(parser.Parser)
)

func main() {

	comp := new(compiler)

	flag.StringVar(&comp.outputFile, "out", "out.c4m", "set the output file for compilation")
	flag.BoolVar(&comp.quiet, "quiet", false, "only print error messages")
	flag.Parse()

	target := flag.Arg(0)
	if target == "" {
		flag.Usage()
		return
	}

	err := comp.Run(target)
	if err != nil {
		comp.logger.Fatalf("Compilation failed: %s", err)
	}
}

func (comp *compiler) Run(target string) error {

	comp.logger = log.New(os.Stderr, fmt.Sprintf("compiling %s: ", target), log.Lmsgprefix|log.Ltime)

	var cancel context.CancelFunc
	comp.context, cancel = context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	comp.logger.Println("Starting")

	// TODO The compiler's behaviour should be to run check not parse
	workspace, err := comp.GetWorkspaceFor(target)
	if err != nil {
		return fmt.Errorf("error compiling: %s", comp.prettyPrintError(err))
	}

	err = comp.WriteOutput(workspace)
	if err != nil {
		return fmt.Errorf("error writing compiled workspace: %s", comp.prettyPrintError(err))
	}

	comp.logger.Println("Compiled successfully")
	comp.logger.Printf("Wrote file to %s\n", comp.outputFile)

	return nil
}

func (c *compiler) GetSourceFor(target string) (*bytes.Reader, error) {
	if c.sources == nil {
		c.sources = make(map[string][]byte)
	}

	if source, has := c.sources[target]; has {
		c.logger.Printf("Fetching cached source for %s\n", target)
		return bytes.NewReader(source), nil
	}

	var load loader.Loader
	if c.loader != nil {
		load = c.loader
	} else {
		load = defaultLoader
	}

	c.logger.Printf("Fetching new source %s\n", target)
	loadedSource, err := load.Load(c.context, target)
	if err != nil {
		return nil, fmt.Errorf("compiler could not provide source: %w", err)
	}

	c.sources[target] = loadedSource

	return bytes.NewReader(loadedSource), nil

}

func (c *compiler) GetTokenStreamFor(target string) (lexer.TokenStream, error) {
	if c.tokens == nil {
		c.tokens = make(map[string]*lexer.LexedSource)
	}

	if tokens, has := c.tokens[target]; has {
		c.logger.Printf("Fetching cached token stream for %s\n", target)
		return tokens.TokenStream(), nil
	}

	var lexer *lexer.Lexer
	if c.lexer != nil {
		lexer = c.lexer
	} else {
		lexer = defaultLexer
	}

	c.logger.Printf("Lexing new source %s\n", target)
	lexedSource, err := lexer.Run(target, c)
	if err != nil {
		return nil, err
	}

	c.tokens[target] = lexedSource
	return lexedSource.TokenStream(), nil
}

func (c *compiler) GetWorkspaceFor(target string) (*parser.Workspace, error) {
	if c.workspaces == nil {
		c.workspaces = make(map[string]*parser.Workspace)
	}

	if workspace, has := c.workspaces[target]; has {
		c.logger.Printf("Fetching cached workspace %s\n", target)
		return workspace, nil
	}

	var parser *parser.Parser
	if c.parser != nil {
		parser = c.parser
	} else {
		parser = defaultParser
	}

	c.logger.Printf("Parsing new workspace %s\n", target)
	workspace, err := parser.Run(target, c)
	if err != nil {
		return nil, err
	}

	c.workspaces[target] = workspace
	return workspace, nil
}

func (c *compiler) WriteOutput(w *parser.Workspace) error {
	file, err := os.Create(c.outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	if c.jsonPretty {
		enc.SetIndent("", "\t")
	}
	return enc.Encode(w)
}
