package azarmpolicy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

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

	myPolicy := &ArmRequestMetricPolicy{} // TODO add collector and able to assert on it.
	clientOptions := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			PerCallPolicies: []policy.Policy{myPolicy},
		},
		DisableRPRegistration: true,
	}
	client, err := armcontainerservice.NewManagedClustersClient(testInfo.SPNClientSecret, token, clientOptions)
	assert.NoError(t, err)

	poller, err := client.BeginCreateOrUpdate(context.Background(), "test", "test", armcontainerservice.ManagedCluster{Location: to.Ptr("eastus")}, nil)
	assert.NoError(t, err)

	result, err := poller.Result(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Println(result)
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
