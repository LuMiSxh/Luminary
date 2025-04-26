package utils

import "strconv"

func ParseFloat64(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func ParseInt(s string) (int, error) {
	return strconv.Atoi(s)
}
