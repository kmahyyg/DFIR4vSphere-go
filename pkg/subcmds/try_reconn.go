package subcmds

import (
	"github.com/kmahyyg/DFIR4vSphere-go/pkg/vsphere_api"
	log "github.com/sirupsen/logrus"
)

func TryReconn() {
	var err error
	err = vsphere_api.GlobalClient.Logout()
	if err != nil {
		log.Fatalln("Trying to logout current session, error: " + err.Error())
	}
	log.Infoln("Log-out current session success.")
	err = vsphere_api.GlobalClient.NewClient()
	if err != nil {
		log.Fatalln("Re-create vSphere Client failed: " + err.Error())
	}
	log.Infoln("Re-create vSphere Client success.")
	err = vsphere_api.GlobalClient.LoginViaPassword()
	if err != nil {
		log.Fatalln("Re-activate new session failed: " + err.Error())
	}
	log.Infoln("Reconnect successfully finished.")
}
