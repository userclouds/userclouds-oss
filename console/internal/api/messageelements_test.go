package api_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/console/internal/api"
	consoletesthelpers "userclouds.com/console/testhelpers"
	"userclouds.com/infra/assert"
	cachetesthelpers "userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/workerclient"
	message "userclouds.com/internal/messageelements"
	"userclouds.com/internal/tenantplex"
	plextest "userclouds.com/internal/tenantplex/test"
	"userclouds.com/plex/manager"
)

const testEmailSender = "nobody@gmail.com"
const testEmailSenderName = "Nobody"
const testEmailSubjectTemplate = "email subject"
const testEmailHTMLTemplate = "<h1>html body</h1>"
const testEmailTextTemplate = "text body"
const testSMSBodyTemplate = "sms body"

func emailElementsURL(tenantID uuid.UUID) string {
	return fmt.Sprintf("/api/tenants/%v/emailelements", tenantID)
}

func smsElementsURL(tenantID uuid.UUID) string {
	return fmt.Sprintf("/api/tenants/%v/smselements", tenantID)
}

func loadTenantConfig(t *testing.T, tf *consoletesthelpers.TestFixture) tenantplex.TenantConfig {
	t.Helper()

	ctx := context.Background()

	// load tenant config
	mgr := manager.NewFromDB(tf.ConsoleTenantDB, cachetesthelpers.NewCacheConfig())
	tp, err := mgr.GetTenantPlex(ctx, tf.ConsoleTenantID)
	assert.NoErr(t, err)
	return tp.PlexConfig
}
func newAPIClient(ctx context.Context, t *testing.T) (tf *consoletesthelpers.TestFixture, client *jsonclient.Client) {
	return newAPIClientWithWorkerClient(ctx, t, workerclient.NewTestClient())
}

func newAPIClientWithWorkerClient(ctx context.Context, t *testing.T, wc workerclient.Client) (tf *consoletesthelpers.TestFixture, client *jsonclient.Client) {
	t.Helper()
	tf = consoletesthelpers.NewTestFixtureWithWorkerClient(t, wc)
	return tf, newAPIClientForFixture(ctx, t, tf)
}

var ucAdminLock sync.Mutex

func newAPIClientForFixture(ctx context.Context, t *testing.T, tf *consoletesthelpers.TestFixture) *jsonclient.Client {
	t.Helper()
	assert.NotNil(t, tf, assert.Must())
	ucAdminLock.Lock()
	_, cookie, _ := tf.MakeUCAdmin(ctx)
	ucAdminLock.Unlock()
	client := jsonclient.New(tf.ConsoleServerURL, jsonclient.Cookie(*cookie))
	assert.NotNil(t, client, assert.Must())
	return client
}

func newModifiedMessageTypeMessageElementsFromTenantConfig(tenantID uuid.UUID, tc tenantplex.TenantConfig, mt message.MessageType) api.ModifiedMessageTypeMessageElements {
	mmtme := api.ModifiedMessageTypeMessageElements{
		TenantID:        tenantID,
		AppID:           tc.PlexMap.Apps[0].ID,
		MessageType:     mt,
		MessageElements: message.ElementsByElementType{}}
	appElementGetter := tc.PlexMap.Apps[0].MakeElementGetter(mt)
	for _, elt := range message.ElementTypes(mt) {
		mmtme.MessageElements[elt] = appElementGetter(elt)
	}
	return mmtme
}

func postSaveTenantAppEmailElements(mmtme api.ModifiedMessageTypeMessageElements, tf *consoletesthelpers.TestFixture, c *jsonclient.Client) error {
	request := api.SaveTenantAppMessageElementsRequest{ModifiedMessageTypeMessageElements: mmtme}

	// execute the request
	var response api.SaveTenantAppMessageElementsResponse
	return c.Post(context.Background(), emailElementsURL(tf.ConsoleTenantID), request, &response)
}

func postSaveTenantAppSMSElements(mmtme api.ModifiedMessageTypeMessageElements, tf *consoletesthelpers.TestFixture, c *jsonclient.Client) error {
	request := api.SaveTenantAppMessageElementsRequest{ModifiedMessageTypeMessageElements: mmtme}

	// execute the request
	var response api.SaveTenantAppMessageElementsResponse
	return c.Post(context.Background(), smsElementsURL(tf.ConsoleTenantID), request, &response)
}

