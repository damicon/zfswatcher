// Package gcfg reads "gitconfig-like" text-based configuration files with
// "name=value" pairs grouped into sections (gcfg files).
// Support for writing gcfg files may be added later.
//
// See ReadInto and the examples to get an idea of how to use it.
//
// This package is still a work in progress, and both the supported syntax and
// the API is subject to change. See below for planned changes.
//
// The syntax is based on that used by git config:
// http://git-scm.com/docs/git-config#_syntax .
// There are some (planned) differences compared to the git config format:
//  - improve data portability:
//    - must be encoded in UTF-8 (for now) and must not contain the 0 byte
//    - include and "path" type is not supported
//      (path type may be implementable as a user-defined type)
//  - disallow potentially ambiguous or misleading definitions:
//    - `[sec.sub]` format is not allowed (deprecated in gitconfig)
//    - `[sec ""]` is not allowed
//      - use `[sec]` for section name "sec" and empty subsection name
//    - (planned) within a single file, definitions must be contiguous for each:
//      - section: '[secA]' -> '[secB]' -> '[secA]' is an error
//      - subsection: '[sec "A"]' -> '[sec "B"]' -> '[sec "A"]' is an error
//      - multivalued variable: 'multi=a' -> 'other=x' -> 'multi=b' is an error
//
// The package may be usable for handling some of the various "INI file" formats
// used by some programs and libraries, but achieving or maintaining
// compatibility with any of those is not a primary concern.
//
// TODO:
//  - format
//    - define valid section and variable names
//    - reconsider valid escape sequences
//      (gitconfig doesn't support \r in value, \t in subsection name, etc.)
//    - define handling of "implicit value" for types other than bool
//    - consider handling of numeric values (decimal only by default?)
//    - complete syntax documentation
//  - reading
//    - define internal representation structure
//    - support multi-value variables
//    - support multiple inputs (readers, strings, files)
//    - support declaring encoding (?)
//    - support automatic dereferencing of pointer fields (?)
//    - support varying fields sets for subsections (?)
//  - scanEnum
//    - should use longest match (?)
//    - support matching on unique prefix (?)
//  - writing gcfg files
//  - error handling
//    - report position of extra characters in value
//    - make error context accessible programmatically?
//    - limit input size?
//  - move TODOs to issue tracker (eventually)
//
package gcfg
