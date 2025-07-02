package layout

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Position represents where to place an overlay
type Position int

const (
	Center Position = iota
	Top
	Bottom
	Left
	Right
	TopLeft
	TopRight
	BottomLeft
	BottomRight
)

// PlaceOverlay places content over a background at the specified position
func PlaceOverlay(width, height int, overlay, background string, pos Position) string {
	overlayLines := strings.Split(overlay, "\n")
	backgroundLines := strings.Split(background, "\n")
	
	// Ensure we have enough background lines
	for len(backgroundLines) < height {
		backgroundLines = append(backgroundLines, strings.Repeat(" ", width))
	}
	
	// Calculate overlay dimensions
	overlayHeight := len(overlayLines)
	overlayWidth := 0
	for _, line := range overlayLines {
		if w := lipgloss.Width(line); w > overlayWidth {
			overlayWidth = w
		}
	}
	
	// Calculate position
	var startX, startY int
	switch pos {
	case Center:
		startX = (width - overlayWidth) / 2
		startY = (height - overlayHeight) / 2
	case Top:
		startX = (width - overlayWidth) / 2
		startY = 0
	case Bottom:
		startX = (width - overlayWidth) / 2
		startY = height - overlayHeight
	case Left:
		startX = 0
		startY = (height - overlayHeight) / 2
	case Right:
		startX = width - overlayWidth
		startY = (height - overlayHeight) / 2
	case TopLeft:
		startX = 0
		startY = 0
	case TopRight:
		startX = width - overlayWidth
		startY = 0
	case BottomLeft:
		startX = 0
		startY = height - overlayHeight
	case BottomRight:
		startX = width - overlayWidth
		startY = height - overlayHeight
	}
	
	// Ensure position is within bounds
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}
	if startX+overlayWidth > width {
		startX = width - overlayWidth
	}
	if startY+overlayHeight > height {
		startY = height - overlayHeight
	}
	
	// Create result lines
	result := make([]string, height)
	copy(result, backgroundLines[:height])
	
	// Overlay the content
	for i, line := range overlayLines {
		if y := startY + i; y >= 0 && y < height {
			// Convert background line to runes for proper positioning
			bgRunes := []rune(result[y])
			
			// Ensure background line is wide enough
			for len(bgRunes) < width {
				bgRunes = append(bgRunes, ' ')
			}
			
			// Overlay the line
			overlayRunes := []rune(line)
			for j, r := range overlayRunes {
				if x := startX + j; x >= 0 && x < len(bgRunes) {
					bgRunes[x] = r
				}
			}
			
			result[y] = string(bgRunes)
		}
	}
	
	return strings.Join(result[:height], "\n")
}