func setupCustomizedEmailElements(t *testing.T, tf *consoletesthelpers.TestFixture, c *jsonclient.Client) {
	t.Helper()

	for _, mt := range message.EmailMessageTypes() {
		// load tenant config
		tc := loadTenantConfig(t, tf)

		// set up a request that customizes every element
		tcb := plextest.NewTenantConfigBuilderFromTenantConfig(tc)
		tcb.SwitchToApp(0).
			CustomizeMessageElement(mt, message.EmailSender, testEmailSender).
			CustomizeMessageElement(mt, message.EmailSenderName, testEmailSenderName).
			CustomizeMessageElement(mt, message.EmailSubjectTemplate, testEmailSubjectTemplate).
			CustomizeMessageElement(mt, message.EmailHTMLTemplate, testEmailHTMLTemplate).
			CustomizeMessageElement(mt, message.EmailTextTemplate, testEmailTextTemplate)

		mmtme := newModifiedMessageTypeMessageElementsFromTenantConfig(tf.ConsoleTenantID, tcb.Build(), mt)
		assert.IsNil(t, postSaveTenantAppEmailElements(mmtme, tf, c), assert.Must())
	}
}

func setupCustomizedSMSElements(t *testing.T, tf *consoletesthelpers.TestFixture, c *jsonclient.Client) {
	t.Helper()

	for _, mt := range message.SMSMessageTypes() {
		// load tenant config
		tc := loadTenantConfig(t, tf)

		// set up a request that customizes every element
		tcb := plextest.NewTenantConfigBuilderFromTenantConfig(tc)
		tcb.SwitchToApp(0).CustomizeMessageElement(mt, message.SMSBodyTemplate, testSMSBodyTemplate)

		mmtme := newModifiedMessageTypeMessageElementsFromTenantConfig(tf.ConsoleTenantID, tcb.Build(), mt)
		assert.IsNil(t, postSaveTenantAppSMSElements(mmtme, tf, c), assert.Must())
	}
}

func TestDefaultMessageElementsTenantPlex(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tf, _ := newAPIClient(ctx, t)

	// load tenant config
	tc := loadTenantConfig(t, tf)

	// verify that we have exactly one app and no custom elements by default
	// so we don't have to check in our other tests
	assert.Equal(t, len(tc.PlexMap.Apps), 1, assert.Must())

	for _, mt := range message.MessageTypes() {
		defaultElementGetter := message.MakeElementGetter(mt)
		appElementGetter := tc.PlexMap.Apps[0].MakeElementGetter(mt)
		for _, elt := range message.ElementTypes(mt) {
			assert.Equal(t, defaultElementGetter(elt), appElementGetter(elt))
		}
	}
}

func TestGetMessageElements(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tf, c := newAPIClient(ctx, t)

	// test email elements
	var emailResp api.GetTenantAppMessageElementsResponse
	assert.IsNil(t, c.Get(ctx, emailElementsURL(tf.ConsoleTenantID), &emailResp), assert.Must())

	// verify response is valid and has the expected number of apps and message type entries
	tame := emailResp.TenantAppMessageElements
	assert.IsNil(t, tame.Validate(), assert.Must())
	assert.Equal(t, len(tame.AppMessageElements), 1, assert.Must())
	assert.Equal(t, len(tame.AppMessageElements[0].MessageTypeMessageElements), len(message.EmailMessageTypes()), assert.Must())

	// verify all default elements are correct and that no custom elements are set
	for _, mtme := range tame.AppMessageElements[0].MessageTypeMessageElements {
		defaultElementGetter := message.MakeElementGetter(mtme.Type)
		for _, me := range mtme.MessageElements {
			assert.Equal(t, me.CustomValue, "")
			assert.Equal(t, me.DefaultValue, defaultElementGetter(me.Type))
		}
	}

	// repeat for sms elements

	var smsResp api.GetTenantAppMessageElementsResponse
	assert.IsNil(t, c.Get(ctx, smsElementsURL(tf.ConsoleTenantID), &smsResp), assert.Must())

	// verify response is valid and has the expected number of apps and message type entries
	tame = smsResp.TenantAppMessageElements
	assert.IsNil(t, tame.Validate(), assert.Must())
	assert.Equal(t, len(tame.AppMessageElements), 1, assert.Must())
	assert.Equal(t, len(tame.AppMessageElements[0].MessageTypeMessageElements), len(message.SMSMessageTypes()), assert.Must())

	// verify all default elements are correct and that no custom elements are set
	for _, mtme := range tame.AppMessageElements[0].MessageTypeMessageElements {
		defaultElementGetter := message.MakeElementGetter(mtme.Type)
		for _, me := range mtme.MessageElements {
			assert.Equal(t, me.CustomValue, "")
			assert.Equal(t, me.DefaultValue, defaultElementGetter(me.Type))
		}
	}
}

