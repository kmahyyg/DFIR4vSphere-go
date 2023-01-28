package vsphere_api

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	ssometd "github.com/vmware/govmomi/ssoadmin/methods"
	ssotypes "github.com/vmware/govmomi/ssoadmin/types"
	"github.com/vmware/govmomi/vim25/types"
)

type vcUser struct {
	Name           string `json:"name"` // name@domain
	Alias          string `json:"alias"`
	Disabled       bool   `json:"disabled"`
	Locked         bool   `json:"locked"`
	Description    string `json:"description"`
	EmailAddr      string `json:"email_addr"`
	IsSolutionUser bool   `json:"is_solution_user"`
	Kind           string `json:"kind"`
	NickName       string `json:"nickname"` // firstname+lastname
}

type VCBasicInfo struct {
	ESXHostList           []string               `json:"esx_host_names"`
	ESXHostObjs           []*object.HostSystem   `json:"-"`
	VCAuthoriRole         []*vcAuthorizationRole `json:"vc_authorization_roles"`
	VCAuthoriPerm         []*vcPermission        `json:"vc_authorization_permissions"`
	EventMaxAge           int                    `json:"event_max_age"`
	SSOPasswordPolicyDesc string                 `json:"sso_password_policy"`
	SSOIDPDesc            []*vcIdentityProvider  `json:"sso_idp"`
	SSOGroups             []*vcGroup             `json:"sso_groups"`
	SSOUsers              []*vcUser              `json:"sso_users"`
}

type vcIdentityProviders struct {
	IDP *ssotypes.IdentitySources
}

func (vcsidp *vcIdentityProviders) ToProviderArray(vcbi *VCBasicInfo) []*vcIdentityProvider {
	idps := make([]*vcIdentityProvider, 0)
	// sso on system domain
	for _, v := range vcsidp.IDP.System.Domains {
		vcidp := &vcIdentityProvider{
			Type:      "System Domain",
			ServerURL: "-",
			Name:      "-",
			Domain:    v.Name,
			Alias:     v.Alias,
		}
		idps = append(idps, vcidp)
	}
	// ldap server
	for _, v := range vcsidp.IDP.LDAPS {
		for _, dm := range v.Domains {
			vcidp := &vcIdentityProvider{
				Type:      "LDAP",
				ServerURL: v.Details.PrimaryURL,
				Name:      v.Name,
				Domain:    dm.Name,
				Alias:     dm.Alias,
			}
			idps = append(idps, vcidp)
		}
	}
	// active directory
	if nativeADir := vcsidp.IDP.NativeAD; nativeADir != nil {
		for _, dm := range nativeADir.Domains {
			vcidp := &vcIdentityProvider{
				Type:      "Native Active Directory",
				ServerURL: "-",
				Name:      nativeADir.Name,
				Domain:    dm.Name,
				Alias:     dm.Alias,
			}
			idps = append(idps, vcidp)
		}
	}
	// if localos
	if vcsidp.IDP.LocalOS != nil {
		for _, v := range vcsidp.IDP.LocalOS.Domains {
			vcidp := &vcIdentityProvider{
				Type:      "LocalOS",
				ServerURL: "-",
				Name:      vcsidp.IDP.LocalOS.Name,
				Domain:    v.Name,
				Alias:     v.Alias,
			}
			idps = append(idps, vcidp)
		}
	}
	vcbi.SSOIDPDesc = idps
	return nil
}

type vcIdentityProvider struct {
	Type      string `json:"idp_type"`
	ServerURL string `json:"idp_server_url,omitempty"`
	Name      string `json:"idp_name,omitempty"`
	Domain    string `json:"idp_domain,omitempty"`
	Alias     string `json:"idp_alias,omitempty"`
}

type vcPermission struct {
	Entity    string `json:"entity"`
	Principal string `json:"principal"`
	IsGroup   bool   `json:"is_group"`
	RoleId    int32  `json:"role_id"`
	Propagate bool   `json:"propagate"`
}

type vcGroup struct {
	Name    string `json:"name"`
	Alias   string `json:"alias"`
	Details string `json:"details"`
}

