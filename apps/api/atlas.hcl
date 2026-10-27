variable "db_url" {
  type    = string
  default = getenv("OMED_DB_URL")
}

variable "dev_url" {
  type    = string
  default = getenv("OMED_DB_DEV_URL")
}

locals {
  resolved_db_url  = var.db_url != "" ? var.db_url : "postgres://omed:omed@localhost:5432/omed?sslmode=disable"
  resolved_dev_url = var.dev_url != "" ? var.dev_url : replace(local.resolved_db_url, "/omed?", "/omed_dev?")
}

env "local" {
  url     = local.resolved_db_url
  dev     = local.resolved_dev_url
  schemas = ["core", "fin", "cms"]
  exclude = ["public", "public.*"]
  migration {
    dir = "file://migrations"
  }
}

