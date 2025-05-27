package color

// Color defines ANSI color strings for terminal coloring
type Color string

// TODO: this should probably live in another package from multirun

// Color constants
const (
	BrightBlue   Color = "1;34m"
	BrightCyan   Color = "1;36m"
	BrightGreen  Color = "1;32m"
	BrightPurple Color = "1;35m"
	BrightRed    Color = "1;31m"
	BrightYellow Color = "1;33m"
	Blue         Color = "0;34m"
	Green        Color = "0;32m"
	Purple       Color = "0;35m"
	Yellow       Color = "0;33m"
	Red          Color = "0;31m"
	Default      Color = "0m"
)

// ANSIEscapeColor is the ansi terminal prefix for changing font foreground color
const ANSIEscapeColor = "\033["
