resource "etcdv2_user" "developer" {
  username = "dev_user"
  password = var.dev_password
  roles = [
    etcdv2_role.app_readonly.name,
    etcdv2_role.monitoring.name
  ]
}
