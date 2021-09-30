package log

type dailyRotator struct {
}

func (r dailyRotator) Write(p []byte) (n int, err error) {
	return 0, nil
}
