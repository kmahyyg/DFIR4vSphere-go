package vsphere_api

import (
	"context"
	log "github.com/sirupsen/logrus"
)

func (vsc *vSphereClient) ListEsxiHost() error {
	tmpctx := context.TODO()
	if !vsc.IsLoggedIn() || !vsc.postInitDone {
		return ErrSessionInvalid
	}
	elemRes, err := vsc.curFinder.ManagedObjectListChildren(tmpctx, "/", "HostSystem")
	if err != nil {
		return err
	}
	if len(elemRes) == 0 {
		log.Warn("Retrieved data length of ESXi Host List is zero.")
	}
	log.Debugf("Retrieved ESXi Host: %v", elemRes)
	// esxiHost, type=([]list.Element)
	vsc.SetCtxData("esxiHostList", elemRes)
	return nil
}

func (vsc *vSphereClient) ListDataCenter() error {
	tmpctx := context.TODO()
	if !vsc.IsLoggedIn() || !vsc.postInitDone {
		return ErrSessionInvalid
	}
	dcLst, err := vsc.curFinder.ManagedObjectListChildren(tmpctx, "/", "DataCenter")
	if err != nil {
		return err
	}
	if len(dcLst) == 0 {
		log.Warn("Retrieved data length of DataCenter List is zero.")
	}
	log.Debugf("Retrieved DataCenter: %v", dcLst)
	// dcList, type=([]list.Element)
	// list.Element can be converted to specific object type easily. However, reverse operation is not supported.
	vsc.SetCtxData("dcList", dcLst)
	return nil
}
