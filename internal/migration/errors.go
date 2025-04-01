package migration

type apiErrMsg struct {
	Err error
}

type timeConvertErrMsg struct {
	Err error
}

type fatalErrMsg struct {
	Msg string
	Err error
}
