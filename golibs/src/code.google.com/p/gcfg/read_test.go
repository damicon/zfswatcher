package gcfg

import (
	"fmt"
	"reflect"
	"testing"
)

const (
	// 64 spaces
	sp64 = "                                                                "
	// 512 spaces
	sp512 = sp64 + sp64 + sp64 + sp64 + sp64 + sp64 + sp64 + sp64
	// 4096 spaces
	sp4096 = sp512 + sp512 + sp512 + sp512 + sp512 + sp512 + sp512 + sp512
)

type sect01 struct{ Name string }
type conf01 struct{ Section sect01 }

type sect02 struct{ Bool bool }
type conf02 struct{ Section sect02 }

type sect03 struct{ Hyphen_In_Name string }
type conf03 struct{ Hyphen_In_Section sect03 }

type sect04 struct{ Name string }
type conf04 struct{ Sub map[string]*sect04 }

type readtest struct {
	gcfg string
	exp  interface{}
	ok   bool
}

var readtests = []struct {
	group string
	tests []readtest
}{{"basic", []readtest{
	// string value
	{"[section]\nname=value", &conf01{sect01{"value"}}, true},
	{"[section]\nname=", &conf01{sect01{""}}, true},
	// non-string value
	{"[section]\nbool=true", &conf02{sect02{true}}, true},
	// default value (true)
	{"[section]\nbool", &conf02{sect02{true}}, true},
	// hyphen in name
	{"[hyphen-in-section]\nhyphen-in-name=value", &conf03{sect03{"value"}}, true},
	// quoted string value
	{"[section]\nname=\"\"", &conf01{sect01{""}}, true},
	{"[section]\nname=\" \"", &conf01{sect01{" "}}, true},
	{"[section]\nname=\"value\"", &conf01{sect01{"value"}}, true},
	{"[section]\nname=\" value \"", &conf01{sect01{" value "}}, true},
	{"\n[section]\nname=\"value ; cmnt\"", &conf01{sect01{"value ; cmnt"}}, true},
	{"[section]\nname=\"val\" \"ue\"", &conf01{sect01{"val ue"}}, true},
	// escape sequences
	{"[section]\nname=\"va\\\\lue\"", &conf01{sect01{"va\\lue"}}, true},
	{"[section]\nname=\"va\\\"lue\"", &conf01{sect01{"va\"lue"}}, true},
	{"[section]\nname=\"va\\nlue\"", &conf01{sect01{"va\nlue"}}, true},
	{"[section]\nname=\"va\\tlue\"", &conf01{sect01{"va\tlue"}}, true},
	// broken line
	{"[section]\nname=value \\\n value", &conf01{sect01{"value  value"}}, true},
	// long lines
	{sp4096 + "[section]\nname=value\n", &conf01{sect01{"value"}}, true},
	{"[" + sp4096 + "section]\nname=value\n", &conf01{sect01{"value"}}, true},
	{"[section" + sp4096 + "]\nname=value\n", &conf01{sect01{"value"}}, true},
	{"[section]" + sp4096 + "\nname=value\n", &conf01{sect01{"value"}}, true},
	{"[section]\n" + sp4096 + "name=value\n", &conf01{sect01{"value"}}, true},
	{"[section]\nname" + sp4096 + "=value\n", &conf01{sect01{"value"}}, true},
	{"[section]\nname=" + sp4096 + "value\n", &conf01{sect01{"value"}}, true},
	{"[section]\nname=value\n" + sp4096, &conf01{sect01{"value"}}, true},
}}, {"bool", []readtest{
	{"[section]\nbool=true", &conf02{sect02{true}}, true},
	{"[section]\nbool=yes", &conf02{sect02{true}}, true},
	{"[section]\nbool=on", &conf02{sect02{true}}, true},
	{"[section]\nbool=1", &conf02{sect02{true}}, true},
	{"[section]\nbool=false", &conf02{sect02{false}}, true},
	{"[section]\nbool=no", &conf02{sect02{false}}, true},
	{"[section]\nbool=off", &conf02{sect02{false}}, true},
	{"[section]\nbool=0", &conf02{sect02{false}}, true},
	{"[section]\nbool=t", &conf02{}, false},
	{"[section]\nbool=truer", &conf02{}, false},
	{"[section]\nbool=-1", &conf02{}, false},
}}, {"whitespace", []readtest{
	{" \n[section]\nbool=true", &conf02{sect02{true}}, true},
	{" [section]\nbool=true", &conf02{sect02{true}}, true},
	{"\t[section]\nbool=true", &conf02{sect02{true}}, true},
	{"[ section]\nbool=true", &conf02{sect02{true}}, true},
	{"[section ]\nbool=true", &conf02{sect02{true}}, true},
	{"[section]\n bool=true", &conf02{sect02{true}}, true},
	{"[section]\nbool =true", &conf02{sect02{true}}, true},
	{"[section]\nbool= true", &conf02{sect02{true}}, true},
	{"[section]\nbool=true ", &conf02{sect02{true}}, true},
	{"[section]\r\nbool=true", &conf02{sect02{true}}, true},
	{"[section]\r\nbool=true\r\n", &conf02{sect02{true}}, true},
	{";cmnt\r\n[section]\r\nbool=true\r\n", &conf02{sect02{true}}, true},
}}, {"comments", []readtest{
	{"; cmnt\n[section]\nname=value", &conf01{sect01{"value"}}, true},
	{"# cmnt\n[section]\nname=value", &conf01{sect01{"value"}}, true},
	{" ; cmnt\n[section]\nname=value", &conf01{sect01{"value"}}, true},
	{"\t; cmnt\n[section]\nname=value", &conf01{sect01{"value"}}, true},
	{"\n[section]; cmnt\nname=value", &conf01{sect01{"value"}}, true},
	{"\n[section] ; cmnt\nname=value", &conf01{sect01{"value"}}, true},
	{"\n[section]\nname=value; cmnt", &conf01{sect01{"value"}}, true},
	{"\n[section]\nname=value ; cmnt", &conf01{sect01{"value"}}, true},
	{"\n[section]\nname=\"value\" ; cmnt", &conf01{sect01{"value"}}, true},
	{"\n[section]\nname=value ; \"cmnt", &conf01{sect01{"value"}}, true},
	{"\n[section]\nname=\"value ; cmnt\" ; cmnt", &conf01{sect01{"value ; cmnt"}}, true},
	{"\n[section]\nname=; cmnt", &conf01{sect01{""}}, true},
}}, {"subsections", []readtest{
	{"\n[sub \"A\"]\nname=value", &conf04{map[string]*sect04{"A": &sect04{"value"}}}, true},
	{"\n[sub \"b\"]\nname=value", &conf04{map[string]*sect04{"b": &sect04{"value"}}}, true},
	{"\n[sub \"A\\\\\"]\nname=value", &conf04{map[string]*sect04{"A\\": &sect04{"value"}}}, true},
	{"\n[sub \"A\\\"\"]\nname=value", &conf04{map[string]*sect04{"A\"": &sect04{"value"}}}, true},
}}, {"errors", []readtest{
	// error: invalid line
	{"\n[section]\n=", &conf01{}, false},
	// error: no section
	{"name=value", &conf01{}, false},
	// error: failed to parse
	{"\n[section]\nbool=maybe", &conf02{sect02{}}, false},
	// error: empty section
	{"\n[]\nname=value", &conf01{}, false},
	// error: empty subsection
	{"\n[sub \"\"]\nname=value", &conf04{}, false},
	// error: section name not matched
	{"\n[nonexistent]\nname=value", &conf01{}, false},
	// error: subsection name not matched
	{"\n[section \"nonexistent\"]\nname=value", &conf01{}, false},
	// error: variable name not matched
	{"\n[section]\nnonexistent=value", &conf01{}, false},
	// error: missing end quote
	{"[section]\nname=\"value", &conf01{}, false},
	// error: invalid escape
	{"\n[section]\nname=\\", &conf01{}, false},
	{"\n[section]\nname=\\a", &conf01{}, false},
	{"\n[section]\nname=\"val\\a\"", &conf01{}, false},
	{"\n[section]\nname=val\\", &conf01{}, false},
	{"\n[sub \"A\\\n\"]\nname=value", &conf04{}, false},
	{"\n[sub \"A\\\t\"]\nname=value", &conf04{}, false},
	// error: invalid broken line
	{"[section]\nname=\"value \\\n value\"", &conf01{}, false},
}},
}

