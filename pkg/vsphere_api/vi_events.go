package vsphere_api

import (
	"context"
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/event"
	"github.com/vmware/govmomi/vim25/types"
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
	if !vsc.postInitDone || !vsc.curSessLoggedIn || !vsc.IsVCenter() {
		return ErrSessionInvalid
	}
	vsc.evntMgr = event.NewManager(vsc.vmwSoapClient)
	return nil
}

func (vsc *vSphereClient) GetEventsFromMgr(lightMode bool, dcList []types.ManagedObjectReference) error {
	// init
	resFinalLst := make([]types.BaseEvent, 0)
	if vsc.evntMgr == nil {
		return ErrPrerequisitesNotSatisfied
	}
	tmpCtx := context.Background()
	// get max age
	err := vsc.GetEventMaxAge()
	if err != nil {
		return err
	}
	// build filter
	qFilter := &types.EventFilterSpec{
		Entity: &types.EventFilterSpecByEntity{
			Entity:    vsc.vmwSoapClient.ServiceContent.RootFolder,
			Recursion: types.EventFilterSpecRecursionOptionAll,
		},
	}
	procFunc := func(ref types.ManagedObjectReference, events []types.BaseEvent) error {
		if lightMode {
			qFilter.EventTypeId = lightVIEventTypesId
		}
		qEvnt, qErr := vsc.evntMgr.QueryEvents(tmpCtx, *qFilter)
		if qErr != nil {
			return qErr
		}
		for i := range qEvnt {
			resFinalLst = append(resFinalLst, qEvnt[i])
		}
		return nil
	}
	var finalObjRefLstBase []types.ManagedObjectReference
	if len(dcList) == 0 {
		finalObjRefLstBase = append(finalObjRefLstBase, vsc.vmwSoapClient.ServiceContent.RootFolder)
	} else {
		finalObjRefLstBase = dcList
	}
	if lightMode {
		err := vsc.evntMgr.Events(tmpCtx, finalObjRefLstBase, 10, false, false, procFunc, lightVIEventTypesId...)
		if err != nil {
			return err
		}
	} else {
		err := vsc.evntMgr.Events(tmpCtx, finalObjRefLstBase, 10, false, false, procFunc)
		if err != nil {
			return err
		}
	}
	//
}

func postProcessReceivedEventMsgAndSave(recvEvnts []types.BaseEvent, outputPath string) error {
	event.Sort(recvEvnts)
	log.Infoln("Note: Current")
}

func (vsc *vSphereClient) GetEventMaxAge() error {

}
