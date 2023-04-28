package azarmpolicy

import (
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

// ArmError is unified Error Experience across AzureResourceManager, it contains Code Message.
type ArmError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// armErrorResponse is internal type, to extract ArmError from response body.
type armErrorResponse struct {
	Error ArmError `json:"error"`
}

// ArmRequestMetricCollector is a interface that collectors need to implement.
// TODO: use *policy.Request or *http.Request?
type ArmRequestMetricCollector interface {
	// RequestStarted is called when a request is about to be sent.
	// context is not provided, get it from Request.Context()
	RequestStarted(*http.Request)

	// RequestCompleted is called when a request is finished (statusCode < 400)
	// context is not provided, get it from Request.Context()
	RequestCompleted(*http.Request, *http.Response)

	// RequestFailed is called when a request is failed (statusCode > 399)
	// context is not provided, get it from Request.Context()
	RequestFailed(*http.Request, *http.Response, *ArmError)
}

// ArmRequestMetricPolicy is a policy that collects metrics/telemetry for ARM requests.
type ArmRequestMetricPolicy struct {
	Collector ArmRequestMetricCollector
}

// Do implements the azcore/policy.Policy interface.
func (p *ArmRequestMetricPolicy) Do(req *policy.Request) (*http.Response, error) {
	p.requestStarted(req.Raw())
	resp, err := req.Next()
	if err != nil {
		// either it's a transport error
		// or it is already handled by previous policy
		// TODO: distinguash
		// - Context Cancelled
		// - ClientTimeout (context still valid, but request exceed certain threshold)
		// - Transport Error (DNS/Dail/TLS/ServerTimeout)
		p.requestFailed(req.Raw(), resp, &ArmError{Code: "TransportError", Message: err.Error()})
		return resp, err
	}

	if resp == nil {
		p.requestFailed(req.Raw(), resp, &ArmError{Code: "UnexpectedTransportorBehavior", Message: "transport return nil, nil"})
		return resp, nil
	}

	if resp != nil && resp.StatusCode > 399 {
		// for 4xx, 5xx response, ARM should include {error:{code, message}} in the body
		err := runtime.NewResponseError(resp)
		respErr := &azcore.ResponseError{}
		if errors.As(err, &respErr) {
			p.requestFailed(req.Raw(), resp, &ArmError{Code: respErr.ErrorCode, Message: ""})
		} else {
			p.requestFailed(req.Raw(), resp, &ArmError{Code: "NotAnArmError", Message: "Response body is not in ARM error form: {error:{code, message}}"})
		}

		// just an observer, caller/client have responder to handle application error.
		return resp, nil
	}

	p.requestCompleted(req.Raw(), resp)
	return resp, nil
}

// shortcut function to handle nil collector
func (p *ArmRequestMetricPolicy) requestStarted(req *http.Request) {
	if p.Collector != nil {
		p.Collector.RequestStarted(req)
	}
}

// shortcut function to handle nil collector
func (p *ArmRequestMetricPolicy) requestCompleted(req *http.Request, resp *http.Response) {
	if p.Collector != nil {
		p.Collector.RequestCompleted(req, resp)
	}
}

// shortcut function to handle nil collector
func (p *ArmRequestMetricPolicy) requestFailed(req *http.Request, resp *http.Response, err *ArmError) {
	if p.Collector != nil {
		p.Collector.RequestFailed(req, resp, err)
	}
}
