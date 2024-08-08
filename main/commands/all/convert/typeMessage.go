package convert

import (
	"encoding/json"
	"fmt"
	"io"

	cserial "github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/main/commands/base"
	"github.com/xtls/xray-core/main/confloader"
	"google.golang.org/protobuf/encoding/protojson"
)

// usage:
// echo '{"cipher_type": 7, "password": "123abc"}' | ./xray.exe convert tmsg -t "xray.proxy.shadowsocks.Account" stdin:

var cmdTypedMessage = &base.Command{
	CustomFlags: true,
	UsageLine:   "{{.Exec}} convert tmsg [-t \"protobuf type name\"] [stdin:] [json file]",
	Short:       "Convert json to TypedMessage",
	Long: `
Convert ONE json to TypedMessage.

Arguments:

	-t, -type
		Xray-core type name.

Examples:

    {{.Exec}} convert tmsg -t "xray.proxy.shadowsocks.Account" user.json
	`,
	Run: executeJsonToTypedMessage,
}

func executeJsonToTypedMessage(cmd *base.Command, args []string) {

	var typename string
	cmd.Flag.StringVar(&typename, "t", "", "")
	cmd.Flag.StringVar(&typename, "type", "", "")
	cmd.Flag.Parse(args)

	if typename == "" {
		base.Fatalf("must provide a type name")
	}
	if cmd.Flag.NArg() < 1 {
		base.Fatalf("empty input list")
	}

	tmsg := cserial.TypedMessage{
		Type:  typename,
		Value: []byte{},
	}

	inst, err := tmsg.GetInstance()
	if err != nil {
		base.Fatalf("create instance failed\n%s", err.Error())
	}

	reader, err := confloader.LoadConfig(cmd.Flag.Arg(0))
	if err != nil {
		base.Fatalf("load input failed\n%s", err.Error())
	}

	b, err := io.ReadAll(reader)
	if err != nil {
		base.Fatalf("read input failed\n%s", err.Error())
	}

	if err = json.Unmarshal(b, &inst); err != nil {
		base.Fatalf(err.Error())
	}

	j := protojson.MarshalOptions{Indent: "  "}.Format(inst)
	fmt.Println(j)
}
