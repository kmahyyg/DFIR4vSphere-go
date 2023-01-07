package subcmds

import (
	"github.com/kmahyyg/DFIR4vSphere-go/pkg/vsphere_api"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/list"
	"github.com/vmware/govmomi/object"
)

func RetrieveBasicInformation() {
	vcbi := &vsphere_api.VCBasicInfo{}
	vcbi.ESXHostObjs = make([]*object.HostSystem, 0)
	// list esxi host
	err := vsphere_api.GlobalClient.ListEsxiHost()
	if err != nil {
		log.Errorln("list esxi host - basic info, err: ", err)
	} else {
		Hsysts, err := vsphere_api.GlobalClient.GetCtxData("esxiHostList")
		if err != nil {
			log.Errorln(err)
		}
		vcbi.ESXHostList = make([]string, len(Hsysts.([]list.Element)))
		for i := range Hsysts.([]list.Element) {
			hss := object.NewHostSystem(vsphere_api.GlobalClient.GetSOAPClient(), Hsysts.([]list.Element)[i].Object.Reference())
			vcbi.ESXHostList[i] = hss.InventoryPath
			vcbi.ESXHostObjs = append(vcbi.ESXHostObjs, hss)
		}
	}
	// processing if only vcenter
	if vsphere_api.GlobalClient.IsVCenter() {
		// retrieve permissions list with role
		err = vsphere_api.GlobalClient.ListPermissions(vcbi)
		if err != nil {
			log.Errorln("retrieve permissions list out, err: ", err)
		}
		//TODO: get local and sso user
		//
		// get max age
		vcbi.EventMaxAge, err = vsphere_api.GlobalClient.GetEventMaxAge()
		if err != nil {
			log.Errorln("getevent-max-age-out, err:", err)
		}
	}
	// globally retrieve data from esxi host

}
