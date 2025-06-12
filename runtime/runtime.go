package runtime

func Runtime() string {
	return `
	func addToArrayStart(arr []interface{}, elem interface{}) []interface{} {
		return append([]interface{}{elem}, arr...)
	}

	func addToArrayEnd(arr []interface{}, elem interface{}) []interface{} {
		return append(arr, elem)
	}
`
}
