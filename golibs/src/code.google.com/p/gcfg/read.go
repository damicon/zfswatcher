package gcfg

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

import (
	"code.google.com/p/gcfg/scanner"
	"code.google.com/p/gcfg/token"
)

var unescape = map[rune]rune{'\\': '\\', '"': '"', 'n': '\n', 't': '\t'}

// no error: invalid literals should be caught by scanner
func unquote(s string) string {
	u, q, esc := make([]rune, 0, len(s)), false, false
	for _, c := range s {
		if esc {
			uc, ok := unescape[c]
			switch {
			case ok:
				u = append(u, uc)
				fallthrough
			case !q && c == '\n':
				esc = false
				continue
			}
			panic("invalid escape sequence")
		}
		switch c {
		case '"':
			q = !q
		case '\\':
			esc = true
		default:
			u = append(u, c)
		}
	}
	if q {
		panic("missing end quote")
	}
	if esc {
		panic("invalid escape sequence")
	}
	return string(u)
}

func readInto(config interface{}, fset *token.FileSet, file *token.File, src []byte) error {
	var s scanner.Scanner
	var errs scanner.ErrorList
	s.Init(file, src, func(p token.Position, m string) { errs.Add(p, m) }, 0)
	sect, sectsub := "", ""
	pos, tok, lit := s.Scan()
	errfn := func(msg string) error {
		return fmt.Errorf("%s: %s", fset.Position(pos), msg)
	}
	for {
		if errs.Len() > 0 {
			return errs.Err()
		}
		switch tok {
		case token.EOF:
			return nil
		case token.EOL, token.COMMENT:
			pos, tok, lit = s.Scan()
		case token.LBRACK:
			pos, tok, lit = s.Scan()
			if errs.Len() > 0 {
				return errs.Err()
			}
			if tok != token.IDENT {
				return errfn("expected section name")
			}
			sect, sectsub = lit, ""
			pos, tok, lit = s.Scan()
			if errs.Len() > 0 {
				return errs.Err()
			}
			if tok == token.STRING {
				sectsub = unquote(lit)
				if sectsub == "" {
					return errfn("empty subsection name")
				}
				pos, tok, lit = s.Scan()
				if errs.Len() > 0 {
					return errs.Err()
				}
			}
			if tok != token.RBRACK {
				if sectsub == "" {
					return errfn("expected subsection name or right bracket")
				}
				return errfn("expected right bracket")
			}
			pos, tok, lit = s.Scan()
			if tok != token.EOL && tok != token.EOF && tok != token.COMMENT {
				return errfn("expected EOL, EOF, or comment")
			}
		case token.IDENT:
			if sect == "" {
				return errfn("expected section header")
			}
			n := lit
			pos, tok, lit = s.Scan()
			if errs.Len() > 0 {
				return errs.Err()
			}
			var v string
			if tok == token.EOF || tok == token.EOL || tok == token.COMMENT {
				v = defaultValue
			} else {
				if tok != token.ASSIGN {
					return errfn("expected '='")
				}
				pos, tok, lit = s.Scan()
				if errs.Len() > 0 {
					return errs.Err()
				}
				if tok != token.STRING {
					return errfn("expected value")
				}
				v = unquote(lit)
				pos, tok, lit = s.Scan()
				if errs.Len() > 0 {
					return errs.Err()
				}
				if tok != token.EOL && tok != token.EOF && tok != token.COMMENT {
					return errfn("expected EOL, EOF, or comment")
				}
			}
			err := set(config, sect, sectsub, n, v)
			if err != nil {
				return err
			}
		default:
			if sect == "" {
				return errfn("expected section header")
			}
			return errfn("expected section header or variable declaration")
		}
	}
	panic("never reached")
}

// ReadInto reads gcfg formatted data from reader and sets the values into the
// corresponding fields in config.
//
// Config must be a pointer to a struct.
// Each section corresponds to a struct field in config, and each variable in a
// section corresponds to a data field in the section struct.
// The name of the field must match the name of the section or variable,
// ignoring case.
// Hyphens in section and variable names correspond to underscores in field
// names.
//
// For sections with subsections, the corresponding field in config must be a
// map, rather than a struct, with string keys and pointer-to-struct values.
// Values for subsection variables are stored in the map with the subsection
// name used as the map key.
// (Note that unlike section and variable names, subsection names are case
// sensitive.)
// When using a map, and there is a section with the same section name but
// without a subsection name, its values are stored with the empty string used
// as the key.
//
// The section structs in the config struct may contain arbitrary types.
// For string fields, the (unquoted and unescaped) value string is assigned to
// the field.
// For bool fields, the field is set to true if the value is "true", "yes", "on"
// or "1", and set to false if the value is "false", "no", "off" or "0",
// ignoring case.
// For all other types, fmt.Sscanf with the verb "%v" is used to parse the value
// string and set it to the field.
// This means that built-in Go types are parseable using the standard format,
// and any user-defined type is parseable if it implements the fmt.Scanner
// interface.
// Note that the value is considered invalid unless fmt.Scanner fully consumes
// the value string without error.
//
// ReadInto panics if config is not a pointer to a struct, or if it encounters a
// field that is not of a suitable type (either a struct or a map with string
// keys and pointer-to-struct values).
//
// See ReadStringInto for examples.
//
func ReadInto(config interface{}, reader io.Reader) error {
	src, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	return readInto(config, fset, file, src)
}

// ReadStringInto reads gcfg formatted data from str and sets the values into
// the corresponding fields in config.
// ReadStringInfo is a wrapper for ReadInfo; see ReadInto(config, reader) for
// detailed description of how data is read and set into config.
func ReadStringInto(config interface{}, str string) error {
	r := strings.NewReader(str)
	return ReadInto(config, r)
}

// ReadFileInto reads gcfg formatted data from the file filename and sets the
// values into the corresponding fields in config.
// ReadFileInto is a wrapper for ReadInfo; see ReadInto(config, reader) for
// detailed description of how data is read and set into config.
func ReadFileInto(config interface{}, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	src, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	fset := token.NewFileSet()
	file := fset.AddFile(filename, fset.Base(), len(src))
	return readInto(config, fset, file, src)
}
