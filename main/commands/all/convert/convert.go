package convert

import (
	"github.com/xtls/xray-core/main/commands/base"
)

// CmdAPI calls an API in an Xray process
var CmdConvert = &base.Command{
	UsageLine: "{{.Exec}} convert",
	Short:     "Convert configs",
	Long: `{{.Exec}} {{.LongName}} provides tools to convert config.
`,
	Commands: []*base.Command{
		cmdProtobuf,
	},
}