func TestReadStringInto(t *testing.T) {
	for _, tg := range readtests {
		for i, tt := range tg.tests {
			id := fmt.Sprintf("%s:%d", tg.group, i)
			// get the type of the expected result
			restyp := reflect.TypeOf(tt.exp).Elem()
			// create a new instance to hold the actual result
			res := reflect.New(restyp).Interface()
			err := ReadStringInto(res, tt.gcfg)
			if tt.ok {
				if err != nil {
					t.Errorf("%s fail: got error %v, wanted ok", id, err)
					continue
				} else if !reflect.DeepEqual(res, tt.exp) {
					t.Errorf("%s fail: got value %#v, wanted value %#v", id, res, tt.exp)
					continue
				}
				if !testing.Short() {
					t.Logf("%s pass: got value %#v", id, res)
				}
			} else { // !tt.ok
				if err == nil {
					t.Errorf("%s fail: got value %#v, wanted error", id, res)
					continue
				}
				if !testing.Short() {
					t.Logf("%s pass: got error %v", id, err)
				}
			}
		}
	}
}

func TestReadFileInto(t *testing.T) {
	res := &struct{ Section struct{ Name string } }{}
	err := ReadFileInto(res, "testdata/gcfg_test.gcfg")
	if err != nil {
		t.Errorf(err.Error())
	}
	if "value" != res.Section.Name {
		t.Errorf("got %q, wanted %q", res.Section.Name, "value")
	}
}
