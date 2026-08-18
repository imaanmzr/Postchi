package script

import (
	"fmt"
	"strings"

	"github.com/dop251/goja"
)

type TestResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

type Sandbox struct{}

func NewSandbox() *Sandbox { return &Sandbox{} }

type Context struct {
	vm       *goja.Runtime
	vars     map[string]string
	console  []string
	tests    []TestResult
	response struct {
		code    int
		body    string
		headers map[string]string
	}
}

func (s *Sandbox) NewContext(vars map[string]string) *Context {
	c := &Context{vars: copyVars(vars)}
	c.vm = goja.New()
	c.setup()
	return c
}

func copyVars(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (c *Context) setup() {
	c.vm.Set("console", map[string]func(string){
		"log": func(msg string) { c.console = append(c.console, msg) },
	})
	pm := map[string]any{
		"environment": map[string]any{
			"get": func(key string) string { return c.vars[key] },
			"set": func(key, val string) { c.vars[key] = val },
		},
		"variables": map[string]any{
			"get": func(key string) string { return c.vars[key] },
			"set": func(key, val string) { c.vars[key] = val },
		},
		"test": c.test,
		"expect": func(actual any) map[string]func(any) {
			return map[string]func(any){
				"to": func(expected any) {
					if fmt.Sprint(actual) != fmt.Sprint(expected) {
						panic(fmt.Sprintf("expected %v to equal %v", actual, expected))
					}
				},
				"eql": func(expected any) {
					if fmt.Sprint(actual) != fmt.Sprint(expected) {
						panic(fmt.Sprintf("expected %v to equal %v", actual, expected))
					}
				},
			}
		},
	}
	c.vm.Set("pm", pm)
}

func (c *Context) test(name string, fn goja.Callable) {
	passed := true
	msg := ""
	func() {
		defer func() {
			if r := recover(); r != nil {
				passed = false
				msg = fmt.Sprint(r)
			}
		}()
		_, _ = fn(goja.Undefined())
	}()
	c.tests = append(c.tests, TestResult{Name: name, Passed: passed, Message: msg})
}

func (c *Context) RunPreRequest(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil
	}
	_, err := c.vm.RunString(code)
	return err
}

func (c *Context) SetResponse(code int, body string, headers map[string]string) {
	c.response.code = code
	c.response.body = body
	c.response.headers = headers
	c.vm.Set("pm", map[string]any{
		"environment": map[string]any{
			"get": func(key string) string { return c.vars[key] },
			"set": func(key, val string) { c.vars[key] = val },
		},
		"variables": map[string]any{
			"get": func(key string) string { return c.vars[key] },
			"set": func(key, val string) { c.vars[key] = val },
		},
		"response": map[string]any{
			"code":    code,
			"status":  code,
			"body":    body,
			"headers": headers,
			"json": func() any {
				_ = c.vm.Set("___body", body)
				val, err := c.vm.RunString("JSON.parse(___body)")
				if err != nil {
					return nil
				}
				return val.Export()
			},
		},
		"test": c.test,
		"expect": func(actual any) map[string]func(any) {
			return map[string]func(any){
				"to": func(expected any) {
					if fmt.Sprint(actual) != fmt.Sprint(expected) {
						panic(fmt.Sprintf("expected %v to equal %v", actual, expected))
					}
				},
			}
		},
	})
}

func (c *Context) RunTests(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil
	}
	_, err := c.vm.RunString(code)
	return err
}

func (c *Context) Variables() map[string]string { return c.vars }
func (c *Context) Console() []string             { return c.console }
func (c *Context) TestResults() []TestResult {
	if c.tests == nil {
		return []TestResult{}
	}
	return c.tests
}
