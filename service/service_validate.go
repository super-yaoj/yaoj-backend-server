package service

import (
	"net/http"
	"yao/internal"
)

func NewServiceValidator() (val *validatorx) {
	val = NewValidator()
	// check whether the user_group belongs to admin
	val.RegisterValidation("admin", func(fv FieldValue) error {
		if !fv.Value.CanInt() {
			return HttpStatErr(http.StatusBadRequest)
		}
		if !internal.IsAdmin(int(fv.Value.Int())) {
			return HttpStatErr(http.StatusForbidden)
		}
		return nil // ok
	})
	// check whether the problem id exist in database
	val.RegisterValidation("probid", func(fv FieldValue) error {
		if !fv.Value.CanInt() {
			return HttpStatErr(http.StatusBadRequest)
		}
		if !internal.ProbExists(int(fv.Value.Int())) {
			return HttpStatErr(http.StatusNotFound)
		}
		return nil
	})
	val.RegisterValidation("submid", func(fv FieldValue) error {
		if !fv.Value.CanInt() {
			return HttpStatErr(http.StatusBadRequest)
		}
		if !internal.SubmExists(int(fv.Value.Int())) {
			return HttpStatErr(http.StatusNotFound)
		}
		return nil
	})
	val.RegisterValidation("ctstid", func(fv FieldValue) error {
		if !fv.Value.CanInt() {
			return HttpStatErr(http.StatusBadRequest)
		}
		if !internal.CTExists(int(fv.Value.Int())) {
			return HttpStatErr(http.StatusNotFound)
		}
		return nil
	})
	val.RegisterValidation("prmsid", func(fv FieldValue) error {
		if !fv.Value.CanInt() {
			return HttpStatErr(http.StatusBadRequest)
		}
		if !internal.PermExists(int(fv.Value.Int())) {
			return HttpStatErr(http.StatusNotFound)
		}
		return nil
	})
	val.RegisterValidation("userid", func(fv FieldValue) error {
		if !fv.Value.CanInt() {
			return HttpStatErr(http.StatusBadRequest)
		}
		if !internal.UserExists(int(fv.Value.Int())) {
			return HttpStatErr(http.StatusNotFound)
		}
		return nil
	})
	val.RegisterValidation("blogid", func(fv FieldValue) error {
		if !fv.Value.CanInt() {
			return HttpStatErr(http.StatusBadRequest)
		}
		if !internal.BlogExists(int(fv.Value.Int())) {
			return HttpStatErr(http.StatusNotFound)
		}
		return nil
	})
	val.RegisterValidation("cmntid", func(fv FieldValue) error {
		if !fv.Value.CanInt() {
			return HttpStatErr(http.StatusBadRequest)
		}
		if !internal.BlogCommentExists(int(fv.Value.Int())) {
			return HttpStatErr(http.StatusNotFound)
		}
		return nil
	})
	// authentication
	val.RegisterValidation("pagecanbound", func(fv FieldValue) error {
		page, ok := fv.Value.Interface().(Page)
		if !ok {
			return ValFailedErr("invalid Page struct")
		}
		// pp.Print(page, page.CanBound())
		if !page.CanBound() {
			return HttpStatErr(http.StatusBadRequest)
		}
		return nil
	})

	return val
}
