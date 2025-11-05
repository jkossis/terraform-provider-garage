// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client is a Garage API client.
type Client struct {
	endpoint   string
	token      string
	httpClient *http.Client
}

// NewClient creates a new Garage API client.
func NewClient(endpoint, token string) *Client {
	return &Client{
		endpoint:   strings.TrimSuffix(endpoint, "/"),
		token:      token,
		httpClient: http.DefaultClient,
	}
}

// Bucket represents a Garage bucket.
type Bucket struct {
	ID                string          `json:"id"`
	GlobalAliases     []string        `json:"globalAliases"`
	WebsiteAccess     bool            `json:"websiteAccess"`
	WebsiteConfig     *WebsiteConfig  `json:"websiteConfig,omitempty"`
	Keys              []BucketKeyInfo `json:"keys"`
	Objects           int64           `json:"objects,omitempty"`
	Bytes             int64           `json:"bytes,omitempty"`
	UnfinishedUploads int64           `json:"unfinishedUploads,omitempty"`
	Quotas            *BucketQuotas   `json:"quotas,omitempty"`
}

// WebsiteConfig represents website configuration for a bucket.
type WebsiteConfig struct {
	IndexDocument string `json:"indexDocument"`
	ErrorDocument string `json:"errorDocument"`
}

// BucketKeyInfo represents key permissions on a bucket.
type BucketKeyInfo struct {
	AccessKeyID string      `json:"accessKeyId"`
	Name        string      `json:"name"`
	Permissions Permissions `json:"permissions"`
}

// Permissions represents the permissions a key has on a bucket.
type Permissions struct {
	Read  bool `json:"read"`
	Write bool `json:"write"`
	Owner bool `json:"owner"`
}

// BucketQuotas represents quotas for a bucket.
type BucketQuotas struct {
	MaxSize    *int64 `json:"maxSize,omitempty"`
	MaxObjects *int64 `json:"maxObjects,omitempty"`
}

// CreateBucketRequest represents the request to create a bucket.
type CreateBucketRequest struct {
	GlobalAlias *string `json:"globalAlias,omitempty"`
	LocalAlias  *struct {
		AccessKeyID string `json:"accessKeyId"`
		Alias       string `json:"alias"`
	} `json:"localAlias,omitempty"`
}

// UpdateBucketRequest represents the request to update a bucket.
type UpdateBucketRequest struct {
	WebsiteAccess *struct {
		Enabled       bool    `json:"enabled"`
		IndexDocument *string `json:"indexDocument,omitempty"`
		ErrorDocument *string `json:"errorDocument,omitempty"`
	} `json:"websiteAccess,omitempty"`
	Quotas *BucketQuotas `json:"quotas,omitempty"`
}

// DeleteBucketRequest represents the request to delete a bucket.
type DeleteBucketRequest struct {
	ID string `json:"id"`
}

// GetBucketInfoRequest represents the request to get bucket info.
type GetBucketInfoRequest struct {
	ID          *string `json:"id,omitempty"`
	GlobalAlias *string `json:"globalAlias,omitempty"`
}

// BucketKeyPermRequest represents the request to allow or deny bucket key permissions.
type BucketKeyPermRequest struct {
	BucketID    string      `json:"bucketId"`
	AccessKeyID string      `json:"accessKeyId"`
	Permissions Permissions `json:"permissions"`
}

// AccessKey represents a Garage access key.
type AccessKey struct {
	AccessKeyID     string          `json:"accessKeyId"`
	Name            string          `json:"name"`
	Expired         bool            `json:"expired"`
	Created         *string         `json:"created,omitempty"`
	Expiration      *string         `json:"expiration,omitempty"`
	SecretAccessKey *string         `json:"secretAccessKey,omitempty"`
	Permissions     KeyPermissions  `json:"permissions"`
	Buckets         []KeyBucketInfo `json:"buckets"`
}

// KeyPermissions represents the permissions a key has.
type KeyPermissions struct {
	CreateBucket bool `json:"createBucket"`
}

// KeyBucketInfo represents bucket information associated with a key.
type KeyBucketInfo struct {
	ID            string      `json:"id"`
	GlobalAliases []string    `json:"globalAliases"`
	LocalAliases  []string    `json:"localAliases"`
	Permissions   Permissions `json:"permissions"`
}

