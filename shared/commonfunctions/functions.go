package commonfunctions

func Getskip(limit int, offset int) int {
	if offset <= 1 {
		return 0
	}
	return (offset - 1) * limit
}
