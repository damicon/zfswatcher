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

type cBasic struct {
	Section           cBasicS1
	Hyphen_In_Section cBasicS2
}
type cBasicS1 struct {
	Name string
	Int  int
}
type cBasicS2 struct {
	Hyphen_In_Name string
}

type cSubs struct{ Sub map[string]*cSubsS1 }
type cSubsS1 struct{ Name string }

type cBool struct{ Section cBoolS1 }
type cBoolS1 struct{ Bool bool }

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
	{"[section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\nname=", &cBasic{Section: cBasicS1{Name: ""}}, true},
	// non-string value
	{"[section]\nint=1", &cBasic{Section: cBasicS1{Int: 1}}, true},
	// hyphen in name
	{"[hyphen-in-section]\nhyphen-in-name=value", &cBasic{Hyphen_In_Section: cBasicS2{"value"}}, true},
	// quoted string value
	{"[section]\nname=\"\"", &cBasic{Section: cBasicS1{Name: ""}}, true},
	{"[section]\nname=\" \"", &cBasic{Section: cBasicS1{Name: " "}}, true},
	{"[section]\nname=\"value\"", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\nname=\" value \"", &cBasic{Section: cBasicS1{Name: " value "}}, true},
	{"\n[section]\nname=\"va ; lue\"", &cBasic{Section: cBasicS1{Name: "va ; lue"}}, true},
	{"[section]\nname=\"val\" \"ue\"", &cBasic{Section: cBasicS1{Name: "val ue"}}, true},
	// escape sequences
	{"[section]\nname=\"va\\\\lue\"", &cBasic{Section: cBasicS1{Name: "va\\lue"}}, true},
	{"[section]\nname=\"va\\\"lue\"", &cBasic{Section: cBasicS1{Name: "va\"lue"}}, true},
	{"[section]\nname=\"va\\nlue\"", &cBasic{Section: cBasicS1{Name: "va\nlue"}}, true},
	{"[section]\nname=\"va\\tlue\"", &cBasic{Section: cBasicS1{Name: "va\tlue"}}, true},
	// broken line
	{"[section]\nname=value \\\n value", &cBasic{Section: cBasicS1{Name: "value  value"}}, true},
}}, {"whitespace", []readtest{
	{" \n[section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{" [section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\t[section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[ section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section ]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\n name=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\nname =value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\nname= value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\nname=value ", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\r\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\r\nname=value\r\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{";cmnt\r\n[section]\r\nname=value\r\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	// long lines
	{sp4096 + "[section]\nname=value\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[" + sp4096 + "section]\nname=value\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section" + sp4096 + "]\nname=value\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]" + sp4096 + "\nname=value\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\n" + sp4096 + "name=value\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\nname" + sp4096 + "=value\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\nname=" + sp4096 + "value\n", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"[section]\nname=value\n" + sp4096, &cBasic{Section: cBasicS1{Name: "value"}}, true},
}}, {"comments", []readtest{
	{"; cmnt\n[section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"# cmnt\n[section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{" ; cmnt\n[section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\t; cmnt\n[section]\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\n[section]; cmnt\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\n[section] ; cmnt\nname=value", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\n[section]\nname=value; cmnt", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\n[section]\nname=value ; cmnt", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\n[section]\nname=\"value\" ; cmnt", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\n[section]\nname=value ; \"cmnt", &cBasic{Section: cBasicS1{Name: "value"}}, true},
	{"\n[section]\nname=\"va ; lue\" ; cmnt", &cBasic{Section: cBasicS1{Name: "va ; lue"}}, true},
	{"\n[section]\nname=; cmnt", &cBasic{Section: cBasicS1{Name: ""}}, true},
}}, {"subsections", []readtest{
	{"\n[sub \"A\"]\nname=value", &cSubs{map[string]*cSubsS1{"A": &cSubsS1{"value"}}}, true},
	{"\n[sub \"b\"]\nname=value", &cSubs{map[string]*cSubsS1{"b": &cSubsS1{"value"}}}, true},
	{"\n[sub \"A\\\\\"]\nname=value", &cSubs{map[string]*cSubsS1{"A\\": &cSubsS1{"value"}}}, true},
	{"\n[sub \"A\\\"\"]\nname=value", &cSubs{map[string]*cSubsS1{"A\"": &cSubsS1{"value"}}}, true},
}}, {"errors", []readtest{ // scanning/parsing errors (except value parsing)
	// invalid line
	{"\n[section]\n=", &cBasic{}, false},
	// no section
	{"name=value", &cBasic{}, false},
	// failed to parse
	{"\n[section]\nbool=maybe", &cBool{cBoolS1{}}, false},
	// empty section
	{"\n[]\nname=value", &cBasic{}, false},
	// empty subsection
	{"\n[sub \"\"]\nname=value", &cSubs{}, false},
	// section name not matched
	{"\n[nonexistent]\nname=value", &cBasic{}, false},
	// subsection name not matched
	{"\n[section \"nonexistent\"]\nname=value", &cBasic{}, false},
	// variable name not matched
	{"\n[section]\nnonexistent=value", &cBasic{}, false},
	// missing end quote
	{"[section]\nname=\"value", &cBasic{}, false},
	// invalid escape
	{"\n[section]\nname=\\", &cBasic{}, false},
	{"\n[section]\nname=\\a", &cBasic{}, false},
	{"\n[section]\nname=\"val\\a\"", &cBasic{}, false},
	{"\n[section]\nname=val\\", &cBasic{}, false},
	{"\n[sub \"A\\\n\"]\nname=value", &cSubs{}, false},
	{"\n[sub \"A\\\t\"]\nname=value", &cSubs{}, false},
	// invalid broken line
	{"[section]\nname=\"value \\\n value\"", &cBasic{}, false},
}}, {"bool", []readtest{
	// explicit values
	{"[section]\nbool=true", &cBool{cBoolS1{true}}, true},
	{"[section]\nbool=yes", &cBool{cBoolS1{true}}, true},
	{"[section]\nbool=on", &cBool{cBoolS1{true}}, true},
	{"[section]\nbool=1", &cBool{cBoolS1{true}}, true},
	{"[section]\nbool=false", &cBool{cBoolS1{false}}, true},
	{"[section]\nbool=no", &cBool{cBoolS1{false}}, true},
	{"[section]\nbool=off", &cBool{cBoolS1{false}}, true},
	// default value (true)
	{"[section]\nbool", &cBool{cBoolS1{true}}, true},
	{"[section]\nbool=0", &cBool{cBoolS1{false}}, true},
	// bool parse errors
	{"[section]\nbool=t", &cBool{}, false},
	{"[section]\nbool=truer", &cBool{}, false},
	{"[section]\nbool=-1", &cBool{}, false},
}},
}

func TestReadStringInto(t *testing.T) {
	for _, tg := range readtests {
		for i, tt := range tg.tests {
			id := fmt.Sprintf("%s:%d", tg.group, i)
			testRead(t, id, tt)
		}
	}
}

func testRead(t *testing.T, id string, tt readtest) {
	// get the type of the expected result
	restyp := reflect.TypeOf(tt.exp).Elem()
	// create a new instance to hold the actual result
	res := reflect.New(restyp).Interface()
	err := ReadStringInto(res, tt.gcfg)
	if tt.ok {
		if err != nil {
			t.Errorf("%s fail: got error %v, wanted ok", id, err)
			return
		} else if !reflect.DeepEqual(res, tt.exp) {
			t.Errorf("%s fail: got value %#v, wanted value %#v", id, res, tt.exp)
			return
		}
		if !testing.Short() {
			t.Logf("%s pass: got value %#v", id, res)
		}
	} else { // !tt.ok
		if err == nil {
			t.Errorf("%s fail: got value %#v, wanted error", id, res)
			return
		}
		if !testing.Short() {
			t.Logf("%s pass: got error %v", id, err)
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
