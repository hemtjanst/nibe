package nibe

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	user        string
	password    string
	endpoint    string
	fingerprint string
	serial      string

	hc *http.Client
}

func New(opts ...ClientOption) *Client {
	c := &Client{}
	for _, opt := range opts {
		opt(c)
	}

	if c.hc == nil {
		c.hc = safeHTTPClient(c.serial, c.fingerprint)
	}

	c.endpoint = strings.TrimSuffix(c.endpoint, "/")
	return c
}

// Device retrieves a single device.
//
// The ID can be either a [Device.Index] or the serial number.
func (c *Client) Device(ctx context.Context, id string) (Device, error) {
	u := c.endpoint + "/api/v1/devices/" + id
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.SetBasicAuth(c.user, c.password)

	resp, err := c.hc.Do(req)
	if err != nil {
		return Device{}, err
	}

	defer resp.Body.Close()

	var res Device
	if err := decode(resp.Body, &res, resp.StatusCode); err != nil {
		return res, err
	}

	return res, nil
}

// Devices retrieves all devices.
func (c *Client) Devices(ctx context.Context) ([]Device, error) {
	u := c.endpoint + "/api/v1/devices"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.SetBasicAuth(c.user, c.password)

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var res struct {
		Devices []Device `json:"devices"`
	}

	if err := decode(resp.Body, &res, resp.StatusCode); err != nil {
		return nil, err
	}

	return res.Devices, nil
}

// Points returns all data points.
//
// Data points can be settings or sensors.
func (c *Client) Points(ctx context.Context, id string) (map[string]Point, error) {
	u := c.endpoint + "/api/v1/devices/" + id + "/points"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.SetBasicAuth(c.user, c.password)

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	res := make(map[string]Point, 840)
	if err := decode(resp.Body, &res, resp.StatusCode); err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) PatchPoints(ctx context.Context, id string, values ...Value) (map[string]Point, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("need at least one value to update")
	}

	patchVals := make([]map[string]any, 0, len(values))
	for _, val := range values {
		patchVals = append(patchVals, val.patchRequest())
	}

	body, err := json.Marshal(patchVals)
	if err != nil {
		return nil, err
	}

	u := c.endpoint + "/api/v1/devices/" + id + "/points"
	req, _ := http.NewRequestWithContext(ctx, http.MethodPatch, u, bytes.NewReader(body))
	req.SetBasicAuth(c.user, c.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	res := make(map[string]Point, len(values))
	if err := decode(resp.Body, &res, resp.StatusCode); err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) Point(ctx context.Context, id string, point string) (Point, error) {
	u := c.endpoint + "/api/v1/devices/" + id + "/points/" + point
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.SetBasicAuth(c.user, c.password)

	resp, err := c.hc.Do(req)
	if err != nil {
		return Point{}, err
	}

	defer resp.Body.Close()

	var res Point
	if err := decode(resp.Body, &res, resp.StatusCode); err != nil {
		return res, err
	}

	return res, nil
}

func decode(r io.Reader, into any, code int) error {
	dec := json.NewDecoder(r)

	if code != http.StatusOK {
		apiErr := APIError{
			Code: code,
		}

		if err := dec.Decode(&apiErr); err != nil {
			apiErr.Message = err.Error()
			return &apiErr
		}

		if _, derr := dec.Token(); derr != io.EOF {
			apiErr.Message = strings.Join([]string{apiErr.Message, "trailing garbage in JSON"}, ", ")
			return &apiErr
		}

		apiErr.Code = code
		return &apiErr
	}

	err := dec.Decode(into)
	if err != nil {
		return err
	}

	if _, derr := dec.Token(); derr != io.EOF {
		return fmt.Errorf("trailing garbage in JSON")
	}

	return nil
}

type ClientOption func(*Client)

func WithEndpoint(s string) ClientOption {
	return func(c *Client) {
		c.endpoint = s
	}
}

func WithFingerprint(s string) ClientOption {
	return func(c *Client) {
		c.fingerprint = s
	}
}

func WithHTTPClient(h *http.Client) ClientOption {
	return func(c *Client) {
		c.hc = h
	}
}

func WithPassword(s string) ClientOption {
	return func(c *Client) {
		c.password = s
	}
}

func WithSerial(s string) ClientOption {
	return func(c *Client) {
		c.serial = s
	}
}

func WithUser(s string) ClientOption {
	return func(c *Client) {
		c.user = s
	}
}

func VerifyCert(serial, fingerprint string) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	expFp, err := hex.DecodeString(fingerprint)
	if err != nil {
		panic("invalid fingerprint hex")
	}

	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return errors.New("no certificates presented")
		}

		cert, err := x509.ParseCertificate(rawCerts[0])
		if err != nil {
			return fmt.Errorf("failed to parse certificate: %w", err)
		}

		if cert.Subject.CommonName != serial {
			return fmt.Errorf("subject CN mismatch: got %s", cert.Subject.CommonName)
		}

		if len(cert.Subject.Organization) == 0 || cert.Subject.Organization[0] != "NIBE" {
			return fmt.Errorf("subject Org mismatch: got %v", cert.Subject.Organization)
		}

		if cert.Issuer.CommonName != serial {
			return fmt.Errorf("issuer CN mismatch: got %s", cert.Issuer.CommonName)
		}

		if len(cert.Issuer.Organization) == 0 || cert.Issuer.Organization[0] != "NIBE" {
			return fmt.Errorf("issuer Org mismatch: got %v", cert.Issuer.Organization)
		}

		sum := sha256.Sum256(cert.Raw)
		if subtle.ConstantTimeCompare(sum[:], expFp) != 1 {
			return errors.New("certificate fingerprint mismatch")
		}

		return nil
	}
}

func safeHTTPClient(
	serial, fingerprint string,
) *http.Client {
	return &http.Client{
		Timeout: 10 * time.Minute,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Protocols: func() *http.Protocols {
				protos := &http.Protocols{}
				protos.SetHTTP1(true)
				return protos
			}(),
			MaxConnsPerHost:       5,
			MaxIdleConnsPerHost:   2,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			ReadBufferSize:        4 << 10,
			WriteBufferSize:       4 << 10,
			DialContext: (&net.Dialer{
				Timeout:       10 * time.Second,
				KeepAlive:     15 * time.Second,
				FallbackDelay: -1,
				Resolver: &net.Resolver{
					Dial: (&net.Dialer{
						Timeout:       50 * time.Millisecond,
						KeepAlive:     15 * time.Second,
						FallbackDelay: 30 * time.Millisecond,
					}).DialContext,
				},
			}).DialContext,
			TLSClientConfig: &tls.Config{
				NextProtos:            []string{"http/1.1"},
				InsecureSkipVerify:    true,
				VerifyPeerCertificate: VerifyCert(serial, fingerprint),
			},
		},
	}
}
