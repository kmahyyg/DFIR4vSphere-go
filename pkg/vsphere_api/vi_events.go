package vsphere_api

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/event"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
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

func (vsc *vSphereClient) NewEventManager() error {
	if !vsc.curSessLoggedIn || !vsc.IsVCenter() {
		return ErrSessionInvalid
	}
	vsc.evntMgr = event.NewManager(vsc.vmwSoapClient)
	return nil
}

func (vsc *vSphereClient) GetEventsFromMgr(lightMode bool, dcList []types.ManagedObjectReference) error {
	// init
	resFinalLst := make([]*wrappedViEvent, 0)
	if vsc.evntMgr == nil || !vsc.IsVCenter() {
		return ErrPrerequisitesNotSatisfied
	}
	tmpCtx := context.Background()
	// get max age
	err := vsc.GetEventMaxAge()
	if err != nil {
		return err
	}
	log.Infoln("Getting vCenter Advanced Config: event.MaxAge finished successfully.")
	// build filter and callback function
	procFunc := func(ref types.ManagedObjectReference, events []types.BaseEvent) error {
		log.Debugf("inline-procFunc: ref: %v , events: %v", ref, events)
		// filter on base and recursively
		qFilter := &types.EventFilterSpec{
			Entity: &types.EventFilterSpecByEntity{
				Entity:    ref,
				Recursion: types.EventFilterSpecRecursionOptionAll,
			},
		}
		// light mode switch
		if lightMode {
			qFilter.EventTypeId = lightVIEventTypesId
		}
		// this function is used to query detailed information
		qEvnt, qErr := vsc.evntMgr.QueryEvents(tmpCtx, *qFilter)
		if qErr != nil {
			return qErr
		}
		log.Debugln("inline-procFunc - QueryEvents: finished with no err.")
		for i := range qEvnt {
			nEvntCate, err := vsc.evntMgr.EventCategory(tmpCtx, qEvnt[i])
			if err != nil {
				log.Errorln("retrieving specific event log level unsuccessful.")
			}
			nEvnt := qEvnt[i].GetEvent()
			// wrap into struct and
			wrapNEvnt := &wrappedViEvent{
				SubjectObj:    ref.String(),
				CreatedTime:   nEvnt.CreatedTime,
				CategoryLevel: nEvntCate,
				Message:       strings.TrimSpace(nEvnt.FullFormattedMessage),
				EventID:       nEvnt.Key,
				EventType:     reflect.TypeOf(nEvnt).Elem().Name(),
				bEvent:        qEvnt[i],
			}
			resFinalLst = append(resFinalLst, wrapNEvnt)
		}
		return nil
	}
	// if selected nothing, use root folder. Multiple events only exists on root.
	// so, it is recommended do not select specific datacenter unless you are required to do so.
	var finalObjRefLstBase []types.ManagedObjectReference
	if len(dcList) == 0 {
		finalObjRefLstBase = append(finalObjRefLstBase, vsc.vmwSoapClient.ServiceContent.RootFolder)
	} else {
		finalObjRefLstBase = dcList
	}
	//
	if lightMode {
		err := vsc.evntMgr.Events(tmpCtx, finalObjRefLstBase, 10, false, true, procFunc, lightVIEventTypesId...)
		if err != nil {
			return err
		}
	} else {
		err := vsc.evntMgr.Events(tmpCtx, finalObjRefLstBase, 10, false, true, procFunc)
		if err != nil {
			return err
		}
	}
	log.Debugln("requesting all related events successfully finished. start post-processing.")
	// do post processing like sorting, printing, saving stuffs
	wdir, _ := os.Getwd()
	wDstFilePath := filepath.Join(wdir, "vi-events-"+strconv.FormatInt(time.Now().Unix(), 10)+".csv")
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

func (vsc *vSphereClient) GetEventMaxAge() error {
	_ = vsc.NewVcsaOptionManager()
	tmpCtx := context.Background()
	opts, err := vsc.vcsaOptionMgr.Query(tmpCtx, "event.maxAge")
	if err != nil {
		return err
	}
	for i := range opts {
		sOpt := opts[i].GetOptionValue()
		log.Infof("VCSA Option: %s = %v ", sOpt.Key, sOpt.GetOptionValue().Value)
	}
	return nil
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
