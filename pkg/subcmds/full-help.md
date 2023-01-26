# DFIR4vSphere Full Sub-Command Help List

This is an interactive command-line tool to help you do forensics job for vSphere products.

Version v0.0.1-snapshot6-0-g8c2522c

Full Commands:
- `support_bundle`
- `try_reconnect`
- `basic_info`
- `vi_events`
- `exit`
- `full_help`

The command parameters should be wrapped using `()`. If there are multiple values, use `|` as seperator.

## full_help

Show this help document.

## exit

Logout current session and exit.

## vi_events

Extract VI events from vCenter. If connected server is not vCenter, throw unsupported error.

Program will ask you which ESXi host and VCSA you want to collect all VI events. If you choose `light_mode`, only the
following listed types of events will be collected. Output to CSV file.

Output file: `VIEvents_<Unix Timestamp>.csv`

Params: `(light_mode=bool) (selected_host=esxi_hostname1|esxi_hostname2)`

```go
package vsphere_api

var (
	lightVIEventTypesId = []string{"ad.event.JoinDomainEvent", "VmFailedToSuspendEvent", "VmSuspendedEvent",
		"VmSuspendingEvent", "VmDasUpdateOkEvent", "VmReconfiguredEvent", "UserUnassignedFromGroup",
		"UserAssignedToGroup", "UserPasswordChanged", "AccountCreatedEvent", "AccountRemovedEvent",
		"AccountUpdatedEvent", "UserLoginSessionEvent", "RoleAddedEvent", "RoleRemovedEvent", "RoleUpdatedEvent",
		"TemplateUpgradeEvent", "TemplateUpgradedEvent", "PermissionAddedEvent", "PermissionUpdatedEvent",
		"PermissionRemovedEvent", "LocalTSMEnabledEvent", "DatastoreFileDownloadEvent", "DatastoreFileUploadEvent",
		"DatastoreFileDeletedEvent", "VmAcquiredMksTicketEvent",
		"com.vmware.vc.guestOperations.GuestOperationAuthFailure", "com.vmware.vc.guestOperations.GuestOperation",
		"esx.audit.ssh.enabled", "esx.audit.ssh.session.failed", "esx.audit.ssh.session.closed",
		"esx.audit.ssh.session.opened", "esx.audit.account.locked", "esx.audit.account.loginfailures",
		"esx.audit.dcui.login.passwd.changed", "esx.audit.dcui.enabled", "esx.audit.dcui.disabled",
		"esx.audit.lockdownmode.exceptions.changed", "esx.audit.shell.disabled", "esx.audit.shell.enabled",
		"esx.audit.lockdownmode.disabled", "esx.audit.lockdownmode.enabled", "com.vmware.sso.LoginSuccess",
		"com.vmware.sso.LoginFailure", "com.vmware.sso.Logout", "com.vmware.sso.PrincipalManagement",
		"com.vmware.sso.RoleManagement", "com.vmware.sso.IdentitySourceManagement", "com.vmware.sso.DomainManagement",
		"com.vmware.sso.ConfigurationManagement", "com.vmware.sso.CertificateManager",
		"com.vmware.trustmanagement.VcTrusts", "com.vmware.trustmanagement.VcIdentityProviders",
		"com.vmware.cis.CreateGlobalPermission", "com.vmware.cis.CreatePermission",
		"com.vmware.cis.RemoveGlobalPermission", "com.vmware.cis.RemovePermission", "com.vmware.vc.host.Crypto.Enabled",
		"com.vmware.vc.host.Crypto.HostCryptoDisabled", "ProfileCreatedEvent", "ProfileChangedEvent",
		"ProfileRemovedEvent", "ProfileAssociatedEvent", "esx.audit.esximage.vib.install.successful",
		"esx.audit.esximage.hostacceptance.changed", "esx.audit.esximage.vib.remove.successful"}
)
```

Load Event from Event Manager. If lite mode, using Event Type ID for loop.

## basic_info

Output: `BasicInfo_<Unix Timestamp>.json`

Will collect the following information:

For ESXi-standalone host:
- [x] | Get running service status (covered in HostSystem "config" property above)
- [x] | Get authentication information (covered in HostSystem "config" property above)
- [ ] | Expose ESXCli v2, and Do following:
    - [x] | Get System Version (covered in HostSystem "config" property above)
    - [ ] | List System Account
    - [ ] | List System Permission
    - [ ] | List System Modules
    - [ ] | List System Processes
    - [x] | List System Certificate Store
    - [ ] | (If Version >= 7.0.2) Get System Encryption Settings
    - [ ] | (If Version >= 7.0.2) Get System Guest Store Repository
    - [ ] | (If Version >= 7.0.2) (list changed items only) List System Advanced Settings using Delta method
    - [ ] | (If Version >= 7.0.2) (list changed items only) List System Kernel Settings List
    - [ ] | (If Version >= 7.0.2) Get System SysLog Config
    - [ ] | (If Version >= 7.0.0) Get System BaseImage Information
    - [ ] | (If Version >= 7.0.0) Get Software VIBs
    - [ ] | (If Version >= 7.0.0) Get Software Profiles
    - [x] | (If Version >= 7.0.0) List Storage IOFilters (covered in HostSystem "config" property above)
    - [ ] | List storage filesystem
    - [x] | Network interface IPs and routes (covered in HostSystem "config" property above)
    - [ ] | Network neighbor list using ARP cache
    - [x] | Network DNS IPs (covered in HostSystem "config" property above)
    - [ ] | Network Connections status
    - [ ] | Network PortGroup VM Lists
    - [x] | Network vSwitch List (covered in HostSystem "config" property above)

The corresponding `esxcli` command lists are, without prefix `govc host.esxcli` :
- [x] done | `system version get`
- `system account list`
- `system permission list`
- `system module list`
- `system process list`
- [x] done | `system security certificatestore list`
- `system settings encryption get`
- `system settings gueststore repository get`
- `system settings advanced list`
- `system settings kernel list`
- `system syslog config get`
- `software vib signature verify`
- `software baseimage get`
- `software vib get`
- `software profile get`
- [x] done | `storage iofilter list`
- `storage filesystem list`
- [x] done | `network ip interface ipv4 get`
- [x] done | `network ip interface ipv6 get`
- [x] done | `network ip route ipv4 list`
- [x] done | `network ip route ipv6 list`
- `network ip neighbor list`
- [x] done | `network ip dns server list`
- `network ip connection list`
- `network vm list`
- [x] done | `network vswitch standard list`

For vCenter-managed ESXi host:
- In addition to standalone ESXi Host, will do following things:
    - [x] | Get Connected ESXi Hosts list
    - [x] | Get Permissions in vCenter sorted via Principal (Level: Entities@DataCenter)
    - [x] | Get Local and SSO users
    - [x] | Get Advanced Settings "event.maxAge" to determine last X days event to retrieve

## try_reconnect

Cleanup program internal VMWare Product API Client. Build new one and re-authenticate.

## support_bundle

Params: `(selected_host=all)`

Generate and download support bundle from vCenter VCSA or ESXi. Will ask you which host you would like to cover.

However, for standalone ESXi Host, `selected_host` will be ignored. for vCenter, it still NEEDs TO BE TESTED.