package ignore

import (
	"bufio"
	"os"
	"path"
	"strings"
)

type rule struct {
	pattern string
	negate  bool
	dirOnly bool
}

type Matcher struct {
	rules []rule
}

func Load(filePath string) (Matcher, error) {
	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return Matcher{}, nil
		}
		return Matcher{}, err
	}
	defer f.Close()

	m := Matcher{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		r := rule{}
		if strings.HasPrefix(line, "!") {
			r.negate = true
			line = strings.TrimPrefix(line, "!")
		}
		if strings.HasSuffix(line, "/") {
			r.dirOnly = true
			line = strings.TrimSuffix(line, "/")
		}
		r.pattern = line
		m.rules = append(m.rules, r)
	}
	if err := s.Err(); err != nil {
		return Matcher{}, err
	}
	return m, nil
}

func (m Matcher) Match(p string) bool {
	p = strings.TrimPrefix(p, "./")
	matched := false
	for _, r := range m.rules {
		hit := false
		if r.dirOnly {
			hit = p == r.pattern || strings.HasPrefix(p, r.pattern+"/")
		} else if strings.Contains(r.pattern, "*") || strings.Contains(r.pattern, "?") {
			h, err := path.Match(r.pattern, p)
			hit = err == nil && h
			if !hit && !strings.Contains(r.pattern, "/") {
				h, err = path.Match(r.pattern, path.Base(p))
				hit = err == nil && h
			}
		} else {
			hit = p == r.pattern || strings.HasSuffix(p, "/"+r.pattern)
		}
		if hit {
			matched = !r.negate
		}
	}
	return matched
}
