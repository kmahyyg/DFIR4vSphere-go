package vsphere_api

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/list"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

// objKind list, from govc/object/find.go in govmomi
//	{"a", "VirtualApp"},
//	{"c", "ClusterComputeResource"},
//	{"d", "Datacenter"},
//	{"f", "Folder"},
//	{"g", "DistributedVirtualPortgroup"},
//	{"h", "HostSystem"},
//	{"m", "VirtualMachine"},
//	{"n", "Network"},
//	{"o", "OpaqueNetwork"},
//	{"p", "ResourcePool"},
//	{"r", "ComputeResource"},
//	{"s", "Datastore"},
//	{"w", "DistributedVirtualSwitch"},

func (vsc *vSphereClient) ListEsxiHost() error {
	tmpctx := context.Background()
	if !vsc.IsLoggedIn() || !vsc.postInitDone {
		return ErrSessionInvalid
	}
	viewMgr := view.NewManager(vsc.vmwSoapClient)
	objKind := []string{"HostSystem"}
	objRootDir := vsc.vmwSoapClient.ServiceContent.RootFolder
	ctnrView, err := viewMgr.CreateContainerView(tmpctx, objRootDir, objKind, true)
	if err != nil {
		return err
	}
	defer func() {
		_ = ctnrView.Destroy(tmpctx)
	}()
	filterSpec := property.Filter{"name": "*"}
	esxHostLstV, err := ctnrView.Find(tmpctx, objKind, filterSpec)
	if err != nil {
		return err
	}
	if len(esxHostLstV) == 0 {
		log.Warn("Retrieved data length of DataCenter List is zero.")
	}
	log.Debugf("Retrieved ESXi Host: %v", esxHostLstV)
	// we should check object existence before finder context destroyed, then
	// save as list.Element only.
	esxHostLst := make([]list.Element, 0)
	tmpFinder := find.NewFinder(vsc.vmwSoapClient, true)
	for _, oMRO := range esxHostLstV {
		elem, err := tmpFinder.Element(tmpctx, oMRO)
		if err != nil {
			if soap.IsSoapFault(err) {
				_, ok := soap.ToSoapFault(err).VimFault().(types.ManagedObjectNotFound)
				if ok {
					// object was deleted after v.Find()
					continue
				}
			}
			log.Errorln("inlinefunc, convert-to-elem, err: ", err)
			continue
		}
		esxHostLst = append(esxHostLst, *elem)
	}
	//esxHostLst, type=([]list.Element)
	vsc.SetCtxData("esxiHostList", esxHostLst)
	return nil
}

func (vsc *vSphereClient) ListDataCenter() error {
	tmpctx := context.Background()
	if !vsc.IsLoggedIn() || !vsc.postInitDone {
		return ErrSessionInvalid
	}
	viewMgr := view.NewManager(vsc.vmwSoapClient)
	objKind := []string{"Datacenter"}
	objRootDir := vsc.vmwSoapClient.ServiceContent.RootFolder
	ctnrView, err := viewMgr.CreateContainerView(tmpctx, objRootDir, objKind, true)
	if err != nil {
		return err
	}
	defer func() {
		_ = ctnrView.Destroy(tmpctx)
	}()
	filterSpec := property.Filter{"name": "*"}
	dcLstV, err := ctnrView.Find(tmpctx, objKind, filterSpec)
	if err != nil {
		return err
	}
	if len(dcLstV) == 0 {
		log.Warn("Retrieved data length of DataCenter List is zero.")
	}
	log.Debugf("Retrieved DataCenter: %v", dcLstV)
	// we should check object existence after finder context destroyed, then
	// save as list.Element only.
	// this will help for preventing error in other methods.
	dcLst := make([]list.Element, 0)
	tmpFinder := find.NewFinder(vsc.vmwSoapClient, true)
	for _, oMRO := range dcLstV {
		elem, err := tmpFinder.Element(tmpctx, oMRO)
		if err != nil {
			if soap.IsSoapFault(err) {
				_, ok := soap.ToSoapFault(err).VimFault().(types.ManagedObjectNotFound)
				if ok {
					// object was deleted after v.Find()
					continue
				}
			}
			log.Errorln("inlinefunc, convert-to-elem, err: ", err)
			continue
		}
		dcLst = append(dcLst, *elem)
	}
	// dcLst, type=([]list.Element)
	// list.Element can be converted to specific object type easily. However, reverse operation is not supported.
	vsc.SetCtxData("dcList", dcLst)
	return nil
}
