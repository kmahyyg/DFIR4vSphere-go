package subcmds

import (
	"context"
	"github.com/AlecAivazis/survey/v2"
	"github.com/kmahyyg/DFIR4vSphere-go/pkg/vsphere_api"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/list"
)

func RetrieveSupportBundle() {
	err := vsphere_api.GlobalClient.ListEsxiHost()
	if err != nil {
		log.Errorln("retrieve esxi host list failed: ", err)
		return
	}
	esxHostLst, err := vsphere_api.GlobalClient.GetCtxData("esxiHostList")
	if err != nil {
		log.Errorln("esxi host list not in ctx: ", err)
		return
	}
	esxHostSelections, err := func() ([]string, error) {
		tmpESX := esxHostLst.([]list.Element)
		res := make([]string, len(tmpESX))
		for i := range tmpESX {
			tmpCtx := context.Background()
			iIPath, err := find.InventoryPath(tmpCtx, vsphere_api.GlobalClient.GetSOAPClient(), tmpESX[i].Object.Reference())
			if err != nil {
				return nil, err
			}
			res[i] = iIPath
		}
		return res, nil
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
	return
}
