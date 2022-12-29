package vsphere_api

import (
	"github.com/vmware/govmomi/vim25/types"
	"strings"
)

func objRefTypeMatch(inref types.ManagedObjectReference, outref types.ManagedObjectReference) bool {
	return strings.EqualFold(inref.Type, outref.Type)
}
