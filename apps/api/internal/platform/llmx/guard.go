package llmx

import "net/http"

func newHTTPClient(allowPrivate bool) *http.Client {
	return http.DefaultClient
}
