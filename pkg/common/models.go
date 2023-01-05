package common

import (
	_ "embed"
	"fmt"
	"os"
)

var (
	UserAnswer = &UserInput{}
	LogFileFD  *os.File
)

//go:embed gitversion.txt
var VersionStr string

type UserInput struct {
	HttpProxyHost string `survey:"http_proxy"`
	HostAddr      string `survey:"vsphere_hostport"`
	Username      string `survey:"vsphere_user"`
	Password      string `survey:"vsphere_pass"`
	SkipTLSVerify bool   `survey:"skip_tls_verify"`
}

func (ui *UserInput) String() string {
	return fmt.Sprintf("Proxy: %s , SkipTLS: %v , Host: %s , Username: %s .", ui.HttpProxyHost,
		ui.SkipTLSVerify, ui.HostAddr, ui.Username)
}
