# Rate-Limiting-API

A modern Go web application using **HTMX** for interactivity and **html/template** for server-side rendering.

## Features

- **SSR**: Fast, SEO-friendly server-side rendering
- **HTMX**: Dynamic interactions without writing JavaScript
- **Embedded Templates**: Templates compiled into binary via `go:embed`
- **Tailwind CSS**: Utility-first styling (via CDN)
- **Thread-Safe**: Mutex-protected state for concurrent access
- **Structured Logging**: Using `log/slog`

## Project Structure

```
.
├── main.go           # HTTP server, handlers, and business logic
├── templates/
│   ├── index.html    # Main page layout
│   └── list.html     # Todo list fragment (partial)
├── go.mod
└── README.md
```

## Running

```bash
go run .
```

Visit [http://localhost:8080](http://localhost:8080)

## HTMX Patterns Used

| Attribute | Purpose |
|-----------|---------|
| `hx-post` | Send POST request (add todo) |
| `hx-delete` | Send DELETE request (remove todo) |
| `hx-target` | Specify which element to update |
| `hx-swap` | Control how HTML is inserted (innerHTML) |
| `hx-indicator` | Show loading spinner during request |
| `hx-confirm` | Confirm before destructive actions |

## How It Works

1. **Initial Load**: Server renders full HTML page
2. **Add Todo**: Form triggers `hx-post="/todos"`, server returns updated list fragment
3. **Toggle/Delete**: Checkbox/button triggers request, server returns updated list
4. **Fragment Swap**: HTMX replaces `#todo-list` content with response

## Customization

- **Database**: Replace `[]Todo` slice with SQLite/Postgres
- **Styling**: Replace Tailwind CDN with build step for production
- **Auth**: Add session middleware for user authentication

## Resources

- [HTMX Documentation](https://htmx.org/docs/)
- [HTMX Reference](https://htmx.org/reference/)
- [Go html/template](https://pkg.go.dev/html/template)
