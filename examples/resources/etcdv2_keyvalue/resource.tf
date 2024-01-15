resource "etcdv2_keyvalue" "hello_world" {
  key   = "/root/hello"
  value = "world"
}
