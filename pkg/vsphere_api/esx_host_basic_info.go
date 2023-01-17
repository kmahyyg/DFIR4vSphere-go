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
	moref         *object.HostSystem `json:"-"`
	InventoryPath string             `json:"inventory_path"`
	// esxi service
	Services []*ESXHostService `json:"services"`
	// esxi authentication info
	AuthInfo []*ESXAuthenticationInfo `json:"auth_info"`
	// esxi product info
	ProductAbout string `json:"product"`
	// esxi dns config
	DNSIPAddrs []string `json:"dns_ip_addrs"`
	// esxi v-iofilters
	// esxi vswitch list

}

type ESXAuthenticationInfo struct {
	Type    string                                    `json:"type"`
	Enabled bool                                      `json:"enabled"`
	ADInfo  *esxHostAcitveDirectoryAuthenticationInfo `json:"active_directory_info,omitempty"`
}

type esxHostAcitveDirectoryAuthenticationInfo struct {
	JoinedDomain           string   `json:"joined_domain"`
	TrustedDomain          []string `json:"trusted_domain"`
	DomainMembershipStatus string   `json:"domain_membership_status"`
	SmartCardAuthEnabled   bool     `json:"smart_card_auth_enabled"`
}

type ESXHostService struct {
	Key            string   `json:"key"`
	Label          string   `json:"label"`
	Required       bool     `json:"required"`
	Running        bool     `json:"running"`
	Policy         string   `json:"policy,omitempty"`
	SourcePkgName  string   `json:"source_pkg_name,omitempty"`
	FWRuleSetNames []string `json:"fw_ruleset_names,omitempty"`
	Uninstallable  bool     `json:"uninstallable"`
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
	esxhbi.InventoryPath = invtpath
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
		esxhbi.ProductAbout = productInfoStringer(hsys.Config.Product)
		esxhbi.Services = convertServiceInfo2External(hsys.Config.Service)
		esxhbi.AuthInfo = convertAuthStoreInfo2External(hsys.Config.AuthenticationManagerInfo)
		esxhbi.DNSIPAddrs = hsys.Config.Network.DnsConfig.GetHostDnsConfig().Address
	} else {
		return ErrPropertiesIsNil
	}
	return nil
}

func convertAuthStoreInfo2External(aInfo *types.HostAuthenticationManagerInfo) []*ESXAuthenticationInfo {
	extAuthI := make([]*ESXAuthenticationInfo, len(aInfo.AuthConfig))
	for i, v := range aInfo.AuthConfig {
		if v != nil {
			switch data := v.(type) {
			case *types.HostLocalAuthenticationInfo:
				authMetd := &ESXAuthenticationInfo{
					Type:    "local",
					Enabled: data.Enabled,
					ADInfo:  nil,
				}
				extAuthI[i] = authMetd
			case *types.HostActiveDirectoryInfo:
				authMetd := &ESXAuthenticationInfo{
					Type:    "active_directory",
					Enabled: data.Enabled,
					ADInfo: &esxHostAcitveDirectoryAuthenticationInfo{
						JoinedDomain:           data.JoinedDomain,
						TrustedDomain:          data.TrustedDomain,
						DomainMembershipStatus: data.DomainMembershipStatus,
						SmartCardAuthEnabled:   *data.SmartCardAuthenticationEnabled,
					},
				}
				extAuthI[i] = authMetd
			default:
				log.Errorf("unknown type when converting hostAuthenticationInfo, data: %+v", data)
			}
		}
	}
	return extAuthI
}

func productInfoStringer(p types.AboutInfo) string {
	return p.FullName + " running " + p.LicenseProductName + " on " + p.OsType
}

func convertServiceInfo2External(srvs *types.HostServiceInfo) []*ESXHostService {
	extSrv := make([]*ESXHostService, len(srvs.Service))
	if len(srvs.Service) == 0 {
		return nil
	}
	for i := range srvs.Service {
		t := srvs.Service[i]
		ext_s_S := &ESXHostService{
			Key:            t.Key,
			Label:          t.Label,
			Required:       t.Required,
			Running:        t.Running,
			Policy:         t.Policy,
			SourcePkgName:  t.SourcePackage.SourcePackageName,
			FWRuleSetNames: t.Ruleset,
			Uninstallable:  t.Uninstallable,
		}
		extSrv[i] = ext_s_S
	}
	return extSrv
}
