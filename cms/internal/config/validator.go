package config

import "github.com/go-playground/validator"

func NewValidator(c *OmedConfig) *validator.Validate {
	return validator.New()
}
