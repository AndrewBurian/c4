package loader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
)

type directiveHandler func(context.Context, *sourceLoader, *bytes.Buffer, ...[]byte) error

type lineModifiers func(*[]byte) error

func (l *sourceLoader) process(ctx context.Context, reader *bytes.Buffer) (*bytes.Buffer, error) {

	directives := map[string]directiveHandler{
		"#define":  defineConstant,
		"#include": includeFile,
		"#envvar":  defineConstantFromEnv,
	}

	output := bytes.NewBuffer(make([]byte, 0, reader.Len()))

	for {
		line, eof := reader.ReadBytes('\n')

		for _, mod := range l.lineMods {
			err := mod(&line)
			if err != nil {
				return nil, err
			}
		}

		fields := bytes.Fields(line)
		if len(fields) > 1 {
			if handler, isDirective := directives[string(fields[0])]; isDirective {
				err := handler(ctx, l, output, fields[1:]...)
				if err != nil {
					return nil, err
				}
				continue
			}
		}

		output.Write(line)

		if eof == io.EOF {
			break
		}
	}

	return output, nil
}

func defineConstant(_ context.Context, l *sourceLoader, output *bytes.Buffer, args ...[]byte) error {
	if len(args) != 2 {
		return fmt.Errorf("error processing #define, expected two arguments")
	}

	f := func(line *[]byte) error {
		*line = bytes.ReplaceAll(*line, []byte(args[0]), []byte(args[1]))
		return nil
	}

	l.lineMods = append(l.lineMods, f)
	output.WriteString(fmt.Sprintf("#define %s %s\n", args[0], args[1]))
	return nil
}

func includeFile(ctx context.Context, l *sourceLoader, output *bytes.Buffer, args ...[]byte) error {
	if len(args) != 1 {
		return fmt.Errorf("error processing #include, expected one argument")
	}

	includeData, err := l.Load(ctx, string(args[0]))
	if err != nil {
		return fmt.Errorf("error including file: %w", err)
	}

	_, err = output.ReadFrom(includeData)
	if err != nil {
		return fmt.Errorf("error including file bytes: %w", err)
	}

	return nil
}

func defineConstantFromEnv(ctx context.Context, l *sourceLoader, output *bytes.Buffer, args ...[]byte) error {
	for _, env := range args {
		if replace, set := os.LookupEnv(string(env)); set {
			err := defineConstant(ctx, l, output, env, []byte(replace))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
