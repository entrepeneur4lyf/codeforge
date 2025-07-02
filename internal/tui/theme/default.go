package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// DefaultTheme implements the Default Theme interface
type DefaultTheme struct {
	BaseTheme
}

// NewDefaultTheme creates a new instance of the default theme.
func NewDefaultTheme() *DefaultTheme {
	// default Dark color palette
	// Primary colors from VSCode theme
	darkBackground := "#0E121B"  // editor.background
	darkCurrentLine := "#1D2535" // sideBar.background
	darkSelection := "#1C7CD650" // editor.selectionBackground
	darkForeground := "#F2F4F8"  // foreground
	darkComment := "#2b8a3e"     // comment color
	darkPrimary := "#1C7CD6"     // Primary blue (focusBorder, button.background)
	darkSecondary := "#329af0"   // Secondary blue (keyword, constant.language)
	darkAccent := "#fff3bf"      // Function color
	darkRed := "#f03e3e"         // Error color
	darkOrange := "#f59f00"      // Warning color
	darkGreen := "#37b24d"       // Success/string interpolation
	darkCyan := "#72c3fc"        // Variable/property color
	darkYellow := "#ffe066"      // Terminal yellow, git modified
	darkBorder := "#2B3750"      // Border color

	// Light mode colors (using inverted/adjusted versions for light theme)
	// Since the VSCode theme is dark-only, we'll create sensible light variants
	lightBackground := "#FFFFFF"
	lightCurrentLine := "#F5F5F5"
	lightSelection := "#1C7CD625"
	lightForeground := "#0E121B"
	lightComment := "#2b8a3e"
	lightPrimary := "#1862AB"   // Darker blue for light mode
	lightSecondary := "#1862AB" // Secondary blue
	lightAccent := "#B8860B"    // Darker gold for light mode
	lightRed := "#C92A2A"       // Error red for light
	lightOrange := "#E67700"    // Warning orange for light
	lightGreen := "#2B8A3E"     // Success green for light
	lightCyan := "#1098AD"      // Info cyan for light
	lightYellow := "#B8860B"    // Emphasized text for light
	lightBorder := "#DEE2E6"    // Border color for light

	theme := &DefaultTheme{}

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
		Dark:  "#A2B0CD", // From terminal.foreground
		Light: "#6C757D",
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
		Dark:  "#07090D", // panel.background
		Light: "#FAFAFA",
	}

	// Border colors
	theme.BorderNormalColor = lipgloss.AdaptiveColor{
		Dark:  darkBorder,
		Light: lightBorder,
	}
	theme.BorderFocusedColor = lipgloss.AdaptiveColor{
		Dark:  darkPrimary,
		Light: lightPrimary,
	}
	theme.BorderDimColor = lipgloss.AdaptiveColor{
		Dark:  darkSelection,
		Light: lightSelection,
	}

	// Diff view colors
	theme.DiffAddedColor = lipgloss.AdaptiveColor{
		Dark:  "#8ce99a", // gitDecoration.untrackedResourceForeground
		Light: "#37B24D",
	}
	theme.DiffRemovedColor = lipgloss.AdaptiveColor{
		Dark:  "#ffa8a8", // gitDecoration.deletedResourceForeground
		Light: "#F03E3E",
	}
	theme.DiffContextColor = lipgloss.AdaptiveColor{
		Dark:  "#A2B0CD",
		Light: "#6C757D",
	}
	theme.DiffHunkHeaderColor = lipgloss.AdaptiveColor{
		Dark:  "#A2B0CD",
		Light: "#6C757D",
	}
	theme.DiffHighlightAddedColor = lipgloss.AdaptiveColor{
		Dark:  "#37b24d",
		Light: "#2B8A3E",
	}
	theme.DiffHighlightRemovedColor = lipgloss.AdaptiveColor{
		Dark:  "#f03e3e",
		Light: "#C92A2A",
	}
	theme.DiffAddedBgColor = lipgloss.AdaptiveColor{
		Dark:  "#37b24d25", // diffEditor.insertedTextBackground
		Light: "#D3F9D8",
	}
	theme.DiffRemovedBgColor = lipgloss.AdaptiveColor{
		Dark:  "#f03e3e25", // diffEditor.removedTextBackground
		Light: "#FFE3E3",
	}
	theme.DiffContextBgColor = lipgloss.AdaptiveColor{
		Dark:  darkBackground,
		Light: lightBackground,
	}
	theme.DiffLineNumberColor = lipgloss.AdaptiveColor{
		Dark:  "#F2F4F850", // editorLineNumber.foreground
		Light: "#ADB5BD",
	}
	theme.DiffAddedLineNumberBgColor = lipgloss.AdaptiveColor{
		Dark:  "#37b24d40",
		Light: "#D3F9D8",
	}
	theme.DiffRemovedLineNumberBgColor = lipgloss.AdaptiveColor{
		Dark:  "#f03e3e40",
		Light: "#FFE3E3",
	}

	// Markdown colors
	theme.MarkdownTextColor = lipgloss.AdaptiveColor{
		Dark:  darkForeground,
		Light: lightForeground,
	}
	theme.MarkdownHeadingColor = lipgloss.AdaptiveColor{
		Dark:  darkSecondary, // Using keyword color
		Light: lightSecondary,
	}
	theme.MarkdownLinkColor = lipgloss.AdaptiveColor{
		Dark:  darkPrimary,
		Light: lightPrimary,
	}
	theme.MarkdownLinkTextColor = lipgloss.AdaptiveColor{
		Dark:  darkCyan,
		Light: lightCyan,
	}
	theme.MarkdownCodeColor = lipgloss.AdaptiveColor{
		Dark:  "#ffa8a8", // String color
		Light: "#E64980",
	}
	theme.MarkdownBlockQuoteColor = lipgloss.AdaptiveColor{
		Dark:  darkComment,
		Light: lightComment,
	}
	theme.MarkdownEmphColor = lipgloss.AdaptiveColor{
		Dark:  darkYellow,
		Light: lightYellow,
	}
	theme.MarkdownStrongColor = lipgloss.AdaptiveColor{
		Dark:  darkSecondary,
		Light: lightSecondary,
	}
	theme.MarkdownHorizontalRuleColor = lipgloss.AdaptiveColor{
		Dark:  "#F2F4F850",
		Light: "#ADB5BD",
	}
	theme.MarkdownListItemColor = lipgloss.AdaptiveColor{
		Dark:  "#6796e6", // From punctuation.definition.list.markdown
		Light: "#4263EB",
	}
	theme.MarkdownListEnumerationColor = lipgloss.AdaptiveColor{
		Dark:  darkCyan,
		Light: lightCyan,
	}
	theme.MarkdownImageColor = lipgloss.AdaptiveColor{
		Dark:  darkPrimary,
		Light: lightPrimary,
	}
	theme.MarkdownImageTextColor = lipgloss.AdaptiveColor{
		Dark:  darkCyan,
		Light: lightCyan,
	}
	theme.MarkdownCodeBlockColor = lipgloss.AdaptiveColor{
		Dark:  darkForeground,
		Light: lightForeground,
	}

	// Syntax highlighting colors
	theme.SyntaxCommentColor = lipgloss.AdaptiveColor{
		Dark:  darkComment,
		Light: lightComment,
	}
	theme.SyntaxKeywordColor = lipgloss.AdaptiveColor{
		Dark:  darkSecondary,
		Light: lightSecondary,
	}
	theme.SyntaxFunctionColor = lipgloss.AdaptiveColor{
		Dark:  darkAccent, // Function declarations
		Light: lightAccent,
	}
	theme.SyntaxVariableColor = lipgloss.AdaptiveColor{
		Dark:  darkCyan, // Variable color
		Light: lightCyan,
	}
	theme.SyntaxStringColor = lipgloss.AdaptiveColor{
		Dark:  "#ffa8a8", // String color
		Light: "#E64980",
	}
	theme.SyntaxNumberColor = lipgloss.AdaptiveColor{
		Dark:  "#d3f9d8", // constant.numeric
		Light: "#2B8A3E",
	}
	theme.SyntaxTypeColor = lipgloss.AdaptiveColor{
		Dark:  "#4EC9B0", // Type declarations
		Light: "#0C7C79",
	}
	theme.SyntaxOperatorColor = lipgloss.AdaptiveColor{
		Dark:  "#d4d4d4", // keyword.operator
		Light: "#495057",
	}
	theme.SyntaxPunctuationColor = lipgloss.AdaptiveColor{
		Dark:  darkForeground,
		Light: lightForeground,
	}

	return theme
}

func init() {
	// Register the default theme with the theme manager
	RegisterTheme("default", NewDefaultTheme())
}
