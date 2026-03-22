// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	endpoint := "http://localhost:3903"
	token := "test-token"

	client := NewClient(endpoint, token)

	if client.endpoint != endpoint {
		t.Errorf("Expected endpoint %s, got %s", endpoint, client.endpoint)
	}

	if client.token != token {
		t.Errorf("Expected token %s, got %s", token, client.token)
	}

	if client.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
}

func TestNewClient_trailingSlash(t *testing.T) {
	endpoint := "http://localhost:3903/"
	client := NewClient(endpoint, "token")

	expected := "http://localhost:3903"
	if client.endpoint != expected {
		t.Errorf("Expected endpoint %s, got %s", expected, client.endpoint)
	}
}

func TestListBuckets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method and path
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/v2/ListBuckets" {
			t.Errorf("Expected path /v2/ListBuckets, got %s", r.URL.Path)
		}

		// Check authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got %s", auth)
		}

		// Return mock response
		buckets := []Bucket{
			{
				ID:            "bucket-1",
				GlobalAliases: []string{"test-bucket-1"},
				WebsiteAccess: false,
			},
			{
				ID:            "bucket-2",
				GlobalAliases: []string{"test-bucket-2"},
				WebsiteAccess: true,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(buckets)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	buckets, err := client.ListBuckets(context.Background())

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(buckets) != 2 {
		t.Errorf("Expected 2 buckets, got %d", len(buckets))
	}

	if buckets[0].ID != "bucket-1" {
		t.Errorf("Expected bucket ID 'bucket-1', got %s", buckets[0].ID)
	}

	if buckets[1].WebsiteAccess != true {
		t.Error("Expected second bucket to have website access enabled")
	}
}

func TestGetBucketInfo_byID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/v2/GetBucketInfo" {
			t.Errorf("Expected path /v2/GetBucketInfo, got %s", r.URL.Path)
		}

		// Check query parameter
		bucketID := r.URL.Query().Get("id")
		if bucketID != "bucket-123" {
			t.Errorf("Expected bucket ID 'bucket-123' in query, got %s", bucketID)
		}

		bucket := Bucket{
			ID:            "bucket-123",
			GlobalAliases: []string{"my-bucket"},
			WebsiteAccess: false,
			Objects:       42,
			Bytes:         1024,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(bucket)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	bucketID := "bucket-123"
	bucket, err := client.GetBucketInfo(context.Background(), GetBucketInfoRequest{
		ID: &bucketID,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if bucket == nil {
		t.Fatal("Expected bucket to be returned")
	}

	if bucket.ID != "bucket-123" {
		t.Errorf("Expected bucket ID 'bucket-123', got %s", bucket.ID)
	}

	if bucket.Objects != 42 {
		t.Errorf("Expected 42 objects, got %d", bucket.Objects)
	}
}

func TestGetBucketInfo_notFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Bucket not found"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	bucketID := "nonexistent"
	bucket, err := client.GetBucketInfo(context.Background(), GetBucketInfoRequest{
		ID: &bucketID,
	})

	if err != nil {
		t.Fatalf("Expected no error for 404, got %v", err)
	}

	if bucket != nil {
		t.Error("Expected nil bucket for 404 response")
	}
}

func TestCreateBucket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v2/CreateBucket" {
			t.Errorf("Expected path /v2/CreateBucket, got %s", r.URL.Path)
		}

		// Verify request body
		var req CreateBucketRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if req.GlobalAlias == nil || *req.GlobalAlias != "new-bucket" {
			t.Errorf("Expected global alias 'new-bucket'")
		}

		// Return created bucket
		bucket := Bucket{
			ID:            "bucket-new-123",
			GlobalAliases: []string{"new-bucket"},
			WebsiteAccess: false,
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(bucket)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	alias := "new-bucket"
	bucket, err := client.CreateBucket(context.Background(), CreateBucketRequest{
		GlobalAlias: &alias,
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if bucket.ID != "bucket-new-123" {
		t.Errorf("Expected bucket ID 'bucket-new-123', got %s", bucket.ID)
	}

	if len(bucket.GlobalAliases) != 1 || bucket.GlobalAliases[0] != "new-bucket" {
		t.Errorf("Expected global alias 'new-bucket'")
	}
}

func TestUpdateBucket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v2/UpdateBucket" {
			t.Errorf("Expected path /v2/UpdateBucket, got %s", r.URL.Path)
		}

		// Check query parameter
		bucketID := r.URL.Query().Get("id")
		if bucketID != "bucket-123" {
			t.Errorf("Expected bucket ID 'bucket-123' in query, got %s", bucketID)
		}

		// Return updated bucket
		maxSize := int64(1073741824)
		bucket := Bucket{
			ID:            "bucket-123",
			GlobalAliases: []string{"my-bucket"},
			WebsiteAccess: false,
			Quotas: &BucketQuotas{
				MaxSize: &maxSize,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(bucket)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	maxSize := int64(1073741824)
	bucket, err := client.UpdateBucket(context.Background(), "bucket-123", UpdateBucketRequest{
		Quotas: &BucketQuotas{
			MaxSize: &maxSize,
		},
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if bucket.Quotas == nil || bucket.Quotas.MaxSize == nil {
		t.Fatal("Expected quotas to be set")
	}

	if *bucket.Quotas.MaxSize != 1073741824 {
		t.Errorf("Expected max size 1073741824, got %d", *bucket.Quotas.MaxSize)
	}
}

func TestDeleteBucket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v2/DeleteBucket" {
			t.Errorf("Expected path /v2/DeleteBucket, got %s", r.URL.Path)
		}

		// Check query parameter
		bucketID := r.URL.Query().Get("id")
		if bucketID != "bucket-123" {
			t.Errorf("Expected bucket ID 'bucket-123' in query, got %s", bucketID)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.DeleteBucket(context.Background(), DeleteBucketRequest{
		ID: "bucket-123",
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestAddBucketAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v2/AddBucketAlias" {
			t.Errorf("Expected path /v2/AddBucketAlias, got %s", r.URL.Path)
		}

		// Verify request body
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if req["id"] != "bucket-123" {
			t.Errorf("Expected bucket ID 'bucket-123', got %s", req["id"])
		}

		if req["alias"] != "new-alias" {
			t.Errorf("Expected alias 'new-alias', got %s", req["alias"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.AddBucketAlias(context.Background(), "bucket-123", "new-alias")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestRemoveBucketAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v2/RemoveBucketAlias" {
			t.Errorf("Expected path /v2/RemoveBucketAlias, got %s", r.URL.Path)
		}

		// Verify request body
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if req["id"] != "bucket-123" {
			t.Errorf("Expected bucket ID 'bucket-123', got %s", req["id"])
		}

		if req["alias"] != "old-alias" {
			t.Errorf("Expected alias 'old-alias', got %s", req["alias"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.RemoveBucketAlias(context.Background(), "bucket-123", "old-alias")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestClient_errorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")

	// Test ListBuckets error
	_, err := client.ListBuckets(context.Background())
	if err == nil {
		t.Error("Expected error for 500 response")
	}

	// Test GetBucketInfo error
	bucketID := "test"
	_, err = client.GetBucketInfo(context.Background(), GetBucketInfoRequest{ID: &bucketID})
	if err == nil {
		t.Error("Expected error for 500 response")
	}

	// Test CreateBucket error
	alias := "test"
	_, err = client.CreateBucket(context.Background(), CreateBucketRequest{GlobalAlias: &alias})
	if err == nil {
		t.Error("Expected error for 500 response")
	}

	// Test UpdateBucket error
	_, err = client.UpdateBucket(context.Background(), "test", UpdateBucketRequest{})
	if err == nil {
		t.Error("Expected error for 500 response")
	}

	// Test DeleteBucket error
	err = client.DeleteBucket(context.Background(), DeleteBucketRequest{ID: "test"})
	if err == nil {
		t.Error("Expected error for 500 response")
	}
}
