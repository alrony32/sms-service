package validator

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

func Init() {
	Validate = validator.New()

	Validate.RegisterValidation("e164_bd", ValidateBDPhone)

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("e164_bd", ValidateBDPhone)
	}
}
