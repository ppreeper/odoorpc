package filter

func FilterList(filters ...any) []any {
	if len(filters) == 0 {
		return []any{}
	}
	if len(filters) == 1 {
		switch values := filters[0].(type) {
		case []any:
			if len(values) == 0 {
				return []any{}
			} else {
				return values
			}
		}
	}
	return filters[0].([]any)
}

func FilterString(filters ...string) []string {
	if len(filters) == 0 {
		return []string{}
	}
	if len(filters) == 1 && len(filters[0]) == 0 {
		return []string{}
	}
	return filters
}

type Filter []any

func NewFilter() *Filter {
	return &Filter{}
}

func (c *Filter) ToList() []any {
	filter := []any{}
	for _, v := range *c {
		switch v.(type) {
		case Term:
			filterTerm := []any{}
			for _, vv := range v.(Term) {
				filterTerm = append(filterTerm, vv)
			}
			filter = append(filter, filterTerm)
		case string:
			filter = append(filter, v)
		}
	}
	return filter
}

func (c *Filter) AddTerm(field, operator string, value any) *Filter {
	*c = append(*c, *NewTerm(field, operator, value))
	return c
}

func (c *Filter) Add(cri *Term) *Filter {
	*c = append(*c, cri)
	return c
}

func (c *Filter) And(c1, c2 *Term) *Filter {
	return c.combinedTerms("&", c1, c2)
}

func (c *Filter) Or(c1, c2 *Term) *Filter {
	return c.combinedTerms("|", c1, c2)
}

func (c *Filter) Not(cri *Term) *Filter {
	return c.combinedTerms("!", cri)
}

func (c *Filter) combinedTerms(operator string, cc ...*Term) *Filter {
	*c = append(*c, operator)
	for _, cri := range cc {
		*c = append(*c, cri)
	}
	return c
}

type Term []any

func NewTerm(field, operator string, value any) *Term {
	c := Term(newTuple(field, operator, value))
	return &c
}

func newTuple(values ...any) []any {
	t := make([]any, len(values))
	copy(t, values)
	return t
}
