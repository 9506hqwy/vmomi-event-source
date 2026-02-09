package loki

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/klauspost/compress/snappy"
	"google.golang.org/protobuf/proto"

	"github.com/9506hqwy/vmomi-event-source/pkg/flag"
)

const (
	ContentTypeProtobuf = "application/x-protobuf"
	XScopeOrgID         = "X-Scope-OrgID"
)

func Post(ctx context.Context, message *Message) error {
	lokiURL, ok := ctx.Value(flag.LokiURLKey{}).(string)
	if !ok {
		return errors.New("url not found in context")
	}

	noVerifySSL, ok := ctx.Value(flag.LokiNoVerifySSLKey{}).(bool)
	if !ok {
		return errors.New("loki_no_verify_ssl not found in context")
	}

	tenantID, ok := ctx.Value(flag.LokiTenantIDKey{}).(string)
	if !ok {
		tenantID = ""
	}

	endpoint, err := url.Parse(lokiURL)
	if err != nil {
		return err
	}

	req, err := createRequest(ctx, endpoint, message, tenantID)
	if err != nil {
		return err
	}

	tlsConfig := tls.Config{InsecureSkipVerify: noVerifySSL}
	status, err := send(req, &tlsConfig)
	if err != nil {
		return err
	}

	//revive:disable:add-constant
	if (*status / 100) != 2 {
		return fmt.Errorf(
			"failed to post message to loki: status code %d message=%v",
			*status,
			message,
		)
	}
	//revive:enable:add-constant

	return nil
}

func createRequest(
	ctx context.Context,
	endpoint *url.URL,
	message *Message,
	tenantID string,
) (*http.Request, error) {
	buf, err := encode(message)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint.String(), bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", ContentTypeProtobuf)

	if tenantID != "" {
		req.Header.Set(XScopeOrgID, tenantID)
	}

	return req, nil
}

func encode(message *Message) ([]byte, error) {
	buf, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}

	buf = snappy.Encode(nil, buf)
	return buf, nil
}

func send(req *http.Request, tlsConfig *tls.Config) (*int, error) {
	client := http.Client{}

	client.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	return &res.StatusCode, nil
}
