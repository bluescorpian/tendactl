package tenda

import (
	"io"
	"net/http"
	"net/url"
)

const BaseURL = "http://192.168.0.1/"

var ParsedBaseURL, _ = url.Parse(BaseURL)

func TendaRequest(method, relPath string, body io.Reader) (*http.Request, error) {
	path, err := JoinBaseURLWithPath(relPath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Host", "192.168.0.1")
	// req.Header.Set("Referer", "http://192.168.0.1/main.html")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("accept", "text/plain, *; q=0.01")
	req.Header.Set("accept-language", "en-US,en;q=0.9,af;q=0.8")
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	return req, nil
}

func CreateHTTPClient() *http.Client {
	jar, err := NewAuthJar()
	if err != nil {
		panic(err)
	}

	client := &http.Client{
		// CheckRedirect: func(req *http.Request, via []*http.Request) error {
		// 	return http.ErrUseLastResponse
		// },
		Jar: jar,
	}
	return client
}

func JoinBaseURLWithPath(path string) (string, error) {
	base, err := url.Parse(BaseURL)
	if err != nil {
		return "", err
	}

	ref, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	// ResolveReference combines the base URL with the path
	fullURL := base.ResolveReference(ref)
	return fullURL.String(), nil
}
