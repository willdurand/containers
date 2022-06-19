package registry

import (
	"net/http"
)

type httpClient struct {
	client *http.Client
	token  string
}

func newHttpClientWithAuthToken(token string) httpClient {
	return httpClient{
		client: http.DefaultClient,
		token:  token,
	}
}

func (c *httpClient) Get(url string, headers map[string]string) (resp *http.Response, err error) {
	return c.do("GET", url, headers)
}

func (c *httpClient) Head(url string, headers map[string]string) (resp *http.Response, err error) {
	return c.do("HEAD", url, headers)
}

func (c *httpClient) do(method, url string, headers map[string]string) (resp *http.Response, err error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+c.token)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return c.client.Do(req)
}
