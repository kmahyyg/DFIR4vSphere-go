package subcmds

import (
	"encoding/json"
	"github.com/kmahyyg/DFIR4vSphere-go/pkg/vsphere_api"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/list"
	"github.com/vmware/govmomi/object"
	"os"
	"strconv"
	"time"
)

func RetrieveBasicInformation() {
	vcbi := &vsphere_api.VCBasicInfo{
		IsVCenter: vsphere_api.GlobalClient.IsVCenter(),
	}
	vcbi.ESXHostObjs = make([]*object.HostSystem, 0)
	// list esxi host
	err := vsphere_api.GlobalClient.ListEsxiHost()
	if err != nil {
		log.Errorln("list esxi host - basic info, err: ", err)
		return
	} else {
		log.Infoln("esxi host list finished.")
		Hsysts, err := vsphere_api.GlobalClient.GetCtxData("esxiHostList")
		if err != nil {
			log.Errorln(err)
		}
		vcbi.ESXHostList = make([]string, len(Hsysts.([]list.Element)))
		for i := range Hsysts.([]list.Element) {
			hss := object.NewHostSystem(vsphere_api.GlobalClient.GetSOAPClient(), Hsysts.([]list.Element)[i].Object.Reference())
			vcbi.ESXHostList[i] = Hsysts.([]list.Element)[i].Path
			vcbi.ESXHostObjs = append(vcbi.ESXHostObjs, hss)
		}
	}
	// processing if only vcenter
	if vsphere_api.GlobalClient.IsVCenter() {
		log.Infoln("vcenter determined. execute vcsa-specific method.")
		// retrieve permissions list with role
		err = vsphere_api.GlobalClient.ListPermissions(vcbi)
		if err != nil {
			log.Errorln("retrieve permissions list out, err: ", err)
		}
		log.Infoln("list permission finished.")
		// ---- must use vcenter specific token authentication ----
		// get local and sso user
		err = vsphere_api.GlobalClient.ListAllUsers(vcbi)
		if err != nil {
			log.Errorln("list all users, err: ", err)
		}
		log.Infoln("list all users finished.")
		// ---- general procedures ----
		// get max age
		vcbi.EventMaxAge, err = vsphere_api.GlobalClient.GetEventMaxAge()
		if err != nil {
			log.Errorln("getevent-max-age-out, err:", err)
		}
		log.Infoln("get vcenter max event age finished.")
	}
	// if: standalone host, only singleHost should be used, do not use esxi host from List method.
	// else: for each esx host, execute other methods.
	err = vsphere_api.GlobalClient.RetrieveESXiHostBasicInfo(vcbi)
	if err != nil {
		log.Errorln("retr esxi info fail, err:", err)
		return
	}
	log.Infoln("retr esxi info finished.")
	// marshal vcbi and save
	vcbiBytes, err := json.MarshalIndent(vcbi, "", "    ")
	if err != nil {
		log.Errorln("json marshal vcbi, err: ", err)
		return
	}
	vcbiOutFd, err := os.Create("output/VCenter_BasicInfo_" + strconv.FormatInt(time.Now().Unix(), 10) + ".json")
	defer vcbiOutFd.Close()
	defer vcbiOutFd.Sync()
	if err != nil {
		log.Errorln("create vcbi marshal output file, err:", err)
		return
	}
	_, err = vcbiOutFd.Write(vcbiBytes)
	if err != nil {
		log.Errorln("write vcbi json to file failed, err: ", err)
		return
	}
	log.Infoln("vcbi info stored in json. operation finished.")
	return
}
