package utils

func FindString(s string, ss []string) (string, int) {
	for i, str := range ss {
		if s == str {
			return str, i
		}
	}
	return "", -1
}
