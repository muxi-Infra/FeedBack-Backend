package prompts

import _ "embed"

//go:embed reference/ccnubox.md
var CCNUBOXAppPrompts string

//go:embed reference/kstack.md
var KSTACKAppPrompts string

var m = map[string]string{
	"ccnubox": CCNUBOXAppPrompts,
	"kstack":  KSTACKAppPrompts,
}

func GetAppPrompts(appName string) (string, bool) {
	prompts, ok := m[appName]
	return prompts, ok
}
