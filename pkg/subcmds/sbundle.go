package subcmds

import (
	"context"
	"github.com/AlecAivazis/survey/v2"
	"github.com/kmahyyg/DFIR4vSphere-go/pkg/vsphere_api"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/list"
	"github.com/vmware/govmomi/object"
	"sync"
)

func RetrieveSupportBundle(wg *sync.WaitGroup) {
	// no need to check if vCenter or standalone ESXi Host, if standalone, then there will only be a single host
	// list and retrieve esxi host from server
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
	// build selection
	tmpESX := esxHostLst.([]list.Element)
	esxHostSelections, err := func() ([]string, error) {
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
	// ask user, query answer is index list
	ansEsxHosts := make([]int, 0)
	qsEsxHosts := &survey.MultiSelect{
		Message:  "Select ESXi Host you would like to request a support bundle:",
		Options:  esxHostSelections,
		PageSize: 10,
	}
	err = survey.AskOne(qsEsxHosts, &ansEsxHosts, survey.WithValidator(survey.Required))
	if err != nil {
		log.Errorln("user answer err: ", err)
		return
	}
	log.Debugln("Retrieved ESXi Host for Selection: ", esxHostSelections)
	log.Debugln("User Selected: ", ansEsxHosts)
	// convert from index list of list.Element to *object.hostsystem
	hsList := make([]*object.HostSystem, len(ansEsxHosts))
	for i, v := range ansEsxHosts {
		sHostElem := tmpESX[v]
		hsList[i] = object.NewHostSystem(vsphere_api.GlobalClient.GetSOAPClient(), sHostElem.Object.Reference())
	}
	// call internal function
	err = vsphere_api.GlobalClient.RequestSupportBundle(hsList, wg)
	if err != nil {
		log.Errorln("request support bundle err: ", err)
		return
	}
	log.Infoln("Request support bundle successfully finished.")
	return
}
