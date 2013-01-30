package gcfg_test

import (
	"fmt"
	"log"
)

import "code.google.com/p/gcfg"

func ExampleReadStringInto() {
	cfgStr := `; Comment line
[section]
name=value # comment`
	cfg := struct {
		Section struct {
			Name string
		}
	}{}
	err := gcfg.ReadStringInto(&cfg, cfgStr)
	if err != nil {
		log.Fatalf("Failed to parse gcfg data: %s", err)
	}
	fmt.Println(cfg.Section.Name)
	// Output: value
}

func ExampleReadStringInto_bool() {
	cfgStr := `; Comment line
[section]
switch=on`
	cfg := struct {
		Section struct {
			Switch bool
		}
	}{}
	err := gcfg.ReadStringInto(&cfg, cfgStr)
	if err != nil {
		log.Fatalf("Failed to parse gcfg data: %s", err)
	}
	fmt.Println(cfg.Section.Switch)
	// Output: true
}

func ExampleReadStringInto_subsections() {
	cfgStr := `; Comment line
[profile "A"]
color = white

[profile "B"]
color = black
`
	cfg := struct {
		Profile map[string]*struct {
			Color string
		}
	}{}
	err := gcfg.ReadStringInto(&cfg, cfgStr)
	if err != nil {
		log.Fatalf("Failed to parse gcfg data: %s", err)
	}
	fmt.Printf("%s %s\n", cfg.Profile["A"].Color, cfg.Profile["B"].Color)
	// Output: white black
}
