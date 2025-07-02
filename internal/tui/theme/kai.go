package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// KaiTheme implements the Theme interface with colors from Kai VSCode theme
type KaiTheme struct {
	BaseTheme
}

// NewKaiTheme creates a new instance of the Kai theme.
func NewKaiTheme() *KaiTheme {
	// Kai Dark color palette from VSCode theme
	darkBackground := "#0c0c15"  // editor.background
	darkCurrentLine := "#11111f" // sideBar.background
	darkForeground := "#d4d4d4"  // editor.foreground / foreground
	darkComment := "#7f848e"     // comment color from tokenColors
	darkPrimary := "#61afef"     // Blue (functions, links)
	darkSecondary := "#c678dd"   // Purple (keywords)
	darkAccent := "#e5c07b"      // Yellow/Gold (types, classes)
	darkRed := "#e06c75"         // Red (variables, tags)
	darkOrange := "#d19a66"      // Orange (constants, numbers)
	darkGreen := "#98c379"       // Green (strings)
	darkCyan := "#56b6c2"        // Cyan (support functions, operators)
	darkYellow := "#e5c07b"      // Yellow (emphasized text)
	darkBorder := "#80808059"    // panel.border

	// Light mode colors (creating appropriate light variants)
	lightBackground := "#FFFFFF"
	lightCurrentLine := "#F5F5F5"
	lightForeground := "#333333"
	lightComment := "#6A737D"
	lightPrimary := "#0366D6"   // Blue
	lightSecondary := "#6F42C1" // Purple
	lightAccent := "#E36209"    // Orange
	lightRed := "#D73A49"       // Red
	lightOrange := "#E36209"    // Orange
	lightGreen := "#22863A"     // Green
	lightCyan := "#1B7C83"      // Cyan
	lightYellow := "#B08800"    // Yellow
	lightBorder := "#E1E4E8"    // Border

	theme := &KaiTheme{}

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
		Dark:  "#070b17", // panel.background
		Light: "#FAFBFC",
	}

	// Border colors
	theme.BorderNormalColor = lipgloss.AdaptiveColor{
		Dark:  darkBorder,
		Light: lightBorder,
	}
	theme.BorderFocusedColor = lipgloss.AdaptiveColor{
		Dark:  "#1d76b2", // focusBorder
		Light: lightPrimary,
	}
	theme.BorderDimColor = lipgloss.AdaptiveColor{
		Dark:  "#1b1b36", // list.inactiveSelectionBackground
		Light: "#F3F4F6",
	}

	// Diff view colors
	theme.DiffAddedColor = lipgloss.AdaptiveColor{
		Dark:  "#81b88b", // gitDecoration.addedResourceForeground
		Light: "#22863A",
	}
	theme.DiffRemovedColor = lipgloss.AdaptiveColor{
		Dark:  "#c74e39", // gitDecoration.deletedResourceForeground
		Light: "#D73A49",
	}
	theme.DiffContextColor = lipgloss.AdaptiveColor{
		Dark:  "#cccccc", // sideBar.foreground
		Light: "#586069",
	}
	theme.DiffHunkHeaderColor = lipgloss.AdaptiveColor{
		Dark:  "#999999", // editorCodeLens.foreground
		Light: "#586069",
	}
	theme.DiffHighlightAddedColor = lipgloss.AdaptiveColor{
		Dark:  "#23d18b", // terminal.ansiBrightGreen
		Light: "#34D058",
	}
	theme.DiffHighlightRemovedColor = lipgloss.AdaptiveColor{
		Dark:  "#f14c4c", // terminal.ansiBrightRed
		Light: "#EA4A5A",
	}
	theme.DiffAddedBgColor = lipgloss.AdaptiveColor{
		Dark:  "#9bb95533", // diffEditor.insertedTextBackground
		Light: "#E6FFED",
	}
	theme.DiffRemovedBgColor = lipgloss.AdaptiveColor{
		Dark:  "#ff000033", // diffEditor.removedTextBackground
		Light: "#FFEEF0",
	}
	theme.DiffContextBgColor = lipgloss.AdaptiveColor{
		Dark:  darkBackground,
		Light: lightBackground,
	}
	theme.DiffLineNumberColor = lipgloss.AdaptiveColor{
		Dark:  "#858585", // editorLineNumber.foreground
		Light: "#959DA5",
	}
	theme.DiffAddedLineNumberBgColor = lipgloss.AdaptiveColor{
		Dark:  "#587c0c", // editorGutter.addedBackground
		Light: "#CDFFD8",
	}
	theme.DiffRemovedLineNumberBgColor = lipgloss.AdaptiveColor{
		Dark:  "#94151b", // editorGutter.deletedBackground
		Light: "#FFDCE0",
	}

	// Markdown colors
	theme.MarkdownTextColor = lipgloss.AdaptiveColor{
		Dark:  darkForeground,
		Light: lightForeground,
	}
	theme.MarkdownHeadingColor = lipgloss.AdaptiveColor{
		Dark:  darkRed, // Headings use red in Kai theme
		Light: lightRed,
	}
	theme.MarkdownLinkColor = lipgloss.AdaptiveColor{
		Dark:  darkPrimary,
		Light: lightPrimary,
	}
	theme.MarkdownLinkTextColor = lipgloss.AdaptiveColor{
		Dark:  darkSecondary,
		Light: lightSecondary,
	}
	theme.MarkdownCodeColor = lipgloss.AdaptiveColor{
		Dark:  darkGreen,
		Light: lightGreen,
	}
	theme.MarkdownBlockQuoteColor = lipgloss.AdaptiveColor{
		Dark:  darkComment,
		Light: lightComment,
	}
	theme.MarkdownEmphColor = lipgloss.AdaptiveColor{
		Dark:  darkSecondary, // Italic uses purple
		Light: lightSecondary,
	}
	theme.MarkdownStrongColor = lipgloss.AdaptiveColor{
		Dark:  darkOrange, // Bold uses orange
		Light: lightOrange,
	}
	theme.MarkdownHorizontalRuleColor = lipgloss.AdaptiveColor{
		Dark:  "#404040", // editorIndentGuide.background
		Light: "#E1E4E8",
	}
	theme.MarkdownListItemColor = lipgloss.AdaptiveColor{
		Dark:  darkRed, // List punctuation uses red
		Light: lightRed,
	}
	theme.MarkdownListEnumerationColor = lipgloss.AdaptiveColor{
		Dark:  darkRed,
		Light: lightRed,
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
		Dark:  darkPrimary,
		Light: lightPrimary,
	}
	theme.SyntaxVariableColor = lipgloss.AdaptiveColor{
		Dark:  darkRed,
		Light: lightRed,
	}
	theme.SyntaxStringColor = lipgloss.AdaptiveColor{
		Dark:  darkGreen,
		Light: lightGreen,
	}
	theme.SyntaxNumberColor = lipgloss.AdaptiveColor{
		Dark:  darkOrange,
		Light: lightOrange,
	}
	theme.SyntaxTypeColor = lipgloss.AdaptiveColor{
		Dark:  darkAccent,
		Light: lightAccent,
	}
	theme.SyntaxOperatorColor = lipgloss.AdaptiveColor{
		Dark:  darkCyan,
		Light: lightCyan,
	}
	theme.SyntaxPunctuationColor = lipgloss.AdaptiveColor{
		Dark:  darkForeground,
		Light: lightForeground,
	}

	return theme
}

func init() {
	// Register the kai theme with the theme manager
	RegisterTheme("kai", NewKaiTheme())
}
