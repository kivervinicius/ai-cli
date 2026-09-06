package release

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Version struct {
	Major, Minor, Patch int
	Pre                 string
	PreNumber           int
	Build               string
}

type Option struct {
	Kind, Label string
}

var semverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)

func Parse(raw string) (Version, error) {
	match := semverPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if match == nil {
		return Version{}, fmt.Errorf("invalid semantic version %q", raw)
	}
	vals := make([]int, 3)
	for i := range vals {
		var err error
		vals[i], err = strconv.Atoi(match[i+1])
		if err != nil {
			return Version{}, fmt.Errorf("invalid semantic version %q: %w", raw, err)
		}
	}
	v := Version{Major: vals[0], Minor: vals[1], Patch: vals[2]}
	if match[4] != "" {
		parts := strings.Split(match[4], ".")
		v.Pre = parts[0]
		if len(parts) == 2 {
			var err error
			v.PreNumber, err = strconv.Atoi(parts[1])
			if err != nil {
				return Version{}, fmt.Errorf("invalid semantic version %q: %w", raw, err)
			}
		}
	}
	v.Build = match[5]
	return v, nil
}

func (v Version) String() string {
	base := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Pre == "" {
		if v.Build != "" {
			return base + "+" + v.Build
		}
		return base
	}
	result := fmt.Sprintf("%s-%s.%d", base, v.Pre, v.PreNumber)
	if v.Build != "" {
		result += "+" + v.Build
	}
	return result
}

func Validate(raw string) error { _, err := Parse(raw); return err }

func NextVersion(current, kind string) (string, error) {
	v, err := Parse(current)
	if err != nil {
		return "", err
	}
	switch kind {
	case "patch":
		v.Patch++
		v.Pre = ""
		v.PreNumber = 0
		v.Build = ""
	case "minor":
		v.Minor++
		v.Patch = 0
		v.Pre = ""
		v.PreNumber = 0
		v.Build = ""
	case "major":
		v.Major++
		v.Minor = 0
		v.Patch = 0
		v.Pre = ""
		v.PreNumber = 0
		v.Build = ""
	case "beta", "rc":
		if kind == "beta" && v.Pre == "rc" {
			return "", fmt.Errorf("cannot return from RC to beta")
		}
		if v.Pre == kind {
			v.PreNumber++
			v.Build = ""
		} else {
			if v.Pre == "" {
				v.Minor++
				v.Patch = 0
			}
			v.Pre, v.PreNumber = kind, 0
			v.Build = ""
		}
	case "stable":
		if v.Pre == "" {
			return "", fmt.Errorf("stable promotion requires a prerelease")
		}
		v.Pre, v.PreNumber = "", 0
		v.Build = ""
	case "hotfix":
		v.Patch++
		v.Pre, v.PreNumber = "beta", 0
		v.Build = ""
	default:
		return "", fmt.Errorf("unknown release type %q", kind)
	}
	return v.String(), nil
}

func Options(current string) []Option {
	v, err := Parse(current)
	if err != nil {
		return nil
	}
	if v.Pre != "" {
		options := []Option{{"rc", "Próximo RC"}, {"stable", "Liberar estável"}, {"custom", "Versão customizada"}}
		if v.Pre == "beta" {
			options = append([]Option{{"beta", "Próximo beta"}}, options...)
		}
		return options
	}
	return []Option{{"patch", "Patch"}, {"minor", "Minor"}, {"major", "Major"}, {"beta", "Novo beta"}, {"rc", "Novo RC"}, {"custom", "Versão customizada"}}
}