func TestCustomizeMessageElements(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tf, c := newAPIClient(ctx, t)

	// setup and validate custom email elements

	setupCustomizedEmailElements(t, tf, c)

	tc := loadTenantConfig(t, tf)

	for _, mt := range message.EmailMessageTypes() {
		appElementGetter := tc.PlexMap.Apps[0].MakeElementGetter(mt)
		assert.Equal(t, appElementGetter(message.EmailSender), testEmailSender)
		assert.Equal(t, appElementGetter(message.EmailSenderName), testEmailSenderName)
		assert.Equal(t, appElementGetter(message.EmailSubjectTemplate), testEmailSubjectTemplate)
		assert.Equal(t, appElementGetter(message.EmailHTMLTemplate), testEmailHTMLTemplate)
		assert.Equal(t, appElementGetter(message.EmailTextTemplate), testEmailTextTemplate)
	}

	// setup and validate custom SMS elements

	setupCustomizedSMSElements(t, tf, c)

	tc = loadTenantConfig(t, tf)

	for _, mt := range message.SMSMessageTypes() {
		appElementGetter := tc.PlexMap.Apps[0].MakeElementGetter(mt)
		assert.Equal(t, appElementGetter(message.SMSBodyTemplate), testSMSBodyTemplate)
	}
}

func TestRemoveMessageElements(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tf, c := newAPIClient(ctx, t)

	// set up tenant config to have all message elements customized

	setupCustomizedEmailElements(t, tf, c)
	setupCustomizedSMSElements(t, tf, c)

	for _, mt := range message.EmailMessageTypes() {
		// remove all message elements from local copy of tenant config
		tcb := plextest.NewTenantConfigBuilderFromTenantConfig(loadTenantConfig(t, tf))
		ab := tcb.SwitchToApp(0)
		for _, elt := range message.ElementTypes(mt) {
			ab.CustomizeMessageElement(mt, elt, "")
		}

		// post a save request for this local copy
		mmtme := newModifiedMessageTypeMessageElementsFromTenantConfig(tf.ConsoleTenantID, tcb.Build(), mt)
		assert.IsNil(t, postSaveTenantAppEmailElements(mmtme, tf, c), assert.Must())
	}

	for _, mt := range message.SMSMessageTypes() {
		// remove all message elements from local copy of tenant config
		tcb := plextest.NewTenantConfigBuilderFromTenantConfig(loadTenantConfig(t, tf))
		ab := tcb.SwitchToApp(0)
		for _, elt := range message.ElementTypes(mt) {
			ab.CustomizeMessageElement(mt, elt, "")
		}

		// post a save request for this local copy
		mmtme := newModifiedMessageTypeMessageElementsFromTenantConfig(tf.ConsoleTenantID, tcb.Build(), mt)
		assert.IsNil(t, postSaveTenantAppSMSElements(mmtme, tf, c), assert.Must())
	}

	// verify that there are no customized message elements
	tame := api.NewTenantAppMessageElementsFromTenantConfig(tf.ConsoleTenantID, loadTenantConfig(t, tf), message.MessageTypes())
	for _, mtme := range tame.AppMessageElements[0].MessageTypeMessageElements {
		for _, me := range mtme.MessageElements {
			assert.Equal(t, me.CustomValue, "")
		}
	}
}

