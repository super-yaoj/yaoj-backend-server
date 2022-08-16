package server

// pagination query param
type Page struct {
	Left     *int `query:"left"`
	Right    *int `query:"right"`
	PageSize *int `query:"pagesize" validate:"gte=1,lte=100"`
}

func (r *Page) CanBound() bool {
	return r.PageSize != nil && (r.Left != nil || r.Right != nil)
}

func (r *Page) Bound() int {
	if r.Left != nil {
		return *r.Left
	} else if r.Right != nil {
		return *r.Right
	}
	return 0
}
func (r *Page) IsLeft() bool {
	return r.Left != nil
}
