package chat

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

// BlockRenderer provides composable rendering options for message blocks
type BlockRenderer struct {
	theme            theme.Theme
	textColor        lipgloss.AdaptiveColor
	backgroundColor  *lipgloss.AdaptiveColor
	border           bool
	borderColor      *lipgloss.AdaptiveColor
	borderStyle      lipgloss.Border
	paddingTop       int
	paddingBottom    int
	paddingLeft      int
	paddingRight     int
	marginTop        int
	marginBottom     int
	marginLeft       int
	marginRight      int
	width            int
	maxWidth         int
	align            lipgloss.Position
	bold             bool
	italic           bool
	underline        bool
	strikethrough    bool
}

// RenderOption configures a BlockRenderer
type RenderOption func(*BlockRenderer)

// NewBlockRenderer creates a new block renderer with default settings
func NewBlockRenderer(theme theme.Theme, opts ...RenderOption) *BlockRenderer {
	r := &BlockRenderer{
		theme:           theme,
		textColor:       theme.Text(),
		backgroundColor: nil,
		border:          true,
		borderStyle:     lipgloss.RoundedBorder(),
		align:           lipgloss.Left,
	}
	
	// Apply options
	for _, opt := range opts {
		opt(r)
	}
	
	return r
}

// Render applies all configured styles and returns the rendered content
func (r *BlockRenderer) Render(content string) string {
	style := lipgloss.NewStyle()
	
	// Text properties
	style = style.Foreground(r.textColor)
	if r.backgroundColor != nil {
		style = style.Background(*r.backgroundColor)
	}
	
	// Text decorations
	if r.bold {
		style = style.Bold(true)
	}
	if r.italic {
		style = style.Italic(true)
	}
	if r.underline {
		style = style.Underline(true)
	}
	if r.strikethrough {
		style = style.Strikethrough(true)
	}
	
	// Padding
	style = style.Padding(r.paddingTop, r.paddingRight, r.paddingBottom, r.paddingLeft)
	
	// Margins
	style = style.Margin(r.marginTop, r.marginRight, r.marginBottom, r.marginLeft)
	
	// Width constraints
	if r.width > 0 {
		style = style.Width(r.width)
	}
	if r.maxWidth > 0 {
		style = style.MaxWidth(r.maxWidth)
	}
	
	// Alignment
	style = style.Align(r.align)
	
	// Border
	if r.border {
		style = style.Border(r.borderStyle)
		if r.borderColor != nil {
			style = style.BorderForeground(*r.borderColor)
		}
	}
	
	return style.Render(content)
}

// Option functions for fluent configuration

// WithTextColor sets the text color
func WithTextColor(color lipgloss.AdaptiveColor) RenderOption {
	return func(r *BlockRenderer) {
		r.textColor = color
	}
}

// WithBackgroundColor sets the background color
func WithBackgroundColor(color lipgloss.AdaptiveColor) RenderOption {
	return func(r *BlockRenderer) {
		r.backgroundColor = &color
	}
}

// WithNoBorder removes the border
func WithNoBorder() RenderOption {
	return func(r *BlockRenderer) {
		r.border = false
	}
}

// WithBorder sets border with optional color
func WithBorder(style lipgloss.Border, color ...lipgloss.AdaptiveColor) RenderOption {
	return func(r *BlockRenderer) {
		r.border = true
		r.borderStyle = style
		if len(color) > 0 {
			r.borderColor = &color[0]
		}
	}
}

// WithPadding sets uniform padding
func WithPadding(padding int) RenderOption {
	return func(r *BlockRenderer) {
		r.paddingTop = padding
		r.paddingBottom = padding
		r.paddingLeft = padding
		r.paddingRight = padding
	}
}

// WithPaddingX sets horizontal padding
func WithPaddingX(padding int) RenderOption {
	return func(r *BlockRenderer) {
		r.paddingLeft = padding
		r.paddingRight = padding
	}
}

// WithPaddingY sets vertical padding
func WithPaddingY(padding int) RenderOption {
	return func(r *BlockRenderer) {
		r.paddingTop = padding
		r.paddingBottom = padding
	}
}

