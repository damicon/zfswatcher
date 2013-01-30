package gcfg

import (
	"fmt"
)

type gbool bool

var gboolValues = map[string]interface{}{
	"true": true, "yes": true, "on": true, "1": true,
	"false": false, "no": false, "off": false, "0": false}

func (b *gbool) Scan(state fmt.ScanState, verb rune) error {
	v, err := scanEnum(state, gboolValues, true)
	if err != nil {
		return err
	}
	bb, _ := v.(bool) // cannot be non-bool
	*b = gbool(bb)
	return nil
}
