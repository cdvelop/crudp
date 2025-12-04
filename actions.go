package crudp

// MethodToAction converts HTTP method to CRUD action byte
func MethodToAction(method string) byte {
	switch method {
	case "POST":
		return 'c'
	case "GET":
		return 'r'
	case "PUT":
		return 'u'
	case "DELETE":
		return 'd'
	default:
		return 0
	}
}

// ActionToMethod converts CRUD action byte to HTTP method
func ActionToMethod(action byte) string {
	switch action {
	case 'c':
		return "POST"
	case 'r':
		return "GET"
	case 'u':
		return "PUT"
	case 'd':
		return "DELETE"
	default:
		return ""
	}
}