// WithMargin sets uniform margin
func WithMargin(margin int) RenderOption {
	return func(r *BlockRenderer) {
		r.marginTop = margin
		r.marginBottom = margin
		r.marginLeft = margin
		r.marginRight = margin
	}
}

// WithMarginX sets horizontal margin
func WithMarginX(margin int) RenderOption {
	return func(r *BlockRenderer) {
		r.marginLeft = margin
		r.marginRight = margin
	}
}

// WithMarginY sets vertical margin
func WithMarginY(margin int) RenderOption {
	return func(r *BlockRenderer) {
		r.marginTop = margin
		r.marginBottom = margin
	}
}

// WithMarginBottom sets bottom margin
func WithMarginBottom(margin int) RenderOption {
	return func(r *BlockRenderer) {
		r.marginBottom = margin
	}
}

// WithWidth sets fixed width
func WithWidth(width int) RenderOption {
	return func(r *BlockRenderer) {
		r.width = width
	}
}

// WithMaxWidth sets maximum width
func WithMaxWidth(maxWidth int) RenderOption {
	return func(r *BlockRenderer) {
		r.maxWidth = maxWidth
	}
}

// WithAlign sets text alignment
func WithAlign(align lipgloss.Position) RenderOption {
	return func(r *BlockRenderer) {
		r.align = align
	}
}

// WithBold makes text bold
func WithBold() RenderOption {
	return func(r *BlockRenderer) {
		r.bold = true
	}
}

// WithItalic makes text italic
func WithItalic() RenderOption {
	return func(r *BlockRenderer) {
		r.italic = true
	}
}

// WithUnderline underlines text
func WithUnderline() RenderOption {
	return func(r *BlockRenderer) {
		r.underline = true
	}
}

// WithStrikethrough adds strikethrough to text
func WithStrikethrough() RenderOption {
	return func(r *BlockRenderer) {
		r.strikethrough = true
	}
}

// Preset renderers for common use cases

// UserMessageRenderer creates a renderer preset for user messages
func UserMessageRenderer(theme theme.Theme) *BlockRenderer {
	return NewBlockRenderer(theme,
		WithBackgroundColor(theme.BackgroundSecondary()),
		WithTextColor(theme.Text()),
		WithPadding(1),
		WithMarginY(1),
		WithBorder(lipgloss.RoundedBorder(), theme.Primary()),
	)
}

// AssistantMessageRenderer creates a renderer preset for assistant messages
func AssistantMessageRenderer(theme theme.Theme) *BlockRenderer {
	return NewBlockRenderer(theme,
		WithBackgroundColor(theme.BackgroundDarker()),
		WithTextColor(theme.Text()),
		WithPadding(1),
		WithMarginY(1),
		WithBorder(lipgloss.RoundedBorder(), theme.Secondary()),
	)
}

// SystemMessageRenderer creates a renderer preset for system messages
func SystemMessageRenderer(theme theme.Theme) *BlockRenderer {
	return NewBlockRenderer(theme,
		WithTextColor(theme.TextMuted()),
		WithItalic(),
		WithNoBorder(),
		WithPaddingX(2),
	)
}

// ErrorMessageRenderer creates a renderer preset for error messages
func ErrorMessageRenderer(theme theme.Theme) *BlockRenderer {
	return NewBlockRenderer(theme,
		WithBackgroundColor(theme.BackgroundDarker()),
		WithTextColor(theme.Error()),
		WithPadding(1),
		WithBorder(lipgloss.DoubleBorder(), theme.Error()),
	)
}

// CodeBlockRenderer creates a renderer preset for code blocks
func CodeBlockRenderer(theme theme.Theme) *BlockRenderer {
	return NewBlockRenderer(theme,
		WithBackgroundColor(theme.BackgroundSecondary()),
		WithTextColor(theme.Text()),
		WithPadding(1),
		WithBorder(lipgloss.NormalBorder(), theme.BorderNormal()),
	)
}