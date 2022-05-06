package utils

import "strconv"

func Int32SliceToStrSlice(intSlice []int32) (strSlice []string) {
	for _, i := range intSlice {
		strSlice = append(strSlice, strconv.FormatInt(int64(i), 10))
	}
	return strSlice
}
