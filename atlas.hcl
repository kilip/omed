data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "./cmd/atlas",
  ]
}
env "local" {
  src = data.external_schema.gorm.url
  dev = "docker://postgres/18beta3-alpine/dev"
  migration {
    dir = "file://migrations"
    format = golang-migrate
  }
  format {
    migrate {
      diff = "{{ sql . \" \" }}"
    }
  }
}
