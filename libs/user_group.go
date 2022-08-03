package libs

const (
	USBanned = 0
	USNormal = 1
	USAdmin  = 2
	USRoot   = 3
)

func IsAdmin(user_group int) bool {
	return (user_group == USAdmin || user_group == USRoot)
}

func IsBanned(user_group int) bool {
	return user_group == USBanned
}