terraform {
  required_providers {
    etcdv2 = {
      // Source code is github.com/retryW/terraform-provider-etcdv2
      source = "github.com/retryW/etcdv2"
    }
  }
}

provider "etcdv2" {
  host = "http://localhost:4001"
  // Leave out creds as my local instance has none, can use explicit or ENV
}

// Create a data source (read a value)
data "etcdv2_keyvalue" "foo" {
  key = "/root/test/foo"
}

// Create a resource (create a desired state resource)
resource "etcdv2_keyvalue" "test_kv_resource" {
  key   = "/root/cale/test_kv_resource"
  value = "myfirstetcdkv"
}

// Just output data for CLI in this case
output "foo_keyvalue" {
  value = data.etcdv2_keyvalue.foo
}

output "resource_keyvalue" {
  value = resource.etcdv2_keyvalue.test_kv_resource
}
