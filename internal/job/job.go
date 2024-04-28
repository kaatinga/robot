package job

type Job interface {
	User() string
	Next()
	PRURLs() []string
	Counter() uint16
}
