package regexp

import "regexp"

func MatchesRegex(pattern, target string) bool {
	if pattern == "" {
		return true
	}

	matched, err := regexp.MatchString(pattern, target)
	if err != nil {
		return false // Assume it's not a match if an error occurs.
	}

	return matched
}
