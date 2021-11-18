package apic

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/encoding/gocode/gocodec"
)

var r cue.Runtime
var codec = gocodec.New(&r, nil)
var myValueConstraints cue.Value

const appSchema = `
GetOutput: {
    id: >=0
    email: email
    born: >= 1900 <= 2019
}
`

type AppRequest struct {
	Test string
}

type AppResponse struct {
	Test string `cue:"c-b" json:"a,omitempty"`
}

// func (v *AppResponse) Validate() error {
// 	codec.
//     // return codec.Validate(myValueConstraints, x)
// }

// func loadSchema() error {
// 	instance, err := r.Compile("test", appSchema)
// 	if err != nil {
// 		return nil
// 	}
// 	return nil
// }
