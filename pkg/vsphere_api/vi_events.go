package vsphere_api

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/types"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrDatetimeUnknown           = errors.New("unknown error when try to build time range")
	ErrPrerequisitesNotSatisfied = errors.New("dependencies not initialized")
)

var (
	lightVIEventTypesId = []string{"ad.event.JoinDomainEvent", "VmFailedToSuspendEvent", "VmSuspendedEvent",
		"VmSuspendingEvent", "VmDasUpdateOkEvent", "VmReconfiguredEvent", "UserUnassignedFromGroup",
		"UserAssignedToGroup", "UserPasswordChanged", "AccountCreatedEvent", "AccountRemovedEvent",
		"AccountUpdatedEvent", "UserLoginSessionEvent", "RoleAddedEvent", "RoleRemovedEvent", "RoleUpdatedEvent",
		"TemplateUpgradeEvent", "TemplateUpgradedEvent", "PermissionAddedEvent", "PermissionUpdatedEvent",
		"PermissionRemovedEvent", "LocalTSMEnabledEvent", "DatastoreFileDownloadEvent", "DatastoreFileUploadEvent",
		"DatastoreFileDeletedEvent", "VmAcquiredMksTicketEvent",
		"com.vmware.vc.guestOperations.GuestOperationAuthFailure", "com.vmware.vc.guestOperations.GuestOperation",
		"esx.audit.ssh.enabled", "esx.audit.ssh.session.failed", "esx.audit.ssh.session.closed",
		"esx.audit.ssh.session.opened", "esx.audit.account.locked", "esx.audit.account.loginfailures",
		"esx.audit.dcui.login.passwd.changed", "esx.audit.dcui.enabled", "esx.audit.dcui.disabled",
		"esx.audit.lockdownmode.exceptions.changed", "esx.audit.shell.disabled", "esx.audit.shell.enabled",
		"esx.audit.lockdownmode.disabled", "esx.audit.lockdownmode.enabled", "com.vmware.sso.LoginSuccess",
		"com.vmware.sso.LoginFailure", "com.vmware.sso.Logout", "com.vmware.sso.PrincipalManagement",
		"com.vmware.sso.RoleManagement", "com.vmware.sso.IdentitySourceManagement", "com.vmware.sso.DomainManagement",
		"com.vmware.sso.ConfigurationManagement", "com.vmware.sso.CertificateManager",
		"com.vmware.trustmanagement.VcTrusts", "com.vmware.trustmanagement.VcIdentityProviders",
		"com.vmware.cis.CreateGlobalPermission", "com.vmware.cis.CreatePermission",
		"com.vmware.cis.RemoveGlobalPermission", "com.vmware.cis.RemovePermission", "com.vmware.vc.host.Crypto.Enabled",
		"com.vmware.vc.host.Crypto.HostCryptoDisabled", "ProfileCreatedEvent", "ProfileChangedEvent",
		"ProfileRemovedEvent", "ProfileAssociatedEvent", "esx.audit.esximage.vib.install.successful",
		"esx.audit.esximage.hostacceptance.changed", "esx.audit.esximage.vib.remove.successful"}
)

type wrappedCallbackInput struct {
	Events  []types.BaseEvent
	BaseObj types.ManagedObjectReference
}

