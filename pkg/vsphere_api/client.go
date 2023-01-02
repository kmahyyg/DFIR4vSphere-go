package vsphere_api

import (
	"context"
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/event"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/session/cache"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
	"net/http"
	"net/url"
	"sync"
)

var (
	GlobalClient = &vSphereClient{}
)

var (
	ErrSessionInvalid = errors.New("current client does NOT have active logged-in session")
	ErrDataInCtx404   = errors.New("data in context not exist")
)

// vSphereClient handle basic authentication and session stuff
type vSphereClient struct {
	soapURL *url.URL
	// skipTLS should be set here since it's always static and user defined it at very beginning
	skipTLS   bool
	httpProxy *url.URL

	// static status mark
	postInitDone    bool
	serverIsVC      bool
	curSessLoggedIn bool

	// soap-based api client
	vmwSoapClient *vim25.Client
	// use a session cache
	curSession *cache.Session

	// finder instance in inventory
	curFinder *find.Finder
	// event manager
	evntMgr    *event.Manager
	evntMaxAge uint32
	// vcsa option manager
	vcsaOptionMgr *object.OptionManager

	// data context when using in the same session
	dataCtx context.Context

	// restapi-based client
	// rest.Client is the base of vim25.Client
	// no need to initialize manually
	// left here for convenience
	__vmwRestClient *rest.Client

	// mutex
	mu *sync.RWMutex
}

// Init for vSphere Client to create environment container
func (vsc *vSphereClient) Init(soapUrl *url.URL, skipTLS bool, proxyURL *url.URL) error {
	vsc.soapURL = soapUrl
	vsc.skipTLS = skipTLS
	vsc.httpProxy = proxyURL
	return nil
}

// NewClient create instance and build session cache to make sure session not leaked,
// must be called after Init() and before any other function call
func (vsc *vSphereClient) NewClient() error {
	// rebuild the whole mu and context
	vsc.dataCtx = context.WithValue(context.TODO(), "data", make(map[string]interface{}))
	vsc.mu = &sync.RWMutex{}
	// soap client initialize
	vsc.vmwSoapClient = new(vim25.Client)
	//
	// here, this vmwSoapClient is not initialized until you try login!
	// all internal field are nil! DO NOT ACCESS!
	// I don't know why VMWare design those things in this shitty way.
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
	log.Debugln("login successfully finished.")
	err = vsc.postLoginSuccessInit()
	if err != nil {
		return err
	}
	log.Debugln("after-login-success initialization complete without error.")
	return nil
}

// postLoginSuccessInit initialize other internal manager or client for further usage
func (vsc *vSphereClient) postLoginSuccessInit() error {
	if !vsc.IsLoggedIn() {
		return ErrSessionInvalid
	}
	// object finder via LDAP over SOAP
	vsc.curFinder = find.NewFinder(vsc.vmwSoapClient, true)
	vsc.postInitDone = true
	return nil
}

// Logout should be called via defer stack, to make sure session is invalid in time.
func (vsc *vSphereClient) Logout() (err error) {
	vsc.curSessLoggedIn = false
	err = vsc.curSession.Logout(context.Background(), vsc.vmwSoapClient)
	if err != nil {
		return err
	}
	return nil
}

// ShowAPIVersion will be used to test connection is working or not
func (vsc *vSphereClient) ShowAPIVersion() (err error) {
	if !vsc.IsLoggedIn() || !vsc.postInitDone {
		err = ErrSessionInvalid
		return
	}
	vsc.serverIsVC = vsc.vmwSoapClient.IsVC()
	log.Infof("Server Is vCenter: %v - version %s", vsc.serverIsVC,
		vsc.vmwSoapClient.ServiceContent.About.Version)
	return nil
}

// IsVCenter will return if this is NOT a standalone ESXi Host
func (vsc *vSphereClient) IsVCenter() bool {
	if !vsc.IsLoggedIn() || !vsc.postInitDone {
		return false
	}
	return vsc.serverIsVC
}

// IsLoggedIn will return if there is an active session
func (vsc *vSphereClient) IsLoggedIn() bool {
	return vsc.curSessLoggedIn
}

// SetCtxData is used to passing volatile data in the same session
func (vsc *vSphereClient) SetCtxData(key string, val interface{}) {
	if vsc.dataCtx == nil || !vsc.postInitDone {
		log.Fatalln("context not initialized in vsc.")
	}
	vsc.mu.Lock()
	defer vsc.mu.Unlock()
	dataMap := vsc.dataCtx.Value("data").(map[string]interface{})
	if dataMap == nil {
		log.Fatalln("context data not initialized.")
	}
	dataMap[key] = val
}

// GetCtxData is used to reading volatile data in the same session
func (vsc *vSphereClient) GetCtxData(key string) (interface{}, error) {
	if vsc.dataCtx == nil || !vsc.postInitDone {
		log.Fatalln("context not initialized in vsc.")
	}
	vsc.mu.RLock()
	defer vsc.mu.RUnlock()
	dataMap := vsc.dataCtx.Value("data").(map[string]interface{})
	if dataMap == nil {
		log.Fatalln("context data not initialized.")
	}
	val, ok := dataMap[key]
	if !ok {
		return nil, ErrDataInCtx404
	}
	return val, nil
}
