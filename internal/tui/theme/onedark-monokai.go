package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// OneDarkMonokaiTheme implements the Theme interface with colors from One Dark Pro Monokai Darker VSCode theme
type OneDarkMonokaiTheme struct {
	BaseTheme
}

// NewOneDarkMonokaiTheme creates a new instance of the One Dark Pro Monokai Darker theme.
func NewOneDarkMonokaiTheme() *OneDarkMonokaiTheme {
	// One Dark Pro Monokai Darker color palette
	darkBackground := "#121212"  // editor.background
	darkCurrentLine := "#181a1f" // sideBar.background / editorWidget.background
	darkForeground := "#bbbbbb"  // Default foreground
	darkComment := "#5c6370"     // Comment color
	darkPrimary := "#61afef"     // Blue (classes, variables)
	darkSecondary := "#c678dd"   // Purple (keywords, numbers)
	darkAccent := "#e5c07b"      // Yellow (strings)
	darkRed := "#e06c75"         // Red (keywords, tags)
	darkGreen := "#98c379"       // Green (functions)
	darkCyan := "#56b6c2"        // Cyan (constants, types)
	darkYellow := "#e5c07b"      // Yellow (strings)
	darkBorder := "#181a1f"      // General border color

	// Light mode colors (creating appropriate light variants)
	lightBackground := "#FAFAFA"
	lightCurrentLine := "#F3F3F3"
	lightForeground := "#383A42"
	lightComment := "#A0A1A7"
	lightPrimary := "#0184BC"   // Blue
	lightSecondary := "#A626A4" // Purple
	lightAccent := "#C18401"    // Yellow
	lightRed := "#E45649"       // Red
	lightGreen := "#50A14F"     // Green
	lightCyan := "#0997B3"      // Cyan
	lightYellow := "#C18401"    // Yellow
	lightBorder := "#D3D3D4"    // Border

	theme := &OneDarkMonokaiTheme{}

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
		Dark:  darkRed,
		Light: lightRed,
	}
	theme.WarningColor = lipgloss.AdaptiveColor{
		Dark:  "#cd9731", // token.warn-token
		Light: "#C18401",
	}
	theme.SuccessColor = lipgloss.AdaptiveColor{
		Dark:  darkGreen,
		Light: lightGreen,
	}
	theme.InfoColor = lipgloss.AdaptiveColor{
		Dark:  "#6796e6", // token.info-token
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
		Dark:  "#abb2bf", // General text color
		Light: "#494C55",
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
		Dark:  "#020202", // statusBar.background
		Light: "#FFFFFF",
	}

	// Border colors
	theme.BorderNormalColor = lipgloss.AdaptiveColor{
		Dark:  darkBorder,
		Light: lightBorder,
	}
	theme.BorderFocusedColor = lipgloss.AdaptiveColor{
		Dark:  "#528bff", // button.background
		Light: lightPrimary,
	}
	theme.BorderDimColor = lipgloss.AdaptiveColor{
		Dark:  "#1d1f23", // Various UI backgrounds
		Light: "#E5E5E6",
	}

	// Diff view colors
	theme.DiffAddedColor = lipgloss.AdaptiveColor{
		Dark:  darkYellow, // diff.inserted
		Light: lightGreen,
	}
	theme.DiffRemovedColor = lipgloss.AdaptiveColor{
		Dark:  darkSecondary, // diff.deleted
		Light: lightRed,
	}
	theme.DiffContextColor = lipgloss.AdaptiveColor{
		Dark:  "#75715E", // diff.header
		Light: "#6A737D",
	}
	theme.DiffHunkHeaderColor = lipgloss.AdaptiveColor{
		Dark:  "#75715E",
		Light: "#6A737D",
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
		Dark:  "#00809b33", // diffEditor.insertedTextBackground
		Light: "#E6FFED",
	}
	theme.DiffRemovedBgColor = lipgloss.AdaptiveColor{
		Dark:  "#c678dd30",
		Light: "#FFEEF0",
	}
	theme.DiffContextBgColor = lipgloss.AdaptiveColor{
		Dark:  darkBackground,
		Light: lightBackground,
	}
	theme.DiffLineNumberColor = lipgloss.AdaptiveColor{
		Dark:  "#495162", // editorLineNumber.foreground
		Light: "#959DA5",
	}
	theme.DiffAddedLineNumberBgColor = lipgloss.AdaptiveColor{
		Dark:  "#98c37940",
		Light: "#CDFFD8",
	}
	theme.DiffRemovedLineNumberBgColor = lipgloss.AdaptiveColor{
		Dark:  "#e06c7540",
		Light: "#FFDCE0",
	}

	// Markdown colors
	theme.MarkdownTextColor = lipgloss.AdaptiveColor{
		Dark:  darkForeground,
		Light: lightForeground,
	}
	theme.MarkdownHeadingColor = lipgloss.AdaptiveColor{
		Dark:  darkRed, // markup.heading.markdown
		Light: lightRed,
	}
	theme.MarkdownLinkColor = lipgloss.AdaptiveColor{
		Dark:  darkPrimary, // markup.underline.link.markdown
		Light: lightPrimary,
	}
	theme.MarkdownLinkTextColor = lipgloss.AdaptiveColor{
		Dark:  darkPrimary,
		Light: lightPrimary,
	}
	theme.MarkdownCodeColor = lipgloss.AdaptiveColor{
		Dark:  darkCyan, // markup.raw
		Light: lightCyan,
	}
	theme.MarkdownBlockQuoteColor = lipgloss.AdaptiveColor{
		Dark:  darkGreen, // markup.quote.markdown
		Light: lightGreen,
	}
	theme.MarkdownEmphColor = lipgloss.AdaptiveColor{
		Dark:  darkForeground,
		Light: lightForeground,
	}
	theme.MarkdownStrongColor = lipgloss.AdaptiveColor{
		Dark:  darkForeground,
		Light: lightForeground,
	}
	theme.MarkdownHorizontalRuleColor = lipgloss.AdaptiveColor{
		Dark:  darkSecondary, // meta.separator.markdown
		Light: lightSecondary,
	}
	theme.MarkdownListItemColor = lipgloss.AdaptiveColor{
		Dark:  "#ffffff", // punctuation.definition.list_item.markdown
		Light: "#383A42",
	}
	theme.MarkdownListEnumerationColor = lipgloss.AdaptiveColor{
		Dark:  "#ffffff",
		Light: "#383A42",
	}
	theme.MarkdownImageColor = lipgloss.AdaptiveColor{
		Dark:  darkPrimary,
		Light: lightPrimary,
	}
	theme.MarkdownImageTextColor = lipgloss.AdaptiveColor{
		Dark:  darkPrimary,
		Light: lightPrimary,
	}
	theme.MarkdownCodeBlockColor = lipgloss.AdaptiveColor{
		Dark:  darkCyan,
		Light: lightCyan,
	}

	// Syntax highlighting colors
	theme.SyntaxCommentColor = lipgloss.AdaptiveColor{
		Dark:  darkComment,
		Light: lightComment,
	}
	theme.SyntaxKeywordColor = lipgloss.AdaptiveColor{
		Dark:  darkRed, // keyword, storage
		Light: lightRed,
	}
	theme.SyntaxFunctionColor = lipgloss.AdaptiveColor{
		Dark:  darkGreen, // entity.name.function
		Light: lightGreen,
	}
	theme.SyntaxVariableColor = lipgloss.AdaptiveColor{
		Dark:  darkPrimary, // variable.readwrite
		Light: lightPrimary,
	}
	theme.SyntaxStringColor = lipgloss.AdaptiveColor{
		Dark:  darkYellow, // string
		Light: lightYellow,
	}
	theme.SyntaxNumberColor = lipgloss.AdaptiveColor{
		Dark:  darkSecondary, // constant.numeric
		Light: lightSecondary,
	}
	theme.SyntaxTypeColor = lipgloss.AdaptiveColor{
		Dark:  darkCyan, // storage.type
		Light: lightCyan,
	}
	theme.SyntaxOperatorColor = lipgloss.AdaptiveColor{
		Dark:  darkRed, // keyword.operator
		Light: lightRed,
	}
	theme.SyntaxPunctuationColor = lipgloss.AdaptiveColor{
		Dark:  "#abb2bf", // General punctuation
		Light: "#383A42",
	}

	return theme
}

func init() {
	// Register the one dark pro monokai theme with the theme manager
	RegisterTheme("onedark-monokai", NewOneDarkMonokaiTheme())
}
