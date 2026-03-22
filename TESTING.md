# Testing Guide

This document describes the testing strategy and available tests for the Terraform Provider for Garage.

## Test Coverage

The provider has comprehensive test coverage including:

1. **Unit Tests** - Test individual components in isolation
2. **Acceptance Tests** - Test the provider against a real Garage instance

## Unit Tests

### Client Tests

Located in `internal/client/client_test.go`, these tests verify the Garage API client functionality using mock HTTP servers.

**Coverage:**
- Client initialization
- Endpoint trailing slash handling
- List buckets
- Get bucket info (by ID and alias)
- Create bucket
- Update bucket
- Delete bucket
- Add bucket alias
- Remove bucket alias
- Error handling

**Run unit tests:**

```bash
go test ./internal/client -v
```

**Example output:**
```
=== RUN   TestNewClient
--- PASS: TestNewClient (0.00s)
=== RUN   TestListBuckets
--- PASS: TestListBuckets (0.00s)
...
PASS
ok      github.com/jkossis/terraform-provider-garage/garage/client       0.221s
```

## Acceptance Tests

Acceptance tests require a running Garage instance with Admin API v2 enabled.

### Setup

Before running acceptance tests, you must:

1. Have a running Garage instance (>= v0.9.0) with Admin API v2 enabled
2. Set the required environment variables:

```bash
export GARAGE_ENDPOINT="http://localhost:3903"
export GARAGE_TOKEN="your-admin-token"
export TF_ACC=1  # Required to enable acceptance tests
```

### Bucket Resource Tests

Located in `internal/provider/bucket_resource_test.go`

**Test cases:**
- `TestAccBucketResource_basic` - Basic bucket creation and deletion
- `TestAccBucketResource_website` - Website hosting configuration (enable/disable, update documents)
- `TestAccBucketResource_quotas` - Bucket quotas (size and object limits)
- `TestAccBucketResource_full` - All features combined
- `TestAccBucketResource_nameChange` - Global alias change (forces replacement)

**Run resource tests:**

```bash
TF_ACC=1 go test ./internal/provider -v -run="TestAccBucketResource"
```

### Bucket Data Source Tests

Located in `internal/provider/bucket_data_source_test.go`

**Test cases:**
- `TestAccBucketDataSource_byAlias` - Look up bucket by global alias
- `TestAccBucketDataSource_byID` - Look up bucket by ID
- `TestAccBucketDataSource_withWebsite` - Read website configuration
- `TestAccBucketDataSource_withQuotas` - Read quota configuration
- `TestAccBucketDataSource_full` - Read all bucket attributes
- `TestAccBucketDataSource_multipleAliases` - Handle multiple global aliases

**Run data source tests:**

```bash
TF_ACC=1 go test ./internal/provider -v -run="TestAccBucketDataSource"
```

## Running All Tests

### Run all unit tests:

```bash
go test ./... -v
```

### Run all acceptance tests:

```bash
# Set up environment
export GARAGE_ENDPOINT="http://localhost:3903"
export GARAGE_TOKEN="your-admin-token"
export TF_ACC=1

# Run tests
go test ./internal/provider -v -timeout 30m
```

### Run specific test:

```bash
TF_ACC=1 go test ./internal/provider -v -run="TestAccBucketResource_website"
```

## Test Configuration

Acceptance tests use the `testAccProtoV6ProviderFactories` from `internal/provider/provider_test.go` to instantiate the provider.

The `testAccPreCheck` function verifies that required environment variables are set before running tests.

## Continuous Integration

For CI/CD pipelines, you can use the following command to run all tests:

```bash
# Unit tests (no Garage instance required)
go test ./internal/client -v -cover

# Acceptance tests (requires Garage instance)
export GARAGE_ENDPOINT="http://garage:3903"
export GARAGE_TOKEN="${GARAGE_ADMIN_TOKEN}"
export TF_ACC=1
go test ./internal/provider -v -timeout 30m
```

## Test Patterns

### Acceptance Test Pattern

Each acceptance test follows this pattern:

1. **PreCheck** - Verify environment is configured
2. **Create** - Apply Terraform configuration
3. **Check** - Verify resource attributes
4. **Import** - Test state import (optional)
5. **Update** - Apply updated configuration (optional)
6. **Check** - Verify updated attributes (optional)
7. **Delete** - Destroy resource (automatic)

### Example:

```go
resource.Test(t, resource.TestCase{
    PreCheck:                 func() { testAccPreCheck(t) },
    ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
    Steps: []resource.TestStep{
        // Create and Read
        {
            Config: testAccBucketResourceConfig_basic("my-bucket"),
            Check: resource.ComposeAggregateTestCheckFunc(
                resource.TestCheckResourceAttr("garage_bucket.test", "global_alias", "my-bucket"),
                resource.TestCheckResourceAttrSet("garage_bucket.test", "id"),
            ),
        },
        // ImportState
        {
            ResourceName:      "garage_bucket.test",
            ImportState:       true,
            ImportStateVerify: true,
        },
    },
})
```

## Coverage Report

To generate a coverage report:

```bash
# Unit tests only
go test ./internal/client -coverprofile=coverage_client.out
go tool cover -html=coverage_client.out

# All tests (requires Garage instance)
export TF_ACC=1
go test ./internal/provider -coverprofile=coverage_provider.out
go tool cover -html=coverage_provider.out
```

## Troubleshooting Tests

### Acceptance tests fail with "GARAGE_ENDPOINT must be set"

Make sure you've exported the required environment variables:

```bash
export GARAGE_ENDPOINT="http://localhost:3903"
export GARAGE_TOKEN="your-admin-token"
export TF_ACC=1
```

### Tests time out

Increase the timeout:

```bash
TF_ACC=1 go test ./internal/provider -v -timeout 60m
```

### Connection refused errors

Verify your Garage instance is running and accessible:

```bash
curl -H "Authorization: Bearer $GARAGE_TOKEN" $GARAGE_ENDPOINT/health
```

## Contributing

When adding new features:

1. Write unit tests for new client methods
2. Write acceptance tests for new resources/data sources
3. Ensure all tests pass before submitting PR
4. Update this document with new test coverage
