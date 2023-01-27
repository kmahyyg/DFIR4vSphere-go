package vsphere_api

import (
	"context"
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/govc/host/esxcli"
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
	inited        bool               `json:"-"`
	esxcliExec    *esxcli.Executor   `json:"-"`
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
	NetIfs []*ESXNetNIC `json:"net_ifs"`
	// esxi v-switch list
	NetVSwitches   []*ESXHostVSW  `json:"net_v_switches"`
	NetVPortGroups []*ESXHostPGrp `json:"net_v_port_groups"`
}

type ESXAuthenticationInfo struct {
	Type    string                                    `json:"type"`
	Enabled bool                                      `json:"enabled"`
	ADInfo  *esxHostAcitveDirectoryAuthenticationInfo `json:"active_directory_info,omitempty"`
}

type ESXHostVSW struct {
	Name      string   `json:"name,omitempty"`
	MTU       int32    `json:"mtu,omitempty"`
	BridgedTo []string `json:"bridged_to,omitempty"`
	PortGroup []string `json:"port_group,omitempty"`
}

type ESXHostPGrp struct {
	Key               string             `json:"key,omitempty"`
	VSwitch           string             `json:"v_switch,omitempty"`
	AllowPromisc      bool               `json:"allow_promisc,omitempty"`
	AllowMacChange    bool               `json:"allow_mac_change,omitempty"`
	AllowTransitForge bool               `json:"allow_transit_forge,omitempty"`
	Ports             []*ESXHostPGrpPort `json:"ports,omitempty"`
}

type ESXHostPGrpPort struct {
	Key         string   `json:"key"`
	MacAddr     []string `json:"mac_addr"`
	Type        string   `json:"type"`
	ActiveNICs  []string `json:"active_nics"`
	StandbyNICs []string `json:"standby_nics"`
}

type ESXNetNIC struct {
	Name       string `json:"name,omitempty"`
	MacAddr    string `json:"mac_addr,omitempty"`
	Type       string `json:"type,omitempty"`
	IsVirtual  bool   `json:"is_virtual,omitempty"`
	IpAddr     string `json:"ip_addr,omitempty"`
	SubnetMask string `json:"subnet_mask,omitempty"`
	UsingDHCP  bool   `json:"using_dhcp,omitempty"`
	GatewayIP  string `json:"gateway_ip,omitempty"`
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
		if err != nil {
			log.Errorln("ESXiHostBasicInfoF1, err:", err)
		}
		err = esxBInfo.ExposeESXCliv2()
		if err != nil {
			return err
		}
		err = esxBInfo.GetInfoFunc2()
		if err != nil {
			log.Errorln("ESXiHostBasicInfoF2, err:", err)
		}
	}
	return nil
}

func (esxhbi *ESXHostBasicInfo) ExposeESXCliv2() (err error) {
	if esxhbi.inited {
		esxhbi.esxcliExec, err = esxcli.NewExecutor(GlobalClient.GetSOAPClient(), esxhbi.moref)
		if err != nil {
			log.Errorln("initiate esxcli executor failed: ", err)
			return err
		}
		return nil
	}
	return ErrPrerequisitesNotSatisfied
}

func (esxhbi *ESXHostBasicInfo) Init(h *object.HostSystem, invtpath string) error {
	esxhbi.moref = h
	esxhbi.InventoryPath = invtpath
	esxhbi.inited = true
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
		esxhbi.NetVSwitches = convertHostNetVSW2External(hsys.Config.Network.Vswitch)
		esxhbi.NetVPortGroups = convertPortGrps2External(hsys.Config.Network.Portgroup)
		// network ifs
		esxhbi.NetIfs = showNICs(hsys.Config.Network)
	} else {
		return ErrPropertiesIsNil
	}
	return nil
}

func showNICs(n *types.HostNetworkInfo) []*ESXNetNIC {
	res := make([]*ESXNetNIC, 0)
	for _, v := range n.Pnic {
		i1 := &ESXNetNIC{
			Name:       v.Key,
			MacAddr:    v.Mac,
			Type:       v.Driver,
			IsVirtual:  false,
			IpAddr:     "-",
			SubnetMask: "-",
			UsingDHCP:  false,
			GatewayIP:  "-",
		}
		res = append(res, i1)
	}
	for _, v := range n.Vnic {
		i2 := &ESXNetNIC{
			Name:       v.Key,
			MacAddr:    v.Spec.Mac,
			Type:       v.Device,
			IsVirtual:  true,
			IpAddr:     v.Spec.Ip.IpAddress,
			SubnetMask: v.Spec.Ip.SubnetMask,
			UsingDHCP:  v.Spec.Ip.Dhcp,
			GatewayIP: func() string {
				if v.Spec.IpRouteSpec != nil {
					if v.Spec.IpRouteSpec.IpRouteConfig != nil {
						return v.Spec.IpRouteSpec.IpRouteConfig.(*types.HostIpRouteConfig).DefaultGateway
					}
				}
				return "-"
			}(),
		}
		res = append(res, i2)
	}
	return res
}

func convertPortGrps2External(g []types.HostPortGroup) []*ESXHostPGrp {
	if len(g) == 0 {
		return nil
	}
	res := make([]*ESXHostPGrp, len(g))
	for i := range g {
		elem1 := &ESXHostPGrp{
			Key:               g[i].Key,
			VSwitch:           g[i].Vswitch,
			AllowPromisc:      *g[i].ComputedPolicy.Security.AllowPromiscuous,
			AllowMacChange:    *g[i].ComputedPolicy.Security.MacChanges,
			AllowTransitForge: *g[i].ComputedPolicy.Security.ForgedTransmits,
			Ports: func() []*ESXHostPGrpPort {
				if g[i].Port != nil {
					res2 := make([]*ESXHostPGrpPort, len(g[i].Port))
					for i2 := range g[i].Port {
						elem2 := &ESXHostPGrpPort{
							Key:     g[i].Port[i2].Key,
							MacAddr: g[i].Port[i2].Mac,
							Type:    g[i].Port[i2].Type,
							ActiveNICs: func() []string {
								if g[i].ComputedPolicy.NicTeaming != nil {
									if g[i].ComputedPolicy.NicTeaming.NicOrder != nil {
										return g[i].ComputedPolicy.NicTeaming.NicOrder.ActiveNic
									}
								}
								return nil
							}(),
							StandbyNICs: func() []string {
								if g[i].ComputedPolicy.NicTeaming != nil {
									if g[i].ComputedPolicy.NicTeaming.NicOrder != nil {
										return g[i].ComputedPolicy.NicTeaming.NicOrder.StandbyNic
									}
								}
								return nil
							}(),
						}
						res2[i] = elem2
					}
					return res2
				}
				return nil
			}(),
		}
		res[i] = elem1
	}
	return res
}

func convertHostNetVSW2External(vsws []types.HostVirtualSwitch) []*ESXHostVSW {
	if len(vsws) == 0 {
		return nil
	}
	res := make([]*ESXHostVSW, len(vsws))
	for i := range vsws {
		// loop through port group
		res[i] = &ESXHostVSW{
			Name:      vsws[i].Key,
			MTU:       vsws[i].Mtu,
			BridgedTo: vsws[i].Pnic,      // []string, to phy-nic
			PortGroup: vsws[i].Portgroup, // []string
		}
	}
	return res
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
