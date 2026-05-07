variable "evict_surveillance_schedule" {
  type        = string
  description = <<-EOT
  When the surveillance cleanup job shall be performed; expressed in cron format (https://crontab.guru/).

  EOT

  default = "*/30 * * * *"
}
