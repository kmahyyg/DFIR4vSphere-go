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
	VIOFilters []*ESXIOFilter `json:"vio_filters"`
	// esxi host certificate
	CertificateInfo *ESXHostCert `json:"tls_certificates"`
	// esxi interface ips

	// esxi netif routes
	NetRoutes []string `json:"net_routes,omitempty"` // this should follow Linux route command format
	// esxi v-switch list

}

type ESXAuthenticationInfo struct {
	Type    string                                    `json:"type"`
	Enabled bool                                      `json:"enabled"`
	ADInfo  *esxHostAcitveDirectoryAuthenticationInfo `json:"active_directory_info,omitempty"`
}

type ESXHostVSW struct {
	Name      string        `json:"name,omitempty"`
	MTU       uint16        `json:"mtu,omitempty"`
	BridgedTo []string      `json:"bridged_to,omitempty"`
	PortGroup []*ESXPortGUP `json:"port_group,omitempty"`
}

type ESXPortGUP struct {
	Name  string       `json:"name,omitempty"`
	Ports []*ESXNetNIC `json:"ports,omitempty"`
}

type ESXNetNIC struct {
	Name       string `json:"name,omitempty"`
	MacAddr    string `json:"mac_addr,omitempty"`
	Type       string `json:"type,omitempty"`
	IsVirtual  bool   `json:"is_virtual,omitempty"`
	IpAddr     string `json:"ip_addr,omitempty"`
	SubnetMask string `json:"subnet_mask,omitempty"`
	UsingDHCP  bool   `json:"using_dhcp,omitempty"`
}

type ESXRouteTable struct {
}

type esxHostAcitveDirectoryAuthenticationInfo struct {
	JoinedDomain           string   `json:"joined_domain"`
	TrustedDomain          []string `json:"trusted_domain"`
	DomainMembershipStatus string   `json:"domain_membership_status"`
	SmartCardAuthEnabled   bool     `json:"smart_card_auth_enabled"`
}

type ESXIOFilter struct {
	Id          string `json:"id"`
	Vendor      string `json:"vendor"`
	Version     string `json:"version"`
	ReleaseDate string `json:"release_date"`
	Summary     string `json:"summary"`
	Type        string `json:"type"`
}

type ESXHostCert struct {
	ThumbprintSHA1   string `json:"thumbprint_sha_1,omitempty"`
	ThumbprintSHA256 string `json:"thumbprint_sha_256,omitempty"`
	SubjectName      string `json:"subject_name,omitempty"`
	IssuerName       string `json:"issuer_name,omitempty"`
	NotAfter         string `json:"not_after,omitempty"`
	NotBefore        string `json:"not_before,omitempty"`
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
		esxhbi.VIOFilters = convertIoFilterInfo2External(hsys.Config.IoFilterInfo)
		// init for zero value
		esxhbi.CertificateInfo = nil
		// obtain cert mgr inst
		certmgr, err := esxhbi.moref.ConfigManager().CertificateManager(tmpCtx)
		if err != nil {
			log.Errorln("Certificate Manager of current host cannot be produced by factory, err: ", err)
		} else {
			// parse
			cinfo, err := certmgr.CertificateInfo(tmpCtx)
			if err != nil {
				log.Errorln("Err when trying to find cert using mgr: ", err)
			} else {
				esxhbi.CertificateInfo = convertHostCertInfo2External(cinfo)
			}
		}
		// vswitch
		// network ifs
	} else {
		return ErrPropertiesIsNil
	}
	return nil
}

func convertHostCertInfo2External(ci *object.HostCertificateInfo) *ESXHostCert {
	res := &ESXHostCert{
		ThumbprintSHA1:   ci.ThumbprintSHA1, // if managed by VCSA, this will be replaced by VCSA cert
		ThumbprintSHA256: ci.ThumbprintSHA256,
		SubjectName:      ci.SubjectName().String(),
		IssuerName:       ci.IssuerName().String(),
		NotAfter:         ci.Certificate.NotAfter.String(),
		NotBefore:        ci.Certificate.NotBefore.String(),
	}
	return res
}

func convertIoFilterInfo2External(iofi []types.HostIoFilterInfo) []*ESXIOFilter {
	res := make([]*ESXIOFilter, len(iofi))
	if len(iofi) == 0 {
		return nil
	}
	for i := range iofi {
		t1 := &ESXIOFilter{
			Id:          iofi[i].Id,
			Vendor:      iofi[i].Vendor,
			Version:     iofi[i].Version,
			ReleaseDate: iofi[i].ReleaseDate,
			Summary:     iofi[i].Summary,
			Type:        iofi[i].Type,
		}
		res[i] = t1
	}
	return res
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
