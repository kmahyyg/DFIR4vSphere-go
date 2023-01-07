package vsphere_api

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

type VCBasicInfo struct {
	ESXHostList   []string               `json:"esx_host_names"`
	ESXHostObjs   []*object.HostSystem   `json:"-"`
	VCAuthoriRole []*vcAuthorizationRole `json:"vc_authorization_roles"`
	VCAuthoriPerm []*vcPermission        `json:"vc_authorization_permissions"`
	EventMaxAge   int                    `json:"event_max_age"`
}

type ESXBasicInfo struct {
}

type vcPermission struct {
	Entity    string `json:"entity"`
	Principal string `json:"principal"`
	IsGroup   bool   `json:"is_group"`
	RoleId    int32  `json:"role_id"`
	Propagate bool   `json:"propagate"`
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
	// role list
	vcbi.VCAuthoriRole = make([]*vcAuthorizationRole, len(rList))
	for i := range rList {
		vcbi.VCAuthoriRole[i] = fromVInternalAuthorizationRoleToOutRole(rList[i])
	}
	// permission set and assignment
	permList, err := authMgr.RetrieveAllPermissions(tmpCtx)
	if err != nil {
		log.Errorln("retr perm list failed.")
		return err
	}
	for i := range permList {
		vcbi.VCAuthoriPerm[i], err = fromVInternalPermissionSetToOutPerm(permList[i])
		if err != nil {
			log.Errorln("type conversion: permission list, err: ", err)
			continue
		}
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
