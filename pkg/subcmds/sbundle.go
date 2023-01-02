package subcmds

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/kmahyyg/DFIR4vSphere-go/pkg/vsphere_api"
	log "github.com/sirupsen/logrus"
)

func RetrieveSupportBundle() {
	err := vsphere_api.GlobalClient.ListEsxiHost()
	if err != nil {
		log.Errorln("retrieve esxi host list failed: ", err)
	}
	esxHostSelections, err := func() ([]string, error) {
		//TODO
		return nil, nil
	}()
	if err != nil {
		log.Errorln("build esxi host option list failed: ", err)
		return
	}
	ansEsxHosts := make([]int, 0)
	qsEsxHosts := &survey.MultiSelect{
		Message:  "Select ESXi Host you would like to request a support bundle:",
		Options:  esxHostSelections,
		PageSize: 10,
	}
	err = survey.AskOne(qsEsxHosts, ansEsxHosts, survey.WithValidator(survey.Required))
	if err != nil {
		log.Errorln("user answer err: ", err)
	}
	log.Debugln("Retrieved ESXi Host for Selection: ", esxHostSelections)
	log.Debugln("User Selected: ", ansEsxHosts)
}