func TestSaveDefaultMessageElements(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tf, c := newAPIClient(ctx, t)

	// set up tenant config to have all email elements customized
	setupCustomizedEmailElements(t, tf, c)

	for _, mt := range message.EmailMessageTypes() {
		// set all email elements to be the default value in tenant config
		tcb := plextest.NewTenantConfigBuilderFromTenantConfig(loadTenantConfig(t, tf))
		ab := tcb.SwitchToApp(0)
		defaultElementGetter := message.MakeElementGetter(mt)
		for _, elt := range message.ElementTypes(mt) {
			ab.CustomizeMessageElement(mt, elt, defaultElementGetter(elt))
		}

		// post a save request for this local copy
		mmtme := newModifiedMessageTypeMessageElementsFromTenantConfig(tf.ConsoleTenantID, tcb.Build(), mt)
		assert.IsNil(t, postSaveTenantAppEmailElements(mmtme, tf, c), assert.Must())
	}

	// verify that there are no customized message elements
	tame := api.NewTenantAppMessageElementsFromTenantConfig(tf.ConsoleTenantID, loadTenantConfig(t, tf), message.EmailMessageTypes())
	for _, mtme := range tame.AppMessageElements[0].MessageTypeMessageElements {
		for _, me := range mtme.MessageElements {
			assert.Equal(t, me.CustomValue, "")
		}
	}

	// repeat for SMS elements
	setupCustomizedSMSElements(t, tf, c)

	for _, mt := range message.SMSMessageTypes() {
		// set all SMS elements to be the default value in tenant config
		tcb := plextest.NewTenantConfigBuilderFromTenantConfig(loadTenantConfig(t, tf))
		ab := tcb.SwitchToApp(0)
		defaultElementGetter := message.MakeElementGetter(mt)
		for _, elt := range message.ElementTypes(mt) {
			ab.CustomizeMessageElement(mt, elt, defaultElementGetter(elt))
		}

		// post a save request for this local copy
		mmtme := newModifiedMessageTypeMessageElementsFromTenantConfig(tf.ConsoleTenantID, tcb.Build(), mt)
		assert.IsNil(t, postSaveTenantAppSMSElements(mmtme, tf, c), assert.Must())
	}

	// verify that there are no customized message elements
	tame = api.NewTenantAppMessageElementsFromTenantConfig(tf.ConsoleTenantID, loadTenantConfig(t, tf), message.SMSMessageTypes())
	for _, mtme := range tame.AppMessageElements[0].MessageTypeMessageElements {
		for _, me := range mtme.MessageElements {
			assert.Equal(t, me.CustomValue, "")
		}
	}
}

func TestBadMessageElementsRequests(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tf, c := newAPIClient(ctx, t)

	// set up tenant config to have all email elements customized, remove element type and ensure save fails
	setupCustomizedEmailElements(t, tf, c)
	tc := loadTenantConfig(t, tf)
	mmtme := newModifiedMessageTypeMessageElementsFromTenantConfig(tf.ConsoleTenantID, tc, message.EmailResetPassword)
	delete(mmtme.MessageElements, message.EmailSender)
	assert.NotNil(t, postSaveTenantAppEmailElements(mmtme, tf, c))

	// set up tenant config to have all email elements customized, set invalid message type and ensure save fails
	setupCustomizedEmailElements(t, tf, c)
	tc = loadTenantConfig(t, tf)
	mmtme = newModifiedMessageTypeMessageElementsFromTenantConfig(tf.ConsoleTenantID, tc, message.EmailPasswordlessLogin)
	mmtme.MessageType = "bad_email_type"
	assert.NotNil(t, postSaveTenantAppEmailElements(mmtme, tf, c))

	// set up tenant config to have all email elements customized, set invalid element type and ensure save fails
	setupCustomizedEmailElements(t, tf, c)
	tc = loadTenantConfig(t, tf)
	mmtme = newModifiedMessageTypeMessageElementsFromTenantConfig(tf.ConsoleTenantID, tc, message.EmailVerifyEmail)
	newValue := mmtme.MessageElements[message.EmailSender]
	mmtme.MessageElements["bad_element_type"] = newValue
	delete(mmtme.MessageElements, message.EmailSender)
	assert.NotNil(t, postSaveTenantAppEmailElements(mmtme, tf, c))

	// save elements where we add an invalid message element
	setupCustomizedEmailElements(t, tf, c)
	tc = loadTenantConfig(t, tf)
	tcb := plextest.NewTenantConfigBuilderFromTenantConfig(tc)
	tcb.SwitchToApp(0).CustomizeMessageElement(message.EmailInviteNewUser, message.EmailSender, "not_an_email_address")
	mmtme = newModifiedMessageTypeMessageElementsFromTenantConfig(tf.ConsoleTenantID, tcb.Build(), message.EmailInviteNewUser)
	assert.NotNil(t, postSaveTenantAppEmailElements(mmtme, tf, c))

	// save elements where we add an invalid templatized element (unsupported parameter)
	templateElementTypes := []message.ElementType{message.EmailSubjectTemplate, message.EmailHTMLTemplate, message.EmailTextTemplate}
	for _, elt := range templateElementTypes {
		setupCustomizedEmailElements(t, tf, c)
		tc = loadTenantConfig(t, tf)
		tcb = plextest.NewTenantConfigBuilderFromTenantConfig(tc)
		tcb.SwitchToApp(0).CustomizeMessageElement(message.EmailInviteNewUser, elt, "This has a {{BadParameter}}")
		mmtme = newModifiedMessageTypeMessageElementsFromTenantConfig(tf.ConsoleTenantID, tcb.Build(), message.EmailInviteNewUser)
		assert.NotNil(t, postSaveTenantAppEmailElements(mmtme, tf, c))
	}
}
