package restClientV1

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/jypelle/mifasol/internal/tool"
	"github.com/jypelle/mifasol/restApiV1"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const JsonContentType = "application/json"

var (
	ErrBadHostname        = fmt.Errorf("Bad hostname: Mifasol server is available but should be reconfigured to accept connection with specified hostname")
	ErrBadCertificate     = fmt.Errorf("Mifasol server certificate has changed")
	ErrInvalidCertificate = fmt.Errorf("Invalid certificate: Mifasol server is available but should regenerate its SSL certificate.")
)

type RestClient struct {
	ClientConfig RestConfig
	httpClient   *http.Client
	token        *restApiV1.Token
}

func NewRestClient(clientConfig RestConfig) (*RestClient, error) {

	var rootCAPool *x509.CertPool = nil

	// Load self-signed server certificate
	if clientConfig.GetServerSsl() && clientConfig.GetServerSelfSigned() {

		// Define Root CA
		rootCAPool = x509.NewCertPool()

		existServerCert, err := tool.IsFileExists(clientConfig.GetCompleteConfigCertFilename())
		if err != nil {
			return nil, fmt.Errorf("Unable to access %s: %v\n", clientConfig.GetCompleteConfigCertFilename(), err)
		}
		if !existServerCert {
			// First connection to mifasol server: retrieve & store self-signed server certificate
			insecureTr := &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
					DualStack: true,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}

			insecureClient := &http.Client{
				Transport: insecureTr,
				Timeout:   time.Second * time.Duration(clientConfig.GetTimeout()),
			}

			// Prepare the request
			req, err := http.NewRequest("GET", getServerUrl(clientConfig)+"/isalive", nil)
			if err != nil {
				return nil, fmt.Errorf("Unable to connect to mifasol server: %v", err)
			}

			// Send the request
			response, err := insecureClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("Unable to connect to mifasol server: %v", err)
			}
			defer response.Body.Close()

			if len(response.TLS.PeerCertificates) == 0 {
				return nil, fmt.Errorf("Unable to connect to mifasol server: certificate is missing")
			}

			// Retrieve server certificate
			cert := response.TLS.PeerCertificates[0]

			// Save server certificate
			tool.CertToFile(clientConfig.GetCompleteConfigCertFilename(), cert.Raw)

			// Append server certificate to root CAs
			rootCAPool.AppendCertsFromPEM(cert.Raw)

		} else {
			// Load local server certificate
			certPem, err := ioutil.ReadFile(clientConfig.GetCompleteConfigCertFilename())
			if err != nil {
				return nil, fmt.Errorf("Reading server certificate failed : %v", err)
			}

			// Append server certificate to root CAs
			rootCAPool.AppendCertsFromPEM(certPem)
		}

	}

	// Configure client
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
			RootCAs:            rootCAPool,
		},
	}

	restClient := &RestClient{
		ClientConfig: clientConfig,
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   time.Second * time.Duration(clientConfig.GetTimeout()),
		},
	}

	// Check secure connection

	// Prepare the request
	req, err := http.NewRequest("GET", getServerUrl(clientConfig)+"/isalive", nil)
	if err != nil {
		return nil, fmt.Errorf("Unable to prepare mifasol server connection: %v\n", err)
	}

	// Send the request
	response, err := restClient.httpClient.Do(req)
	if err != nil {
		if urlErr, ok := err.(*url.Error); ok {
			if _, ok := urlErr.Err.(x509.HostnameError); ok {
				return nil, ErrBadHostname
			}
			if _, ok := urlErr.Err.(x509.UnknownAuthorityError); ok {
				return nil, ErrBadCertificate
			}
			if _, ok := urlErr.Err.(x509.CertificateInvalidError); ok {
				return nil, ErrInvalidCertificate
			}
		}
		return nil, fmt.Errorf("Unable to connect to mifasol server: %v", err)
	} else {
		defer response.Body.Close()
	}

	return restClient, nil
}

func getServerUrl(restConfig RestConfig) string {
	if restConfig.GetServerSsl() {
		return "https://" + restConfig.GetServerHostname() + ":" + strconv.FormatInt(restConfig.GetServerPort(), 10)
	} else {
		return "http://" + restConfig.GetServerHostname() + ":" + strconv.FormatInt(restConfig.GetServerPort(), 10)
	}
}

func (c *RestClient) getServerApiUrl() string {
	return getServerUrl(c.ClientConfig) + "/api/v1"
}

// doRequest prepare and send an http request, managing access token renewal for expired token
func (c *RestClient) doRequest(method, relativeUrl string, contentType string, body io.Reader) (*http.Response, ClientError) {

	// Dear mifasolsrv, could you gimme a token ?
	if c.token == nil {
		cliErr := c.refreshToken()
		if cliErr != nil {
			return nil, cliErr
		}
	}

	// Prepare the request
	req, err := http.NewRequest(method, c.getServerApiUrl()+relativeUrl, body)
	if err != nil {
		return nil, NewClientError(err)
	}

	// Embed the token in the request
	req.Header.Add("Authorization", "Bearer "+c.token.AccessToken)

	// Add optional body content for POST & PUT request
	if body != nil {
		req.Header.Set("Content-Type", contentType)
	}

	// Send the request
	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, NewClientError(err)
	}

	// Is the response OK ?
	cliErr := checkStatusCode(response)
	if cliErr != nil {
		// Is the token expired ?
		if cliErr.Code() == restApiV1.InvalidTokenErrorCode {
			// Ask a new one and retry
			c.token = nil
			return c.doRequest(method, relativeUrl, contentType, body)
		}

		return nil, cliErr
	}

	// Return response
	return response, nil
}

func (c *RestClient) doGetRequest(relativeUrl string) (*http.Response, ClientError) {
	return c.doRequest("GET", relativeUrl, "", nil)
}
func (c *RestClient) doDeleteRequest(relativeUrl string) (*http.Response, ClientError) {
	return c.doRequest("DELETE", relativeUrl, "", nil)
}
func (c *RestClient) doPostRequest(relativeUrl string, contentType string, body io.Reader) (*http.Response, ClientError) {
	return c.doRequest("POST", relativeUrl, contentType, body)
}
func (c *RestClient) doPutRequest(relativeUrl string, contentType string, body io.Reader) (*http.Response, ClientError) {
	return c.doRequest("PUT", relativeUrl, contentType, body)
}

func checkStatusCode(response *http.Response) ClientError {

	if response.StatusCode >= 400 {
		var apiErr restApiV1.ApiError
		if err := json.NewDecoder(response.Body).Decode(&apiErr); err != nil {
			apiErr.ErrorCode = restApiV1.UnknownErrorCode
		}

		return &apiErr
	}

	return nil
}

func (c *RestClient) UserId() restApiV1.UserId {
	if c.token == nil {
		cliErr := c.refreshToken()
		if cliErr != nil {
			return "xxx"
		}
	}
	return c.token.UserId
}
