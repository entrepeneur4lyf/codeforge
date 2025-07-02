package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// NarutoDarkTheme implements the Theme interface with Naruto colors.
type NarutoDarkTheme struct {
	BaseTheme
}

// NewNarutoDarkTheme creates a new instance of the Dark theme.
func NewNarutoDarkTheme() *NarutoDarkTheme {
	// Dark do dark color palette from VSCode theme
	darkBackground := "#0d0d1d"  // editor.background
	darkCurrentLine := "#070416" // sideBar.background
	darkForeground := "#ffffff"  // editor.foreground
	darkComment := "#48515e"     // comment colors
	darkPrimary := "#55acf3"     // Blue (methods, imports)
	darkSecondary := "#cc75f5"   // Purple (using, namespace)
	darkAccent := "#f8aa35"      // Orange (classes, types)
	darkRed := "#fd4c87"         // Red/Pink (keywords like void, class, new)
	darkOrange := "#f0608d"      // Orange/Pink (packages, functions)
	darkGreen := "#00faa7"       // Green (modifiers like public, private)
	darkCyan := "#649fec"        // Cyan (cursor color)
	darkYellow := "#faa84b"      // Yellow (variables, parameters)
	darkBorder := "#161620"      // activityBar.background

	// Light mode colors (creating appropriate light variants)
	lightBackground := "#FFFFFF"
	lightCurrentLine := "#F8F8F8"
	lightForeground := "#000000"
	lightComment := "#6A737D"
	lightPrimary := "#0366D6"   // Blue
	lightSecondary := "#6F42C1" // Purple
	lightAccent := "#D15704"    // Orange
	lightRed := "#D73A49"       // Red/Pink
	lightOrange := "#D1456D"    // Orange/Pink
	lightGreen := "#22863A"     // Green
	lightCyan := "#1B7C83"      // Cyan
	lightYellow := "#B08800"    // Yellow
	lightBorder := "#E1E4E8"    // Border

	theme := &NarutoDarkTheme{}

	// Base colors
	theme.PrimaryColor = lipgloss.AdaptiveColor{
		Dark:  darkPrimary,
		Light: lightPrimary,
	}
	theme.SecondaryColor = lipgloss.AdaptiveColor{
		Dark:  darkSecondary,
		Light: lightSecondary,
	}
	theme.AccentColor = lipgloss.AdaptiveColor{
		Dark:  darkAccent,
		Light: lightAccent,
	}

	// Status colors
	theme.ErrorColor = lipgloss.AdaptiveColor{
		Dark:  "#FF5370", // From tokenColors
		Light: "#D73A49",
	}
	theme.WarningColor = lipgloss.AdaptiveColor{
		Dark:  darkOrange,
		Light: lightOrange,
	}
	theme.SuccessColor = lipgloss.AdaptiveColor{
		Dark:  darkGreen,
		Light: lightGreen,
	}
	theme.InfoColor = lipgloss.AdaptiveColor{
		Dark:  darkCyan,
		Light: lightCyan,
	}

	// Text colors
	theme.TextColor = lipgloss.AdaptiveColor{
		Dark:  darkForeground,
		Light: lightForeground,
	}
	theme.TextMutedColor = lipgloss.AdaptiveColor{
		Dark:  darkComment,
		Light: lightComment,
	}
	theme.TextEmphasizedColor = lipgloss.AdaptiveColor{
		Dark:  darkYellow,
		Light: lightYellow,
	}

	// Background colors
	theme.BackgroundColor = lipgloss.AdaptiveColor{
		Dark:  darkBackground,
		Light: lightBackground,
	}
	theme.BackgroundSecondaryColor = lipgloss.AdaptiveColor{
		Dark:  darkCurrentLine,
		Light: lightCurrentLine,
	}
	theme.BackgroundDarkerColor = lipgloss.AdaptiveColor{
		Dark:  "#09051b", // tab.inactiveBackground
		Light: "#FAFBFC",
	}

	// Border colors
	theme.BorderNormalColor = lipgloss.AdaptiveColor{
		Dark:  darkBorder,
		Light: lightBorder,
	}
	theme.BorderFocusedColor = lipgloss.AdaptiveColor{
		Dark:  "#00aeff", // tab.activeBorder
		Light: lightPrimary,
	}
	theme.BorderDimColor = lipgloss.AdaptiveColor{
		Dark:  "#0f082e", // tab.activeBackground
		Light: "#F3F4F6",
	}

	// Diff view colors
	theme.DiffAddedColor = lipgloss.AdaptiveColor{
		Dark:  "#C3E88D", // String color (green)
		Light: "#22863A",
	}
	theme.DiffRemovedColor = lipgloss.AdaptiveColor{
		Dark:  "#FF5370", // Deleted/error color
		Light: "#D73A49",
	}
	theme.DiffContextColor = lipgloss.AdaptiveColor{
		Dark:  "#B2CCD6", // CSS support color
		Light: "#586069",
	}
	theme.DiffHunkHeaderColor = lipgloss.AdaptiveColor{
		Dark:  "#65737E", // Markdown punctuation
		Light: "#586069",
	}
	theme.DiffHighlightAddedColor = lipgloss.AdaptiveColor{
		Dark:  darkGreen,
		Light: lightGreen,
	}
	theme.DiffHighlightRemovedColor = lipgloss.AdaptiveColor{
		Dark:  darkRed,
		Light: lightRed,
	}
	theme.DiffAddedBgColor = lipgloss.AdaptiveColor{
		Dark:  "#00faa730",
		Light: "#E6FFED",
	}
	theme.DiffRemovedBgColor = lipgloss.AdaptiveColor{
		Dark:  "#fd4c8730",
		Light: "#FFEEF0",
	}
	theme.DiffContextBgColor = lipgloss.AdaptiveColor{
		Dark:  darkBackground,
		Light: lightBackground,
	}
	theme.DiffLineNumberColor = lipgloss.AdaptiveColor{
		Dark:  "#4e566660", // scrollbarSlider.background
		Light: "#959DA5",
	}
	theme.DiffAddedLineNumberBgColor = lipgloss.AdaptiveColor{
		Dark:  "#00faa740",
		Light: "#CDFFD8",
	}
	theme.DiffRemovedLineNumberBgColor = lipgloss.AdaptiveColor{
		Dark:  "#fd4c8740",
		Light: "#FFDCE0",
	}

	// Markdown colors
	theme.MarkdownTextColor = lipgloss.AdaptiveColor{
		Dark:  "#5d90ff", // Markdown plain text
		Light: "#24292E",
	}
	theme.MarkdownHeadingColor = lipgloss.AdaptiveColor{
		Dark:  "#ff8800", // Markdown heading
		Light: "#D15704",
	}
	theme.MarkdownLinkColor = lipgloss.AdaptiveColor{
		Dark:  "#fffc35", // Markdown link
		Light: "#0366D6",
	}
	theme.MarkdownLinkTextColor = lipgloss.AdaptiveColor{
		Dark:  "#C792EA", // Markdown link description
		Light: "#6F42C1",
	}
	theme.MarkdownCodeColor = lipgloss.AdaptiveColor{
		Dark:  "#C792EA", // Markup raw inline
		Light: "#6F42C1",
	}
	theme.MarkdownBlockQuoteColor = lipgloss.AdaptiveColor{
		Dark:  "#65737E", // Markdown blockquote
		Light: "#6A737D",
	}
	theme.MarkdownEmphColor = lipgloss.AdaptiveColor{
		Dark:  "#f07178", // Markup italic
		Light: "#D1456D",
	}
	theme.MarkdownStrongColor = lipgloss.AdaptiveColor{
		Dark:  "#f07178", // Markup bold
		Light: "#D1456D",
	}
	theme.MarkdownHorizontalRuleColor = lipgloss.AdaptiveColor{
		Dark:  "#65737E", // Separator
		Light: "#E1E4E8",
	}
	theme.MarkdownListItemColor = lipgloss.AdaptiveColor{
		Dark:  "#5d90ff", // List item
		Light: "#24292E",
	}
	theme.MarkdownListEnumerationColor = lipgloss.AdaptiveColor{
		Dark:  "#5d90ff",
		Light: "#24292E",
	}
	theme.MarkdownImageColor = lipgloss.AdaptiveColor{
		Dark:  darkPrimary,
		Light: lightPrimary,
	}
	theme.MarkdownImageTextColor = lipgloss.AdaptiveColor{
		Dark:  "#C792EA",
		Light: "#6F42C1",
	}
	theme.MarkdownCodeBlockColor = lipgloss.AdaptiveColor{
		Dark:  "#EEFFFF", // Fenced code block
		Light: "#24292E",
	}

	// Syntax highlighting colors
	theme.SyntaxCommentColor = lipgloss.AdaptiveColor{
		Dark:  darkComment,
		Light: lightComment,
	}
	theme.SyntaxKeywordColor = lipgloss.AdaptiveColor{
		Dark:  darkRed, // Keywords (void, class, new)
		Light: lightRed,
	}
	theme.SyntaxFunctionColor = lipgloss.AdaptiveColor{
		Dark:  "#2ff76b", // Functions JS
		Light: "#22863A",
	}
	theme.SyntaxVariableColor = lipgloss.AdaptiveColor{
		Dark:  "#ff91de", // Variables JS
		Light: "#D73A49",
	}
	theme.SyntaxStringColor = lipgloss.AdaptiveColor{
		Dark:  "#C3E88D", // Strings
		Light: "#22863A",
	}
	theme.SyntaxNumberColor = lipgloss.AdaptiveColor{
		Dark:  "#F78C6C", // Numbers, constants
		Light: "#D15704",
	}
	theme.SyntaxTypeColor = lipgloss.AdaptiveColor{
		Dark:  darkAccent, // Types, classes
		Light: lightAccent,
	}
	theme.SyntaxOperatorColor = lipgloss.AdaptiveColor{
		Dark:  "#89DDFF", // Operators
		Light: "#1B7C83",
	}
	theme.SyntaxPunctuationColor = lipgloss.AdaptiveColor{
		Dark:  "#89DDFF", // Punctuation
		Light: "#24292E",
	}

	return theme
}

func init() {
	// Register the dark theme with the theme manager
	RegisterTheme("naruto", NewNarutoDarkTheme())
}
