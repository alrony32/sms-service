package validator

import (
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	bdPhoneRe     = regexp.MustCompile(`^(01[3-9]\d{8}|8801[3-9]\d{8}|\+8801[3-9]\d{8})$`)
	localRe       = regexp.MustCompile(`^01[3-9]\d{8}$`)
	with88Re      = regexp.MustCompile(`^8801[3-9]\d{8}$`)
	withPlus88Re  = regexp.MustCompile(`^\+8801[3-9]\d{8}$`)
	phoneStripper = strings.NewReplacer(" ", "", "-", "")
)

func ValidateBDPhone(fl validator.FieldLevel) bool {
	return bdPhoneRe.MatchString(fl.Field().String())
}

func NormalizeBDPhone(phone string) string {
	phone = phoneStripper.Replace(strings.TrimSpace(phone))

	switch {
	case localRe.MatchString(phone):
		return "+88" + phone
	case with88Re.MatchString(phone):
		return "+" + phone
	case withPlus88Re.MatchString(phone):
		return phone
	default:
		return ""
	}
}
