package httputil

import (
	"fmt"
	"strconv"
)

// ParsePagination parses and validates page/per_page query parameters.
// Returns (page, perPage, error). Defaults: page=1, perPage=20.
func ParsePagination(pageStr, perPageStr string) (int, int, error) {
	page := 1
	perPage := 20

	if pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid page parameter: must be an integer")
		}
		if p < 1 {
			p = 1
		}
		page = p
	}

	if perPageStr != "" {
		pp, err := strconv.Atoi(perPageStr)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid per_page parameter: must be an integer")
		}
		if pp < 1 || pp > 100 {
			return 0, 0, fmt.Errorf("per_page must be between 1 and 100")
		}
		perPage = pp
	}

	return page, perPage, nil
}
