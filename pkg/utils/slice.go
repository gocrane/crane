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
	for index, item := range slice {
		if item == str {
			newSlice := make([]string, 0, len(slice)-1)
			newSlice = append(newSlice, slice[:index]...)
			return append(newSlice, slice[index+1:]...)
		}
	}
	return slice
}
