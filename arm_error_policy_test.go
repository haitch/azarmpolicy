package azarmpolicy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
	"github.com/stretchr/testify/assert"
)

func TestArmRequestMetrics(t *testing.T) {
	testInfo, err := testInfoFromEnv()
	if err != nil {
		t.Skipf("test requires setup: %s", err)
	}
	token, err := azidentity.NewClientSecretCredential(testInfo.TenantID, testInfo.SPNClientID, testInfo.SPNClientSecret, nil)
	assert.NoError(t, err)

	myPolicy := &ArmRequestMetricPolicy{
		Collector: &myCollector{},
	}
	clientOptions := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			PerCallPolicies: []policy.Policy{myPolicy},
		},
		DisableRPRegistration: true,
	}
	client, err := armcontainerservice.NewManagedClustersClient("notexistingSub", token, clientOptions)
	assert.NoError(t, err)

	_, err = client.BeginCreateOrUpdate(context.Background(), "test", "test", armcontainerservice.ManagedCluster{Location: to.Ptr("eastus")}, nil)
	// here the error is parsed from response body twice
	// 1. by ArmRequestMetricPolicy, parse and log, and throw away.
	// 2. by generated client function: runtime.HasStatusCode, and return to here.
	assert.Error(t, err)

	respErr := &azcore.ResponseError{}
	assert.True(t, errors.As(err, &respErr))
	assert.Equal(t, respErr.ErrorCode, "InvalidSubscriptionId")
}

var _ ArmRequestMetricCollector = &myCollector{}

type myCollector struct{}

func (c *myCollector) RequestStarted(req *http.Request) {
	fmt.Printf("RequestStarted, %s\n", req.URL)
}

func (c *myCollector) RequestCompleted(req *http.Request, resp *http.Response) {
	fmt.Printf("RequestCompleted %s, %d\n", req.URL, resp.StatusCode)
}

func (c *myCollector) RequestFailed(req *http.Request, resp *http.Response, armErr *ArmError) {
	fmt.Printf("RequestCompleted %s, %d, %s\n", req.URL, resp.StatusCode, armErr.Code)
}

type aadInfo struct {
	TenantID        string
	SPNClientID     string
	SPNClientSecret string
	SubscriptionID  string
}

func testInfoFromEnv() (*aadInfo, error) {
	tenantID, ok := os.LookupEnv("AAD_Tenant")
	if !ok {
		return nil, errors.New("AAD_Tenant is not set")
	}

	clientID, ok := os.LookupEnv("AAD_ClientID")
	if !ok {
		return nil, errors.New("AAD_ClientID is not set")
	}

	clientSecret, ok := os.LookupEnv("AAD_ClientSecret")
	if !ok {
		return nil, errors.New("AAD_ClientSecret is not set")
	}

	subscriptionID, ok := os.LookupEnv("Azure_Subscription")
	if !ok {
		return nil, errors.New("Azure_Subscription is not set")
	}

	return &aadInfo{
		TenantID:        tenantID,
		SPNClientID:     clientID,
		SPNClientSecret: clientSecret,
		SubscriptionID:  subscriptionID,
	}, nil
}
