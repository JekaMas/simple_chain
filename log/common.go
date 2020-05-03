package log

const (
	SimplifiedLen = 5
)

func Simplify(address string) string {
	if len(address) < SimplifiedLen {
		return address
	}
	return address[:SimplifiedLen]
}
