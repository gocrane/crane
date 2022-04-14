package utils

func ContainsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, str string) []string {
	var newSlice []string
	for _, item := range slice {
		if item != str {
			newSlice = append(newSlice, item)
		}
	}
	return newSlice
}
