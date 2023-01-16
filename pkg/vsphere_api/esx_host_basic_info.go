package vsphere_api

import (
	"context"
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

var (
	ErrNoObjectInMoList = errors.New("no mo ref in array")
)

type ESXHostBasicInfo struct {
	moref *object.HostSystem
}

func (vsc *vSphereClient) RetrieveESXiHostBasicInfo(vcbi *VCBasicInfo) error {
	if len(vcbi.ESXHostObjs) == 0 {
		return ErrNoObjectInMoList
	}
	for _, h1 := range vcbi.ESXHostObjs {
		esxBInfo := &ESXHostBasicInfo{}
		err := esxBInfo.Init(h1)
		if err != nil {
			return err
		}
		err = esxBInfo.GetInfoFunc1()
	}
	return nil
}

func (esxhbi *ESXHostBasicInfo) Init(h *object.HostSystem) error {
	esxhbi.moref = h
	return nil
}

func (esxhbi *ESXHostBasicInfo) GetInfoFunc1() (err error) {
	// config properties
	tmpCtx := context.Background()
	var hcInfo types.HostConfigInfo
	err = esxhbi.moref.Properties(tmpCtx, esxhbi.moref.Reference(), []string{"config"}, &hcInfo)
	if err != nil {
		log.Error("err")
	}
	return nil
}
