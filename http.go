package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type HttpRequestConfig struct {
	URL     string
	Headers map[string]string
	Timeout time.Duration
}

func HttpGet(config *HttpRequestConfig) (string, error) {
	var result string
	if config.Timeout == 0 {
		config.Timeout = 64 * time.Second
	}
	client := &http.Client{
		Timeout: config.Timeout,
	}
	req, err := http.NewRequest("GET", config.URL, nil)
	if err != nil {
		return result, fmt.Errorf("error creating request: %w", err)
	}
	for key, value := range config.Headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("error making GET request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("error reading response body: %w", err)
	}
	return string(body), err
}
