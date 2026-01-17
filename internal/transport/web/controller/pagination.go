package controller

import (
	"fmt"
	"net/url"
	"strconv"
)

const (
	defaultPage     = 1
	defaultPageSize = 50
	maxPageSize     = 200
)

func parsePagination(q url.Values) (page, pageSize int, err error) {
	page = defaultPage
	pageSize = defaultPageSize

	if q.Has("page") {
		p, err := strconv.ParseInt(q.Get("page"), 10, 32)
		if err != nil {
			return 0, 0, fmt.Errorf("unable to parse page from query: %w", err)
		}
		if p < 1 {
			return 0, 0, fmt.Errorf("invalid page value [%d]", p)
		}
		page = int(p)
	}

	if q.Has("page_size") {
		ps, err := strconv.ParseInt(q.Get("page_size"), 10, 32)
		if err != nil {
			return 0, 0, fmt.Errorf("unable to parse page size from query: %w", err)
		}
		if ps > maxPageSize {
			return 0, 0, fmt.Errorf("page size [%d] exceeds limit [%d]", ps, maxPageSize)
		}
		if ps < 1 {
			return 0, 0, fmt.Errorf("invalid page size value [%d]", ps)
		}
		pageSize = int(ps)
	}

	return page, pageSize, nil
}