// CreateKeyRequest represents the request to create an access key.
type CreateKeyRequest struct {
	Name       *string `json:"name,omitempty"`
	Expiration *string `json:"expiration,omitempty"`
}

// DeleteKeyRequest represents the request to delete an access key.
type DeleteKeyRequest struct {
	ID string `json:"id"`
}

// GetKeyInfoRequest represents the request to get key info.
type GetKeyInfoRequest struct {
	ID string `json:"id"`
}

// doRequest makes an HTTP request to the Garage API.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

// ListBuckets lists all buckets.
func (c *Client) ListBuckets(ctx context.Context) ([]Bucket, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/v2/ListBuckets", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var buckets []Bucket
	if err := json.NewDecoder(resp.Body).Decode(&buckets); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return buckets, nil
}

// GetBucketInfo gets information about a specific bucket.
func (c *Client) GetBucketInfo(ctx context.Context, req GetBucketInfoRequest) (*Bucket, error) {
	// Build query parameters
	path := "/v2/GetBucketInfo?"
	if req.ID != nil {
		path += "id=" + *req.ID
	} else if req.GlobalAlias != nil {
		path += "globalAlias=" + *req.GlobalAlias
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var bucket Bucket
	if err := json.NewDecoder(resp.Body).Decode(&bucket); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &bucket, nil
}

// CreateBucket creates a new bucket.
func (c *Client) CreateBucket(ctx context.Context, req CreateBucketRequest) (*Bucket, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/v2/CreateBucket", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var bucket Bucket
	if err := json.NewDecoder(resp.Body).Decode(&bucket); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &bucket, nil
}

// UpdateBucket updates an existing bucket.
func (c *Client) UpdateBucket(ctx context.Context, bucketID string, req UpdateBucketRequest) (*Bucket, error) {
	// The UpdateBucket endpoint requires the bucket ID as a query parameter
	path := fmt.Sprintf("/v2/UpdateBucket?id=%s", bucketID)

	resp, err := c.doRequest(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var bucket Bucket
	if err := json.NewDecoder(resp.Body).Decode(&bucket); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &bucket, nil
}

// DeleteBucket deletes a bucket.
func (c *Client) DeleteBucket(ctx context.Context, req DeleteBucketRequest) error {
	// Build query parameters
	path := fmt.Sprintf("/v2/DeleteBucket?id=%s", req.ID)

	resp, err := c.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// AddBucketAlias adds a global alias to a bucket.
func (c *Client) AddBucketAlias(ctx context.Context, bucketID, alias string) error {
	req := map[string]string{
		"id":    bucketID,
		"alias": alias,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/v2/AddBucketAlias", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// RemoveBucketAlias removes a global alias from a bucket.
func (c *Client) RemoveBucketAlias(ctx context.Context, bucketID, alias string) error {
	req := map[string]string{
		"id":    bucketID,
		"alias": alias,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/v2/RemoveBucketAlias", req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// AllowBucketKey grants permissions for an access key on a bucket.
func (c *Client) AllowBucketKey(ctx context.Context, req BucketKeyPermRequest) (*Bucket, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/v2/AllowBucketKey", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var bucket Bucket
	if err := json.NewDecoder(resp.Body).Decode(&bucket); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &bucket, nil
}

// DenyBucketKey revokes permissions for an access key on a bucket.
func (c *Client) DenyBucketKey(ctx context.Context, req BucketKeyPermRequest) (*Bucket, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/v2/DenyBucketKey", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var bucket Bucket
	if err := json.NewDecoder(resp.Body).Decode(&bucket); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &bucket, nil
}

// CreateKey creates a new access key.
func (c *Client) CreateKey(ctx context.Context, req CreateKeyRequest) (*AccessKey, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/v2/CreateKey", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var key AccessKey
	if err := json.NewDecoder(resp.Body).Decode(&key); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &key, nil
}

// GetKeyInfo gets information about a specific access key.
func (c *Client) GetKeyInfo(ctx context.Context, req GetKeyInfoRequest) (*AccessKey, error) {
	path := fmt.Sprintf("/v2/GetKeyInfo?id=%s", req.ID)

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var key AccessKey
	if err := json.NewDecoder(resp.Body).Decode(&key); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &key, nil
}

// DeleteKey deletes an access key.
func (c *Client) DeleteKey(ctx context.Context, req DeleteKeyRequest) error {
	path := fmt.Sprintf("/v2/DeleteKey?id=%s", req.ID)

	resp, err := c.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
