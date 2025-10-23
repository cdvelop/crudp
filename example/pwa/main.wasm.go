//go:build wasm
// +build wasm

package main

import (
	"reflect"
	"syscall/js"

	"github.com/cdvelop/crudp/example/modules/contact"
	. "github.com/cdvelop/tinystring"
)

func main() {
	// Crear el elemento div
	dom := js.Global().Get("document").Call("createElement", "div")

	buf := Convert().
		Write("<h1>CRUDP WebAssembly..</h1>").
		Write("<div>Running...</div>").
		Write(`<div id="results"></div>`) // Añadir el div de resultados aquí

	dom.Set("innerHTML", buf.String())

	// Obtener el body del documento y agregar el elemento
	body := js.Global().Get("document").Get("body")
	body.Call("appendChild", dom)

	logger := func(msg ...any) {
		js.Global().Get("console").Call("log", Translate(msg...).String())
	}

	// Initialize struct with test contact values
	data := contact.Contact{
		Name:    "Alice Example",
		Email:   "alice@example.com",
		Phone:   "+1-555-0100",
		Subject: "Inquiry about services",
		Message: "Hello, I'd like more information about your offerings.",
	}

	logger("=== Getting field names and values ===")

	v := reflect.ValueOf(data)
	t := v.Type()
	numFields := t.NumField()

	logger("Found", numFields, "fields:")

	// Build HTML form output
	htmlOutput := Convert().Write("<form>")

	for i := 0; i < numFields; i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		fieldName := fieldType.Name
		fieldValue := field.Interface()

		logger("Field", i, ":", fieldName, "=", fieldValue)

		htmlOutput.Write(`<div>`)
		// Create a label for the field
		htmlOutput.Write(`<label for="`).Write(fieldName).Write(`">`).Write(fieldName).Write(`:</label>`)

		// Create an input element
		if fieldName == "Message" {
			htmlOutput.Write(`<textarea id="`).Write(fieldName).Write(`"name="`).Write(fieldName).Write(`">`).Write(fieldValue).Write(`</textarea>`)
		} else {
			inputType := "text"
			if fieldName == "Email" {
				inputType = "email"
			}
			htmlOutput.Write(`<input type="`).Write(inputType).Write(`"id="`).Write(fieldName).Write(`"name="`).Write(fieldName).Write(`"value="`).Write(fieldValue).Write(`">`)
		}
		htmlOutput.Write(`</div>`)
	}

	htmlOutput.Write(`<button type="submit">Send</button>`)
	htmlOutput.Write("</form>")

	// Update DOM with results
	resultsDiv := js.Global().Get("document").Call("getElementById", "results")
	resultsDiv.Set("innerHTML", htmlOutput.String())

	logger("Test completed successfully!")

	select {}
}
