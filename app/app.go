package app

type App struct {
	name        string
	description string
}

type Option func(*App)

func WithDescription(description string) Option {
	return func(a *App) {
		a.description = description
	}
}

func WithGrpc() Option {
	return func(app *App) {

	}
}

func WithHttpServer() Option {
	return func(app *App) {

	}
}
