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
- [Go](https://go.dev/doc/install) >= 1.25.8 (for development)
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

These can be configured in three ways. Values set in the provider block take precedence. Environment variables are used only when the corresponding provider attribute is omitted; explicit empty or unknown values are errors.

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

#### 3. Mixed approach (environment variable fallback):

```hcl
provider "garage" {
  endpoint = "http://localhost:3903"
  # token will be read from GARAGE_TOKEN environment variable
}
```

### Resources and Data Sources

Generated documentation is the authoritative schema reference:

- [`garage_bucket` resource](./docs/resources/bucket.md)
- [`garage_key` resource](./docs/resources/key.md)
- [`garage_bucket_permission` resource](./docs/resources/bucket_permission.md)
- [`garage_bucket` data source](./docs/data-sources/bucket.md)

Changing an access key `name` replaces the key. Bucket permission imports must use exactly `bucket_id/access_key_id`.

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

Fast local provider tests do not require Garage credentials. They cover provider configuration precedence and diagnostics, key replacement schema behavior, bucket recovery and refresh state behavior, bucket permission import validation, and bucket data-source selector validation.

To run the full suite of acceptance tests, you'll need a running Garage instance with Admin API v2 enabled.

Set up your acceptance-test environment:

```bash
export TF_ACC=1
export GARAGE_ENDPOINT="http://localhost:3903"
export GARAGE_TOKEN="your-test-admin-token"
```

If `TF_ACC` is unset, acceptance tests are skipped. When `TF_ACC=1`, both `GARAGE_ENDPOINT` and `GARAGE_TOKEN` must be set.

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