func (vsc *vSphereClient) GetEventsFromMgr(lightMode bool, dcList []types.ManagedObjectReference) error {
	// init
	resFinalLst := make([]*wrappedViEvent, 0)
	if vsc.evntMgr == nil || !vsc.IsVCenter() || !vsc.postInitDone {
		return ErrPrerequisitesNotSatisfied
	}
	// get max age
	_, err := vsc.GetEventMaxAge()
	if err != nil {
		return err
	}
	if vsc.evntMaxAge <= 0 {
		return ErrDatetimeUnknown
	}
	log.Infoln("Getting vCenter Advanced Config: event.MaxAge finished successfully.")

	// go coroutine-processing
	sPageChan := make(chan wrappedCallbackInput, 256)
	wgEventsProc := &sync.WaitGroup{}
	sCallBackFnDone := make(chan struct{}, 0)
	// build filter and callback function
	pageCallBackFn := func(srcObj types.ManagedObjectReference, cPageEvnts []types.BaseEvent) error {
		tmpCtx := context.Background()
		log.Debugf("inline-procFunc: srcObj: %v , len(cPageEvnts): %d", srcObj, len(cPageEvnts))
		for i := range cPageEvnts {
			nEvntCate, err := vsc.evntMgr.EventCategory(tmpCtx, cPageEvnts[i])
			if err != nil {
				log.Errorln("retrieving specific event log level unsuccessful. err: ", err)
				continue
			}
			nEvnt := cPageEvnts[i].GetEvent()
			// wrap into struct and
			wrapNEvnt := &wrappedViEvent{
				SubjectObj:    srcObj.String(),
				CreatedTime:   nEvnt.CreatedTime,
				CategoryLevel: nEvntCate,
				Message:       strings.TrimSpace(nEvnt.FullFormattedMessage),
				EventID:       nEvnt.Key,
				EventType:     reflect.TypeOf(nEvnt).Elem().Name(),
				bEvent:        cPageEvnts[i],
			}
			resFinalLst = append(resFinalLst, wrapNEvnt)
		}
		return nil
	}
	go func() {
		for {
			sWcbIpt, evntHasNext := <-sPageChan
			if !evntHasNext {
				break
			}
			err := pageCallBackFn(sWcbIpt.BaseObj, sWcbIpt.Events)
			if err != nil {
				log.Errorln("recv-pagecallback-proc, err: ", err)
				continue
			}
		}
		sCallBackFnDone <- struct{}{}
		close(sCallBackFnDone)
	}()

	// collector builder function
	collectorBuilderFn := func(baseRef types.ManagedObjectReference, lightMode bool) (collectorFilter types.EventFilterSpec, err error) {
		tmpCtx := context.Background()
		// specify time range must from today to maxAge days ago
		endUntil, err := methods.GetCurrentTime(tmpCtx, vsc.vmwSoapClient)
		if err != nil {
			return types.EventFilterSpec{}, err
		}
		startFrom := endUntil.AddDate(0, 0, -vsc.evntMaxAge)
		// create filter spec
		collectorFilter = types.EventFilterSpec{
			Entity: &types.EventFilterSpecByEntity{
				Entity:    baseRef,
				Recursion: types.EventFilterSpecRecursionOptionAll,
			},
			Time: &types.EventFilterSpecByTime{
				BeginTime: &startFrom,
				EndTime:   endUntil,
			},
		}
		if lightMode {
			collectorFilter.EventTypeId = lightVIEventTypesId
		}
		return collectorFilter, nil
	}
	collectorInWorkFn := func(fRefBase types.ManagedObjectReference, filterSpec types.EventFilterSpec) {
		defer wgEventsProc.Done()
		log.Debugln("root object ref retrieved from param, now requesting...")
		tmpCtx := context.Background()
		collector, err := vsc.evntMgr.CreateCollectorForEvents(tmpCtx, filterSpec)
		if err != nil {
			log.Errorln("events collector creator func called got errors, err: ", err)
			return
		}
		defer collector.Destroy(tmpCtx)
		log.Infoln("collector-in-work, collector created from spec using builder.")
		for {
			events, err := collector.ReadNextEvents(tmpCtx, 500)
			if err != nil {
				log.Errorln("readNextNEvents: ", err)
			}
			log.Infof("readNextNEvents: currently %d events read.", len(events))
			if len(events) == 0 {
				break
			}
			sPageChan <- wrappedCallbackInput{
				Events:  events,
				BaseObj: fRefBase,
			}
			log.Infoln("sent read events out for callback fn processing.")
		}
	}
	log.Debugln("procFunc successfully defined, building base ref...")

	// if selected nothing, use root folder. Multiple events only exists on root.
	if len(dcList) == 0 {
		// so, it is recommended do not select specific datacenter unless you are required to do so.
		fRefBase := vsc.vmwSoapClient.ServiceContent.RootFolder
		collectorFilter, err := collectorBuilderFn(fRefBase, lightMode)
		if err != nil {
			log.Debugln("collector builder func called got errors.")
			return err
		}
		log.Debugln("collector filter spec set up.")
		wgEventsProc.Add(1)
		collectorInWorkFn(fRefBase, collectorFilter)
	} else {
		for _, vSingleDC := range dcList {
			fRefBase := vSingleDC
			collectorSFilter, err := collectorBuilderFn(fRefBase, lightMode)
			if err != nil {
				log.Debugln("collector builder func called got errors.")
				continue
			}
			log.Debugln("collector filter spec set up.")
			wgEventsProc.Add(1)
			collectorInWorkFn(fRefBase, collectorSFilter)
		}
	}
	// the max single page size is 1000, cannot be bigger,
	// > From VMWare Document:
	// > This parameter is ignored when the Start and Finish parameters are specified and all events from the specified period are retrieved.
	log.Debugln("collector created, wait for jobs getting done.")
	wgEventsProc.Wait()
	// all processor and collector successfully exited, close receiver chan
	close(sPageChan)
	// wait for callback done
	<-sCallBackFnDone
	log.Debugln("requesting all related events successfully finished. start post-processing.")
	// do post processing like sorting, printing, saving stuffs
	wdir, _ := os.Getwd()
	wDstFilePath := filepath.Join(wdir, "output", "VIEvents_"+strconv.FormatInt(time.Now().Unix(), 10)+".csv")
	// create output file
	outputFd, err := os.Create(wDstFilePath)
	if err != nil {
		return err
	}
	defer outputFd.Close()
	defer outputFd.Sync()
	log.Debugln("VI-Events CSV file for outputting has been created.")
	// create csv writer
	// static headers
	var outputCSVWr = csv.NewWriter(outputFd)
	defer outputCSVWr.Flush()
	// write header first
	outputCSVWr.Write([]string{"Timestamp", "ID", "Level", "Event Type", "Message"})
	SortWrappedEvents(resFinalLst)
	for _, v := range resFinalLst {
		err := outputCSVWr.Write(v.CSVString())
		if err != nil {
			log.Errorln("write event to csv failed: ", err)
		}
	}
	return nil
}

