# hyproxia/adapter

Building adapters for hyproxia to keep your code clean and organized, without the boilerplate.

## hyproxia/adapter/fasthttp

Example of using the adapter with fasthttp:

```go
func main() {
	apiProxy := hyproxia.New("https://api.example.com")
	defer apiProxy.Close()

	apiAdapter := fasthttpadapter.New("/api", apiProxy, fasthttpadapter.WithStripPrefix())

	handler := func(ctx *fasthttp.RequestCtx) {
		if apiAdapter.Handle(ctx) {
			return
		}

		switch string(ctx.Path()) {
		case "/health":
			ctx.WriteString("OK")
		case "/":
			ctx.WriteString("Welcome!")
		default:
			ctx.SetStatusCode(404)
			ctx.WriteString("Not Found")
		}
	}

	fasthttp.ListenAndServe(":8080", handler)
}
```

## hyproxia/adapter/fiber

Example of using the adapter with fiber:

```go
func main() {
	app := fiber.New()

	apiProxy := hyproxia.New("https://api.example.com")
	defer apiProxy.Close()

	apiAdapter := fiberadapter.New("/api", apiProxy, fiberadapter.WithStripPrefix())

	app.Use(apiAdapter.Handler())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Welcome!")
	})

	app.Listen(":8080")
}
```