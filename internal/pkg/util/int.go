package util

import "strconv"

func ToInt(input string, defaultValue ...int) (int, error) {
	if input == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return 1, nil
	}

	integer, err := strconv.Atoi(input)
	if err != nil {
		return 0, err
	}

	return integer, nil
}