package vsphere_api

import (
	"context"
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/session/cache"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
	"net/http"
	"net/url"
)

var (
	GlobalClient = &vSphereClient{}
)

var (
	ErrSessionInvalid = errors.New("current client does NOT have active logged-in session")
)

// vSphereClient handle basic authentication and session stuff
type vSphereClient struct {
	soapURL   *url.URL
	skipTLS   bool
	httpProxy *url.URL

	// soap-based api client
	vmwSoapClient *vim25.Client
	// use a session cache
	curSession      *cache.Session
	curSessLoggedIn bool
	// vCenter flag
	serverIsVCenter bool

	// restapi-based client
	// rest.Client is the base of vim25.Client
	// no need to initialize manually
	// left here for convenience
	__vmwRestClient *rest.Client
}

// Init for vSphere Client to create environment container
func (vsc *vSphereClient) Init(soapUrl *url.URL, skipTLS bool, proxyURL *url.URL) error {
	vsc.soapURL = soapUrl
	vsc.skipTLS = skipTLS
	vsc.httpProxy = proxyURL
	return nil
}

// NewClient create instance and build session cache to make sure session not leaked
func (vsc *vSphereClient) NewClient() error {
	// soap client initialize
	vsc.vmwSoapClient = new(vim25.Client)
	//
	// here, this vmwSoapClient is not initialized until you try login!
	// all internal field are nil! DO NOT ACCESS!
	// I don't know why VMWare design those things in this shitty way.
	//
	//
	// build session cache
	vsc.curSession = new(cache.Session)
	vsc.curSession.URL = vsc.soapURL
	vsc.curSession.Insecure = vsc.skipTLS
	return nil
}

// LoginViaPassword will try to log in using credentials, if Token is required, you may query STS, then
// issue ticket or token yourself.
func (vsc *vSphereClient) LoginViaPassword() (err error) {
	// before login, configure client
	soapConfigFunc := func(sc *soap.Client) error {
		if vsc.httpProxy != nil {
			sc.DefaultTransport().Proxy = http.ProxyURL(vsc.httpProxy)
		}
		if vsc.skipTLS {
			sc.DefaultTransport().TLSClientConfig.InsecureSkipVerify = vsc.skipTLS
		}
		sc.UserAgent = "DFIR4vSphere-Go"
		// now this client is initialized without error
		return nil
	}
	// start login
	loginErr := vsc.curSession.Login(context.Background(), vsc.vmwSoapClient, soapConfigFunc)
	if loginErr != nil {
		return loginErr
	} else {
		vsc.curSessLoggedIn = true
	}
	return nil
}

// Logout should be called via defer stack, to make sure session is invalid in time.
func (vsc *vSphereClient) Logout() (err error) {
	err = vsc.curSession.Logout(context.Background(), vsc.vmwSoapClient)
	if err != nil {
		return err
	}
	return nil
}

// ShowAPIVersion will be used to test connection is working or not
func (vsc *vSphereClient) ShowAPIVersion() (err error) {
	if !vsc.curSessLoggedIn {
		err = ErrSessionInvalid
		return
	}
	vsc.serverIsVCenter = vsc.vmwSoapClient.IsVC()
	log.Infof("Server Is vCenter: %v - version %s", vsc.serverIsVCenter,
		vsc.vmwSoapClient.ServiceContent.About.Version)
	return nil
}
