package configuration

import (
	"fmt"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	entranslations "github.com/go-playground/validator/v10/translations/en"
	"regexp"
	"strings"
)

var (
	validate = validator.New(validator.WithRequiredStructEnabled())
	uni      *ut.UniversalTranslator
	trans    ut.Translator
)

const (
	errMsgK8sSecretName    = "only allows the characters: a-z, 0-9, \".\", \"-\", starts and ends with alphanumeric character and has a maximum length of 253"
	errMsgK8sSecretDataKey = "only allows the characters: a-z, 0-9, \".\", \"-\", \"_\", starts and ends with alphanumeric character and has a maximum length of 253"
)

var customTranslations = map[string]string{
	"hostname_rfc1123":       "must be a valid RFC 1123 hostname",
	"k8sSecretName":          errMsgK8sSecretName,
	"k8sSecretDataKey":       errMsgK8sSecretDataKey,
	"len=0|k8sSecretName":    errMsgK8sSecretName,
	"len=0|k8sSecretDataKey": errMsgK8sSecretDataKey,
	"len=0|email":            "must be a valid email address",
}

func init() {
	en := en.New()
	uni = ut.New(en, en)

	trans, _ = uni.GetTranslator("en")

	err := entranslations.RegisterDefaultTranslations(validate, trans)
	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("k8sSecretName", k8sSecretName)
	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("k8sSecretDataKey", k8sSecretDataKey)
	if err != nil {
		panic(err)
	}

	for tag, errMessage := range customTranslations {
		addTranslation(tag, errMessage)
	}

}

// k8sSecretName validates the string against:
// - lowercase
// - alphanum with hyphens(-) and dot(.)
// - max=253
// - starts with alpha-num
// - ends with alpha-num
func k8sSecretName(fl validator.FieldLevel) bool {
	name := fl.Field().String()

	if validate.Var(name, "lowercase,max=253") != nil {
		return false
	}

	if !(startsWithAlphaNum(name) && endsWithAlphaNum(name) && alphaNumericWith(name, "-", ".")) {
		return false
	}

	return true
}

// k8sSecretName validates the string against:
// - alphanum with hyphens(-), dots(.) and underscores(_)
// - max=253
// - starts with alpha-num
// - ends with alpha-num
func k8sSecretDataKey(fl validator.FieldLevel) bool {
	key := fl.Field().String()

	if validate.Var(key, "max=253") != nil {
		return false
	}

	if !(startsWithAlphaNum(key) && endsWithAlphaNum(key) && alphaNumericWith(key, "-", ".", "_")) {
		return false
	}

	return true
}

func startsWithAlphaNum(s string) bool {
	regex := `^[A-Za-z0-9]`
	match, _ := regexp.MatchString(regex, s)

	return match
}

func endsWithAlphaNum(s string) bool {
	regex := `[A-Za-z0-9]$`
	match, _ := regexp.MatchString(regex, s)

	return match
}

func alphaNumericWith(s string, chars ...string) bool {
	regex := fmt.Sprintf(`[A-Za-z0-9%s]*`, strings.Join(chars, ""))
	matched, _ := regexp.MatchString(regex, s)

	return matched
}

func addTranslation(tag string, errMessage string) {
	registerFn := func(ut ut.Translator) error {
		return ut.Add(tag, fmt.Sprintf("{0} %s", errMessage), false)
	}

	transFn := func(ut ut.Translator, fe validator.FieldError) string {
		t, err := ut.T(tag, fe.Field())
		if err != nil {
			return fe.Error()
		}

		return t
	}

	_ = validate.RegisterTranslation(tag, trans, registerFn, transFn)
}
