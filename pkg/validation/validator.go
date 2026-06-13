package validation

import (
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func ValidateJSON(c *gin.Context, obj any) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		return handleValidation(c, err, obj)
	}
	return true
}

func handleValidation(c *gin.Context, err error, obj any) bool {

	if errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Request body is required",
		})
		return false
	}

	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"message": "Validation failed",
			"errors":  formatValidationErrors(ve, obj),
		})
		return false
	}

	c.JSON(http.StatusBadRequest, gin.H{
		"message": err.Error(),
	})
	return false
}

func formatValidationErrors(errs validator.ValidationErrors, obj any) map[string]string {
	errorsMap := make(map[string]string)

	msgs := ValidationMessages()

	t := reflect.TypeOf(obj)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for _, e := range errs {

		field, ok := t.FieldByName(e.StructField())
		jsonKey := ""

		if ok {
			jsonKey = field.Tag.Get("json")
		}

		if jsonKey == "" {
			jsonKey = strings.ToLower(e.Field())
		} else {
			jsonKey = strings.Split(jsonKey, ",")[0]
		}

		template, exists := msgs[e.Tag()]
		if !exists {
			errorsMap[jsonKey] = "The " + jsonKey + " is invalid"
			continue
		}

		msg := strings.ReplaceAll(template, ":field", jsonKey)
		msg = strings.ReplaceAll(msg, ":param", e.Param())

		errorsMap[jsonKey] = msg
	}

	return errorsMap
}