func (vsc *vSphereClient) NewVcsaOptionManager() error {
	vsc.vcsaOptionMgr = object.NewOptionManager(vsc.vmwSoapClient, *vsc.vmwSoapClient.ServiceContent.Setting)
	return nil
}

func (vsc *vSphereClient) GetEventMaxAge() (int, error) {
	_ = vsc.NewVcsaOptionManager()
	tmpCtx := context.Background()
	opts, err := vsc.vcsaOptionMgr.Query(tmpCtx, "event.maxAge")
	if err != nil {
		return -1, err
	}
	for i := range opts {
		sOpt := opts[i].GetOptionValue()
		if sOpt.Key == "event.maxAge" {
			resTmp := sOpt.GetOptionValue().Value.(int32)
			vsc.evntMaxAge = int(resTmp)
		}
		log.Infof("VCSA Option: %s = %v ", sOpt.Key, sOpt.GetOptionValue().Value)
	}
	return vsc.evntMaxAge, nil
}

type wrappedViEvent struct {
	SubjectObj    string //from source object
	CreatedTime   time.Time
	CategoryLevel string
	Message       string
	EventID       int32
	EventType     string
	bEvent        types.BaseEvent
}

func (wvie *wrappedViEvent) CSVString() []string {
	// if this is a TaskEvent gather a little more information
	if tmpTaskEvent, ok := wvie.bEvent.(*types.TaskEvent); ok {
		// some tasks won't have this information, so just use the event message
		if tmpTaskEvent.Info.Entity != nil {
			wvie.Message = fmt.Sprintf("%s (target=%s - %s)", wvie.Message, tmpTaskEvent.Info.Entity.Type,
				tmpTaskEvent.Info.EntityName)
		}
	}
	// "Timestamp", "ID", "Level", "Event Type", "Message"
	return []string{strconv.FormatInt(wvie.CreatedTime.Unix(), 10),
		strconv.FormatInt(int64(wvie.EventID), 10), wvie.CategoryLevel, wvie.EventType, wvie.Message}
}

type wrappedViEventList []*wrappedViEvent

func (wvies wrappedViEventList) Len() int {
	return len(wvies)
}

func (wvies wrappedViEventList) Less(i, j int) bool {
	return wvies[i].EventID < wvies[j].EventID
}

func (wvies wrappedViEventList) Swap(i, j int) {
	wvies[i], wvies[j] = wvies[j], wvies[i]
}

func SortWrappedEvents(wvies wrappedViEventList) {
	sort.Sort(wvies)
}
