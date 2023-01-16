package vsphere_api

import (
	"context"
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

var (
	ErrNoObjectInMoList  = errors.New("no managed object ref in array")
	ErrFuzzyResultInList = errors.New("multiple result returned with single-result filter expected")
	ErrPropertiesIsNil   = errors.New("properties retrieving finished with nil result")
)

type ESXHostBasicInfo struct {
	moref    *object.HostSystem
	invtPath string
}

func (vsc *vSphereClient) RetrieveESXiHostBasicInfo(vcbi *VCBasicInfo) error {
	if len(vcbi.ESXHostObjs) == 0 {
		return ErrNoObjectInMoList
	}
	for i := range vcbi.ESXHostObjs {
		esxBInfo := &ESXHostBasicInfo{}
		err := esxBInfo.Init(vcbi.ESXHostObjs[i], vcbi.ESXHostList[i])
		if err != nil {
			return err
		}
		err = esxBInfo.GetInfoFunc1()
	}
	return nil
}

func (esxhbi *ESXHostBasicInfo) Init(h *object.HostSystem, invtpath string) error {
	esxhbi.moref = h
	esxhbi.invtPath = invtpath
	return nil
}

func (esxhbi *ESXHostBasicInfo) GetInfoFunc1() (err error) {
	// config properties
	tmpCtx := context.Background()
	coll := property.DefaultCollector(GlobalClient.GetSOAPClient())
	filter := new(property.WaitFilter)
	filter.Add(esxhbi.moref.Reference(), esxhbi.moref.Reference().Type, []string{"config"})
	req := types.RetrieveProperties{
		SpecSet: []types.PropertyFilterSpec{filter.Spec},
	}
	res, err := coll.RetrieveProperties(tmpCtx, req)
	if err != nil {
		log.Errorln("err when retr props. ")
		return err
	}
	ctnt := res.Returnval
	if len(ctnt) != 1 {
		log.Errorln("should only have 1 result in array being matched, got ", len(ctnt))
		return ErrFuzzyResultInList
	}
	obj, err := mo.ObjectContentToType(ctnt[0])
	// here obj should be the same as ref type.
	if err != nil {
		log.Errorln("err when obt-content to type.")
		return err
	}
	// copy obj and corresponding property
	// note: you should only retreive specific property as filter specified above, other properties are nil!
	hsys := obj.(mo.HostSystem)
	if hsys.Config != nil {
		//TODO: copy to our native data structure, print data inside ptr
		log.Infof("Host System [%s] Config: %+s", esxhbi.invtPath, hsys.Config)
	} else {
		return ErrPropertiesIsNil
	}
	return nil
}
