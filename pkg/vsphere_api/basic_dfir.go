package vsphere_api

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
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
	esxHostLst, err := ctnrView.Find(tmpctx, objKind, filterSpec)
	if err != nil {
		return err
	}
	if len(esxHostLst) == 0 {
		log.Warn("Retrieved data length of DataCenter List is zero.")
	}
	log.Debugf("Retrieved ESXi Host: %v", esxHostLst)
	//TODO: we should check object existence after finder context destroyed, then
	// save as list.Element only.
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
	dcLst, err := ctnrView.Find(tmpctx, objKind, filterSpec)
	if err != nil {
		return err
	}
	if len(dcLst) == 0 {
		log.Warn("Retrieved data length of DataCenter List is zero.")
	}
	log.Debugf("Retrieved DataCenter: %v", dcLst)
	//TODO: we should check object existence after finder context destroyed, then
	// save as list.Element only.
	// this will help for preventing error in other methods.
	//  github.com/vmware/govmomi@v0.30.0/govc/object/find.go:389
	// dcLst, type=([]list.Element)
	// list.Element can be converted to specific object type easily. However, reverse operation is not supported.
	vsc.SetCtxData("dcList", dcLst)
	return nil
}
