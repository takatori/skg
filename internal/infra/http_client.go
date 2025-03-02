package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/morikuni/failure/v2"
	"github.com/takatori/skg/internal/errors"
)

type HttpClient struct {
	Client *http.Client
}

type Request struct {
	Url     string
	Headers map[string]string
	Cookies []http.Cookie
	IsTrace bool
}

type PostRequest struct {
	Request
	Entity any
}

func NewHttpClient() *HttpClient {

	dt := http.DefaultTransport
	transport := dt.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 10
	transport.IdleConnTimeout = time.Duration(30) * time.Second
	transport.MaxIdleConns = transport.MaxIdleConnsPerHost * 2
	return &HttpClient{
		Client: &http.Client{
			Transport: transport,
			Timeout:   time.Duration(1) * time.Second,
		},
	}
}

func (c *HttpClient) Get(ctx context.Context, req Request, expected any) error {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, req.Url, nil)
	if err != nil {
		return failure.Translate(
			err,
			errors.ErrInternal,
			failure.Field(failure.Message("failed to create request")),
			failure.Context{
				"url": req.Url,
			},
		)
	}
	for k, v := range req.Headers {
		if v != "" {
			r.Header.Set(k, v)
		}
	}
	for _, cookie := range req.Cookies {
		if len(cookie.Value) > 0 {
			r.AddCookie(&cookie)
		}
	}

	res, err := c.Client.Do(r)
	if err != nil {
		return failure.Translate(
			err,
			errors.ErrInternal,
			failure.Field(failure.Message("failed to send request")),
			failure.Context{
				"url": req.Url,
			},
		)
	}
	defer res.Body.Close() // TODO: error handling

	if res.StatusCode != http.StatusOK {
		return failure.New(
			errors.ErrInternal,
			failure.Field(failure.Message("unexpected status code")),
			failure.Context{
				"url":  req.Url,
				"code": fmt.Sprintf("%d", res.StatusCode),
			},
		)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return failure.Translate(
			err,
			errors.ErrInternal,
			failure.Field(failure.Message("failed to read response body")),
			failure.Context{
				"url": req.Url,
			},
		)
	}

	if err := json.Unmarshal(body, expected); err != nil {
		return failure.Translate(
			err,
			errors.ErrInternal,
			failure.Field(failure.Message("failed to decode response body")),
			failure.Context{
				"url": req.Url,
			},
		)
	}

	return nil
}

func (c *HttpClient) Post(ctx context.Context, req PostRequest, expected any) error {
	encoded, err := json.Marshal(req.Entity)
	if err != nil {
		return failure.Translate(
			err,
			errors.ErrInternal,
			failure.Field(failure.Message("failed to encode request entity")),
			failure.Context{
				"url":    req.Url,
				"entity": fmt.Sprintf("%+v", req.Entity),
			},
		)
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, req.Url, bytes.NewBuffer(encoded))
	if err != nil {
		return failure.Translate(
			err,
			errors.ErrInternal,
			failure.Field(failure.Message("failed to create request")),
			failure.Context{
				"url": req.Url,
				"req": string(encoded),
			},
		)
	}
	for k, v := range req.Headers {
		if v != "" {
			r.Header.Set(k, v)
		}
	}
	for _, cookie := range req.Cookies {
		if len(cookie.Value) > 0 {
			r.AddCookie(&cookie)
		}
	}
	r.Header.Set("Content-Type", "application/json")

	res, err := c.Client.Do(r)
	if err != nil {
		return failure.Translate(
			err,
			errors.ErrInternal,
			failure.Field(failure.Message("failed to send request")),
			failure.Context{
				"url": req.Url,
				"req": string(encoded),
			},
		)
	}
	defer res.Body.Close() // TODO: error handling

	if res.StatusCode != http.StatusOK {
		return failure.New(
			errors.ErrInternal,
			failure.Field(failure.Message("unexpected status code")),
			failure.Context{
				"url":  req.Url,
				"req":  string(encoded),
				"code": fmt.Sprintf("%d", res.StatusCode),
			},
		)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return failure.Translate(
			err,
			errors.ErrInternal,
			failure.Field(failure.Message("failed to read response body")),
			failure.Context{
				"url": req.Url,
				"req": string(encoded),
			},
		)
	}

	if err := json.Unmarshal(body, expected); err != nil {
		return failure.Translate(
			err,
			errors.ErrInternal,
			failure.Field(failure.Message("failed to decode response body")),
			failure.Context{
				"url": req.Url,
				"req": string(encoded),
			},
		)
	}

	return nil

}
