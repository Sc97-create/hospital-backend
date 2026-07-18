package render

type Renderer interface {
	Render(
		notificationType string,
		data any,
	) (string, string, error)
}