type vcAuthorizationRole struct {
	RoleId     int32    `json:"role_id"`
	System     bool     `json:"system_role"`
	Name       string   `json:"name"`
	Info       []string `json:"info"`
	Privileges []string `json:"privileges"`
}

func (vsc *vSphereClient) ListPermissions(vcbi *VCBasicInfo) error {
	authMgr := object.NewAuthorizationManager(vsc.GetSOAPClient())
	tmpCtx := context.Background()
	// role list
	rList, err := authMgr.RoleList(tmpCtx)
	if err != nil {
		log.Errorln("list role failed.")
		return err
	}
	log.Debugln("executed: list role method in ListPermissions")
	// role list
	vcbi.VCAuthoriRole = make([]*vcAuthorizationRole, len(rList))
	for i := range rList {
		vcbi.VCAuthoriRole[i] = fromVInternalAuthorizationRoleToOutRole(rList[i])
	}
	log.Debugln("executed: transform role object in ListPermissions")
	// permission set and assignment
	permList, err := authMgr.RetrieveAllPermissions(tmpCtx)
	if err != nil {
		log.Errorln("retr perm list failed.")
		return err
	}
	log.Debugln("executed: retrieve all perms in ListPermissions")
	if len(permList) != 0 {
		log.Debugln("permission list length is not zero.")
		vcbi.VCAuthoriPerm = make([]*vcPermission, len(permList))
		for i := range permList {
			vcbi.VCAuthoriPerm[i], err = fromVInternalPermissionSetToOutPerm(permList[i])
			if err != nil {
				log.Errorln("type conversion: permission list, err: ", err)
				continue
			}
		}
		log.Debugln("executed: transform permissions object in ListPermissions")
	} else {
		log.Errorln("retrieve perm list with ZERO length.")
	}
	return nil
}

func fromVInternalAuthorizationRoleToOutRole(r1 types.AuthorizationRole) *vcAuthorizationRole {
	res := &vcAuthorizationRole{
		RoleId:     r1.RoleId,
		System:     r1.System,
		Name:       r1.Name,
		Info:       []string{r1.Info.GetDescription().Label, r1.Info.GetDescription().Summary},
		Privileges: r1.Privilege,
	}
	return res
}

func fromVInternalPermissionSetToOutPerm(r1 types.Permission) (*vcPermission, error) {
	tmpCtx := context.Background()
	ivtPath, err := find.InventoryPath(tmpCtx, GlobalClient.GetSOAPClient(), r1.Entity.Reference())
	if err != nil {
		return nil, err
	}
	res := &vcPermission{
		Entity:    ivtPath,
		Principal: r1.Principal,
		IsGroup:   r1.Group,
		RoleId:    r1.RoleId,
		Propagate: r1.Propagate,
	}
	return res, nil
}

