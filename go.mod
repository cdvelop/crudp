module github.com/cdvelop/crudp

go 1.24.4

require (
	github.com/cdvelop/tinybin v0.2.0
	github.com/cdvelop/tinystring v0.8.1
)

require github.com/cdvelop/tinyreflect v0.2.2 // indirect

replace github.com/cdvelop/tinybin => ../tinybin

replace github.com/cdvelop/tinyreflect => ../tinyreflect

replace github.com/cdvelop/tinystring => ../tinystring
