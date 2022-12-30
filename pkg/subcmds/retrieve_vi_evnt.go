package subcmds

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/kmahyyg/DFIR4vSphere-go/pkg/vsphere_api"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/list"
	"github.com/vmware/govmomi/vim25/types"
)

type viEventsQuery struct {
	LightMode bool  `survey:"light_mode"`
	DCList    []int `survey:"selectedDC_list"`
}

func RetrieveVIEvents() {
	if !vsphere_api.GlobalClient.IsLoggedIn() {
		log.Errorln("Current session is NOT LOGGED IN. Run try_reconnect for retry.")
		return
	}
	if !vsphere_api.GlobalClient.IsVCenter() {
		log.Errorln("Current session is NOT connected to a valid vCenter. Unsupported operation.")
		return
	}
	err := vsphere_api.GlobalClient.ListDataCenter()
	if err != nil {
		log.Errorln("Cannot list datacenter from server: ", err)
	}
	log.Infoln("datacenter list successfully retrieved.")
	allDC, err := vsphere_api.GlobalClient.GetCtxData("dcList")
	if err != nil {
		log.Errorln("Cannot get cached DC List: ", err)
		return
	}
	dcSelectOptions := func() []string {
		tmpDcLst := allDC.([]list.Element)
		res := make([]string, len(tmpDcLst))
		for i := range tmpDcLst {
			res[i] = tmpDcLst[i].Path
		}
		return res
	}()

	survAns := &viEventsQuery{
		LightMode: false,
		DCList:    make([]int, 0),
	}
	survQes := []*survey.Question{
		{
			Name: "light_mode",
			Prompt: &survey.Confirm{
				Message: "Use Light Mode When Extract?",
				Default: false,
				Help:    "If true, only extract specific types of events.",
			},
			Validate: survey.Required,
		},
		{
			Name: "selectedDC_list",
			Prompt: &survey.MultiSelect{
				Message:  "Select Datacenter that you would like to extract events from: (if all, press enter, do not select anything)",
				Options:  dcSelectOptions,
				PageSize: 10,
			},
		},
	}
	err = survey.Ask(survQes, survAns)
	if err != nil {
		log.Errorln("User answer invalid: ", err)
		return
	}
	log.Debugln("VI Events Retrieve, User Query Answer: ", survAns)
	// build selected dc list
	selectedDC := make([]types.ManagedObjectReference, 0)
	// append selected data center to list, note: careful with empty selection
	if len(survAns.DCList) != 0 {
		for _, v := range survAns.DCList {
			selectedDC = append(selectedDC, (allDC.([]list.Element)[v]).Object.Reference())
		}
	}
	log.Infoln("user selected datacenter list length: ", len(selectedDC))
	// create event mgr
	err = vsphere_api.GlobalClient.NewEventManager()
	if err != nil {
		log.Errorln("createEventMgr err: ", err)
		return
	}
	log.Infoln("event manager successfully initialized.")
	// start collector working
	err = vsphere_api.GlobalClient.GetEventsFromMgr(survAns.LightMode, selectedDC)
	if err != nil {
		log.Errorln("getEvntsFromMgr err: ", err)
		return
	}
	log.Infoln("successfully finished retrieve_vi_events.")
}
