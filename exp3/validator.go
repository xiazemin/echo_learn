package main

import (
	"sync"

	"github.com/go-playground/validator/v10"
)

type CustomValidator struct {
	once     sync.Once
	validate *validator.Validate
}

func (c *CustomValidator) Validate(i interface{}) error {
	c.lazyInit()
	return c.validate.Struct(i)
}

func (c *CustomValidator) lazyInit() {
	c.once.Do(func() {
		c.validate = validator.New()
	})
}

func NewCustomValidator() *CustomValidator {
	return &CustomValidator{}
}
