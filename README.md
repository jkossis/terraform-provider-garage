# Terraform Provider for Garage

This is a Terraform provider for managing [Garage](https://garagehq.deuxfleurs.fr/) S3 buckets via the Garage Admin API.

Garage is an S3-compatible distributed object storage service designed for self-hosting at a small-to-medium scale. This provider allows you to manage Garage buckets declaratively using Terraform.

## Table of Contents

- [Requirements](#requirements)
- [Quick Start](#quick-start)
- [Using the Provider](#using-the-provider)
- [Resources](#resources)
- [Data Sources](#data-sources)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)
- [Developing the Provider](#developing-the-provider)
- [About Garage](#about-garage)

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (for development)
- [Garage](https://garagehq.deuxfleurs.fr/) >= 0.9.0 with Admin API v2 enabled

## Quick Start

### 1. Build the Provider

```bash
# Clone the repository
git clone <your-repo-url>
cd terraform-provider-garage

# Build the provider
go build -o terraform-provider-garage
```

### 2. Install the Provider Locally

For local development, use Terraform's [development overrides](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers).

Create or edit `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "jkossis/garage" = "/path/to/terraform-provider-garage"
  }

  direct {}
}
```

Replace `/path/to/terraform-provider-garage` with the actual path to your repository directory.

### 3. Configure Your Environment

Set your Garage endpoint and token:

```bash
export GARAGE_ENDPOINT="http://localhost:3903"
export GARAGE_TOKEN="your-admin-token-here"
```

### 4. Create Your First Configuration

Create a file named `main.tf`:

```hcl
terraform {
  required_providers {
    garage = {
      source = "jkossis/garage"
    }
  }
}

provider "garage" {
  # endpoint and token will be read from environment variables
}

resource "garage_bucket" "example" {
  global_alias = "my-first-bucket"
}

output "bucket_id" {
  value = garage_bucket.example.id
}
```

### 5. Run Terraform

```bash
# Initialize Terraform
terraform init

# Plan the changes
terraform plan

# Apply the configuration
terraform apply

# When you're done, destroy the resources
terraform destroy
```

## Using the Provider

### Configuration

The provider requires two configuration values:

- `endpoint` - The URL of your Garage Admin API endpoint (default port: 3903)
- `token` - Your Garage admin API bearer token

These can be configured in three ways:

#### 1. In the provider block:

```hcl
provider "garage" {
  endpoint = "http://localhost:3903"
  token    = "your-admin-token-here"
}
```

#### 2. Via environment variables:

```bash
export GARAGE_ENDPOINT="http://localhost:3903"
export GARAGE_TOKEN="your-admin-token-here"
```

#### 3. Mixed approach (environment variables override provider config):

```hcl
provider "garage" {
  endpoint = "http://localhost:3903"
  # token will be read from GARAGE_TOKEN environment variable
}
```

### Resources

#### `garage_bucket`

Manages a Garage S3 bucket.

**Example Usage:**

```hcl
# Basic bucket
resource "garage_bucket" "example" {
  global_alias = "my-bucket"
}

# Bucket with website hosting
resource "garage_bucket" "website" {
  global_alias             = "my-website"
  website_enabled          = true
  website_index_document   = "index.html"
  website_error_document   = "error.html"
}

# Bucket with quotas
resource "garage_bucket" "limited" {
  global_alias = "limited-bucket"
  max_size     = 1073741824  # 1 GB in bytes
  max_objects  = 10000
}
```

**Schema:**

- `global_alias` (Required, String) - The global alias (name) for the bucket. Changing this forces a new resource.
- `website_enabled` (Optional, Bool) - Enable website hosting for this bucket. Default: `false`
- `website_index_document` (Optional, String) - The index document for website hosting (e.g., 'index.html')
- `website_error_document` (Optional, String) - The error document for website hosting (e.g., 'error.html')
- `max_size` (Optional, Int64) - Maximum size of the bucket in bytes. Leave unset for unlimited.
- `max_objects` (Optional, Int64) - Maximum number of objects in the bucket. Leave unset for unlimited.

**Computed Attributes:**

- `id` (String) - The unique identifier of the bucket

#### `garage_key`

Manages a Garage access key for S3 API authentication.

**Example Usage:**

```hcl
# Create an access key
resource "garage_key" "app" {
  name = "my-application"
}

# Output credentials (use caution with secrets!)
output "access_key_id" {
  value = garage_key.app.id
}

output "secret_access_key" {
  value     = garage_key.app.secret_access_key
  sensitive = true
}
```

**Schema:**

- `name` (Optional, String) - A human-friendly name for the access key

**Computed Attributes:**

- `id` (String) - The access key ID
- `secret_access_key` (String, Sensitive) - The secret access key (only available on creation)

**Note:** The secret access key is only returned when the key is created. It's not available via the API after creation, so it won't be populated when importing an existing key.

#### `garage_bucket_permission`

Manages permissions for an access key on a bucket.

**Example Usage:**

```hcl
# Create bucket and key
resource "garage_bucket" "data" {
  global_alias = "app-data"
}

resource "garage_key" "app" {
  name = "application-key"
}

# Grant read/write permissions
resource "garage_bucket_permission" "app_access" {
  bucket_id     = garage_bucket.data.id
  access_key_id = garage_key.app.id
  read          = true
  write         = true
  owner         = false
}

# Read-only access for another key
resource "garage_key" "readonly" {
  name = "readonly-key"
}

resource "garage_bucket_permission" "readonly_access" {
  bucket_id     = garage_bucket.data.id
  access_key_id = garage_key.readonly.id
  read          = true
  write         = false
  owner         = false
}
```

**Schema:**

- `bucket_id` (Required, String) - The ID of the bucket. Changing this forces a new resource.
- `access_key_id` (Required, String) - The ID of the access key. Changing this forces a new resource.
- `read` (Optional, Bool) - Grant read permission. Default: `false`
- `write` (Optional, Bool) - Grant write permission. Default: `false`
- `owner` (Optional, Bool) - Grant owner permission. Default: `false`

**Computed Attributes:**

- `id` (String) - The unique identifier (format: `bucket_id/access_key_id`)

**Permission Types:**
- **Read**: List objects, download objects, read metadata
- **Write**: Upload objects, delete objects, modify metadata
- **Owner**: All read/write operations plus bucket management and permission grants

### Data Sources

#### `garage_bucket`

Retrieves information about an existing Garage bucket.

**Example Usage:**

```hcl
# Look up bucket by global alias
data "garage_bucket" "example" {
  global_alias = "my-bucket"
}

# Look up bucket by ID
data "garage_bucket" "by_id" {
  id = "8d7c3c6e-7b9d-4c3a-9f2e-1a5b6c7d8e9f"
}

# Use data source output
output "bucket_info" {
  value = {
    id      = data.garage_bucket.example.id
    objects = data.garage_bucket.example.objects
    bytes   = data.garage_bucket.example.bytes
  }
}
```

**Schema:**

Either `id` or `global_alias` must be specified.

- `id` (Optional, String) - The unique identifier of the bucket
- `global_alias` (Optional, String) - The primary global alias (name) of the bucket

**Computed Attributes:**

- `id` (String) - The unique identifier of the bucket
- `global_alias` (String) - The primary global alias of the bucket
- `global_aliases` (List of String) - All global aliases for this bucket
- `website_enabled` (Bool) - Whether website hosting is enabled
- `website_index_document` (String) - The index document for website hosting
- `website_error_document` (String) - The error document for website hosting
- `max_size` (Int64) - Maximum size of the bucket in bytes
- `max_objects` (Int64) - Maximum number of objects in the bucket
- `objects` (Int64) - Current number of objects in the bucket
- `bytes` (Int64) - Current size of the bucket in bytes
- `unfinished_uploads` (Int64) - Number of unfinished multipart uploads

## Examples

Check out the [examples](./examples/) directory for more configurations:

- [Basic Provider Configuration](./examples/provider/provider.tf)
- [Bucket Resource Examples](./examples/resources/garage_bucket/resource.tf)
- [Access Key Resource Examples](./examples/resources/garage_key/resource.tf)
- [Bucket Permission Resource Examples](./examples/resources/garage_bucket_permission/resource.tf)
- [Bucket Data Source Examples](./examples/data-sources/garage_bucket/data-source.tf)

## Troubleshooting

### "Provider not found" error

If you see an error like "provider registry.terraform.io/jkossis/garage not found", make sure:

1. You've set up the dev overrides in `~/.terraformrc` correctly
2. The path in dev overrides points to the directory containing the built binary
3. You've built the provider binary (`go build -o terraform-provider-garage`)

### Connection errors

If you see connection errors:

1. Verify your Garage instance is running and accessible
2. Check that the endpoint URL is correct (including protocol and port)
3. Verify your admin token is valid
4. Check Garage logs for any API errors

### API version mismatch

This provider requires Garage Admin API v2. If you're using an older version of Garage:

1. Upgrade to Garage >= 0.9.0
2. Update your Garage configuration to enable API v2
3. Regenerate your admin tokens if needed

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

### Testing

To run the full suite of acceptance tests, you'll need a running Garage instance with Admin API v2 enabled.

Set up your test environment:

```bash
export GARAGE_ENDPOINT="http://localhost:3903"
export GARAGE_TOKEN="your-test-admin-token"
```

Then run the tests:

```bash
make testacc
```

## License

This provider is published under the MPL-2.0 license.

## About Garage

[Garage](https://garagehq.deuxfleurs.fr/) is an S3-compatible distributed object storage service designed for self-hosting at a small-to-medium scale. It's lightweight, easy to operate, and supports geo-distributed deployments.

For more information about Garage:
- [Garage Documentation](https://garagehq.deuxfleurs.fr/documentation/)
- [Garage Admin API Reference](https://garagehq.deuxfleurs.fr/documentation/reference-manual/admin-api/)
- [Garage GitHub Repository](https://github.com/deuxfleurs-org/garage)