func (vsc *vSphereClient) ListAllUsers(vcbi *VCBasicInfo) error {
	tmpCtx := context.Background()
	// custom login method
	ssocli, err := vsc.Login2SSOMgmt()
	if err != nil || ssocli == nil {
		log.Errorln("cannot create ssoadmin client, err:", err)
		return err
	}
	log.Debugln("executed: created sso admin client in ListAllUsers")
	// list sso idp src
	vcidps := &vcIdentityProviders{}
	vcidps.IDP, err = ssocli.IdentitySources(tmpCtx)
	if err != nil {
		log.Errorln("get identity sources, err:", err)
		return err
	}
	log.Debugln("executed: list identity sources in ListAllUsers")
	// list sso groups
	grupInfo, err := ssocli.FindGroups(tmpCtx, "")
	if err != nil {
		return err
	}
	log.Debugln("executed: find group in ListAllUsers")
	resGrps := make([]*vcGroup, 0)
	for i := range grupInfo {
		cGrpInfo := grupInfo[i]
		grp := &vcGroup{
			Name: cGrpInfo.Id.Name + "@" + cGrpInfo.Id.Domain,
			Alias: func() string {
				if cGrpInfo.Alias != nil {
					return cGrpInfo.Alias.Name + "@" + cGrpInfo.Alias.Domain
				} else {
					return "-"
				}
			}(),
			Details: cGrpInfo.Details.Description,
		}
		resGrps = append(resGrps, grp)
	}
	vcbi.SSOGroups = resGrps
	// list sso users
	// solution users first
	resUsers := make([]*vcUser, 0)
	soUsers, err := ssocli.FindSolutionUsers(tmpCtx, "")
	if err != nil {
		return err
	}
	log.Debugln("executed: find solution users in ListAllUsers")
	for _, v := range soUsers {
		tU := &vcUser{
			Name: v.Id.Name + "@" + v.Id.Domain,
			Alias: func() string {
				if v.Alias != nil {
					return v.Alias.Name + "@" + v.Alias.Domain
				} else {
					return "-"
				}
			}(),
			Disabled:       v.Disabled,
			Locked:         false,
			Description:    v.Details.Description,
			EmailAddr:      "-",
			IsSolutionUser: true,
			Kind:           "SolutionUsers",
			NickName:       "-",
		}
		resUsers = append(resUsers, tU)
	}
	// personal users next
	pUsers, err := ssocli.FindPersonUsers(tmpCtx, "")
	if err != nil {
		return err
	}
	log.Debugln("executed: find person users in ListAllUsers")
	for _, v := range pUsers {
		pu := &vcUser{
			Name: v.Id.Name + "@" + v.Id.Domain,
			Alias: func() string {
				if v.Alias != nil {
					return v.Alias.Name + "@" + v.Alias.Domain
				} else {
					return "-"
				}
			}(),
			Disabled:       v.Disabled,
			Locked:         v.Locked,
			Description:    v.Details.Description,
			EmailAddr:      v.Details.EmailAddress,
			IsSolutionUser: false,
			Kind:           "PersonalUsers",
			NickName:       v.Details.FirstName + " " + v.Details.LastName,
		}
		resUsers = append(resUsers, pu)
	}
	vcbi.SSOUsers = resUsers
	// list local password policies
	lppreq := &ssotypes.GetLocalPasswordPolicy{
		This: ssocli.ServiceContent.PasswordPolicyService,
	}
	lppi := &lppInfo{}
	retV, err := ssometd.GetLocalPasswordPolicy(tmpCtx, ssocli, lppreq)
	if err != nil {
		log.Errorln("get local password policy err.")
		return err
	}
	lppi.LocalPasswordPolicy = retV.Returnval
	log.Debugln("executed: show local password policy in ListAllUsers")
	vcbi.SSOPasswordPolicyDesc = lppi.String()
	return nil
}

type lppInfo struct {
	LocalPasswordPolicy ssotypes.AdminPasswordPolicy
}

func (r *lppInfo) String() string {
	return fmt.Sprintf(
		"Description: %s,"+
			"MinLength: %d,"+
			"MaxLength: %d,"+
			"MinAlphabeticCount: %d,"+
			"MinUppercaseCount: %d,"+
			"MinLowercaseCount: %d,"+
			"MinNumericCount: %d,"+
			"MinSpecialCharCount: %d,"+
			"MaxIdenticalAdjacentCharacters: %d,"+
			"ProhibitedPreviousPasswordsCount: %d,"+
			"PasswordLifetimeDays: %d.",
		r.LocalPasswordPolicy.Description,
		r.LocalPasswordPolicy.PasswordFormat.LengthRestriction.MinLength,
		r.LocalPasswordPolicy.PasswordFormat.LengthRestriction.MaxLength,
		r.LocalPasswordPolicy.PasswordFormat.AlphabeticRestriction.MinAlphabeticCount,
		r.LocalPasswordPolicy.PasswordFormat.AlphabeticRestriction.MinUppercaseCount,
		r.LocalPasswordPolicy.PasswordFormat.AlphabeticRestriction.MinLowercaseCount,
		r.LocalPasswordPolicy.PasswordFormat.MinNumericCount,
		r.LocalPasswordPolicy.PasswordFormat.MinSpecialCharCount,
		r.LocalPasswordPolicy.PasswordFormat.MaxIdenticalAdjacentCharacters,
		r.LocalPasswordPolicy.ProhibitedPreviousPasswordsCount,
		r.LocalPasswordPolicy.PasswordLifetimeDays,
	)
}
