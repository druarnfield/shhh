package components

// banner is the ASCII art for shhh.
const banner = `  ███████╗██╗  ██╗██╗  ██╗██╗  ██╗
  ██╔════╝██║  ██║██║  ██║██║  ██║
  ███████╗███████║███████║███████║
  ╚════██║██╔══██║██╔══██║██╔══██║
  ███████║██║  ██║██║  ██║██║  ██║
  ╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝`

// RenderBanner returns the styled ASCII banner.
func RenderBanner(styles Styles) string {
	return styles.Title.Render(banner)
}
