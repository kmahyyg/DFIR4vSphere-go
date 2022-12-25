package common

import (
	"fmt"
	"os"
)

var (
	UserAnswer = &UserInput{}
	VersionStr = ""
	LogFileFD  *os.File
)

type UserInput struct {
	HttpProxyHost        string `survey:"http_proxy"`
	HostAddr             string `survey:"vsphere_hostport"`
	Username             string `survey:"vsphere_user"`
	Password             string `survey:"vsphere_pass"`
	IsESXiStandaloneHost bool   `survey:"standalone_esxi"`
	SkipTLSVerify        bool   `survey:"skip_tls_verify"`
}

func (ui *UserInput) String() string {
	return fmt.Sprintf("Proxy: %s , SkipTLS: %v , Host: %s , IsESXi: %v, Username: %s .", ui.HttpProxyHost, ui.SkipTLSVerify, ui.HostAddr,
		ui.IsESXiStandaloneHost, ui.Username)
}
