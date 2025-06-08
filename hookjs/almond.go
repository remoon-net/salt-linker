package hookjs

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/dop251/goja"
)

//go:embed almond.js
var almondJs string

var AlmondProg = goja.MustCompile("almond.js", almondJs, false)

//go:embed test.js
var tJs string

func FixAlmondDefine(m, s string) string {
	m = fmt.Sprintf(`define("%s",`, m)
	s = strings.Replace(s, `define(`, m, 1)
	return s
}
