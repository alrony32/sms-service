package validation

func ValidationMessages() map[string]string {
	return map[string]string{
		"required": ":field field is required",
		"email":    ":field must be a valid email address",
		"min":      ":field must be at least :param characters",
		"max":      ":field may not be greater than :param characters",
	}
}
