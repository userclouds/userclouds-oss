package common

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// FetchFunc is a function that fetches a page of results
type FetchFunc[T any] func(ctx context.Context, cursor string, limit int) ([]T, string, error)

// TableColumn defines a column in a table
type TableColumn struct {
	Header string
	Width  int // 0 = dynamic width (auto-size to content), >0 = fixed width
}

// TableRow represents a row of data
type TableRow []string

// TableRenderFunc converts items to table rows with column definitions
type TableRenderFunc[T any] func(items []T) ([]TableColumn, []TableRow)

// PagerConfig configures the pager
type PagerConfig[T any] struct {
	Ctx             context.Context
	FetchFunc       FetchFunc[T]
	TableRenderFunc TableRenderFunc[T]
	InitialCursor   string
	NoItemsMessage  string
	ItemName        string // e.g., "object types", "edges"
}

// pagerModel is the Bubble Tea model for the pager
type pagerModel[T any] struct {
	config        PagerConfig[T]
	items         []T
	renderedLines []string
	columnWidths  []int // Cached column widths
	cursor        string
	scrollOffset  int
	horizOffset   int // Horizontal scroll offset
	height        int
	width         int
	loading       bool
	err           error
	hasMore       bool
	isFirstPage   bool
}

type itemsLoadedMsg[T any] struct {
	items      []T
	nextCursor string
	err        error
}

func (m pagerModel[T]) Init() tea.Cmd {
	return m.fetchPage
}

func (m pagerModel[T]) fetchPage() tea.Msg {
	// Limit fetch size to avoid blocking
	limit := m.height - 3
	if limit < 20 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	items, nextCursor, err := m.config.FetchFunc(m.config.Ctx, m.cursor, limit)
	return itemsLoadedMsg[T]{
		items:      items,
		nextCursor: nextCursor,
		err:        err,
	}
}

func (m pagerModel[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		case " ", "f", "pgdown", "enter":
			// Page down - scroll or fetch more
			contentHeight := m.height - 2
			maxScroll := len(m.renderedLines) - contentHeight
			if maxScroll < 0 {
				maxScroll = 0
			}

			// If we're near the end and have more data, prefetch
			if m.scrollOffset+contentHeight*2 >= len(m.renderedLines) && m.hasMore && !m.loading {
				m.loading = true
				m.scrollOffset += contentHeight
				if m.scrollOffset > maxScroll {
					m.scrollOffset = maxScroll
				}
				return m, m.fetchPage
			}

			if m.scrollOffset < maxScroll {
				// Still have content to scroll through
				m.scrollOffset += contentHeight
				if m.scrollOffset > maxScroll {
					m.scrollOffset = maxScroll
				}
				return m, nil
			}

			// Already at the end
			return m, nil

		case "down", "j":
			// Scroll down one line
			contentHeight := m.height - 2
			maxScroll := len(m.renderedLines) - contentHeight
			if maxScroll < 0 {
				maxScroll = 0
			}

			if m.scrollOffset < maxScroll {
				m.scrollOffset++
				return m, nil
			} else if m.hasMore && !m.loading {
				// Reached end, fetch more
				m.loading = true
				return m, m.fetchPage
			}
			return m, nil

		case "up", "k":
			// Scroll up one line
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
			return m, nil

		case "b", "pgup":
			// Page up
			contentHeight := m.height - 2
			m.scrollOffset -= contentHeight
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
			return m, nil

		case "g", "home":
			// Go to top
			m.scrollOffset = 0
			return m, nil

		case "left", "h":
			// Scroll left
			if m.horizOffset > 0 {
				m.horizOffset -= 5
				if m.horizOffset < 0 {
					m.horizOffset = 0
				}
			}
			return m, nil

		case "right", "l":
			// Scroll right
			m.horizOffset += 5
			return m, nil

		case "0":
			// Go to beginning of line
			m.horizOffset = 0
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		return m, nil

	case itemsLoadedMsg[T]:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}

		if len(msg.items) == 0 {
			if m.isFirstPage {
				// Show empty message but stay in pager mode
				m.hasMore = false
				m.isFirstPage = false
				return m, nil
			}
			m.hasMore = false
			return m, nil
		}

		oldItemCount := len(m.items)

		// Append new items
		m.items = append(m.items, msg.items...)
		m.cursor = msg.nextCursor
		m.hasMore = msg.nextCursor != ""
		m.isFirstPage = false

		if oldItemCount == 0 {
			// First load - render everything and cache column widths
			m.renderedLines, m.columnWidths = m.renderTableInitial()
		} else {
			// Incremental load - only render new items with cached column widths
			newLines := m.renderTableIncremental(msg.items)
			m.renderedLines = append(m.renderedLines, newLines...)
		}

		return m, nil
	}

	return m, nil
}

// renderTableInitial renders the table for the first batch of items and calculates column widths
func (m pagerModel[T]) renderTableInitial() ([]string, []int) {
	if len(m.items) == 0 {
		return []string{}, []int{}
	}

	columns, rows := m.config.TableRenderFunc(m.items)

	// Calculate actual widths (use fixed width or compute max from data)
	actualWidths := make([]int, len(columns))

	for i, col := range columns {
		if col.Width > 0 {
			// Fixed width column
			actualWidths[i] = col.Width
		} else {
			// Dynamic width - compute from header and all row values
			maxWidth := len(col.Header)
			for _, row := range rows {
				if i < len(row) {
					if len(row[i]) > maxWidth {
						maxWidth = len(row[i])
					}
				}
			}
			actualWidths[i] = maxWidth
		}
	}

	var lines []string

	headerParts := make([]string, len(columns))
	for i, col := range columns {
		headerParts[i] = padOrTruncate(col.Header, actualWidths[i], col.Width == 0)
	}
	lines = append(lines, strings.Join(headerParts, "   "))

	for _, row := range rows {
		rowParts := make([]string, len(columns))
		for i := range columns {
			cellValue := ""
			if i < len(row) {
				cellValue = row[i]
			}
			// Truncate dynamic columns (Width == 0), don't truncate fixed columns
			rowParts[i] = padOrTruncate(cellValue, actualWidths[i], columns[i].Width == 0)
		}
		lines = append(lines, strings.Join(rowParts, "   "))
	}

	return lines, actualWidths
}

// renderTableIncremental renders only new items using cached column widths
// Note: Dynamic columns will use the width from the first batch and won't expand
func (m pagerModel[T]) renderTableIncremental(newItems []T) []string {
	if len(newItems) == 0 {
		return []string{}
	}

	columns, rows := m.config.TableRenderFunc(newItems)

	var lines []string

	for _, row := range rows {
		rowParts := make([]string, len(columns))
		for i := range columns {
			cellValue := ""
			if i < len(row) {
				cellValue = row[i]
			}
			width := m.columnWidths[i]
			// Truncate dynamic columns (Width == 0), don't truncate fixed columns
			rowParts[i] = padOrTruncate(cellValue, width, columns[i].Width == 0)
		}
		lines = append(lines, strings.Join(rowParts, "   "))
	}

	return lines
}

func (m pagerModel[T]) View() string {
	if m.err != nil {
		return m.err.Error() + "\n"
	}

	if m.loading && len(m.items) == 0 {
		return "Loading...\n"
	}

	var b strings.Builder

	// Show empty message if no data
	if len(m.items) == 0 && !m.loading {
		b.WriteString(m.config.NoItemsMessage)
		b.WriteString("\n\n")
	} else {
		// Calculate how many lines we can show
		contentHeight := m.height - 2 // Reserve 1 line for status

		// Show the visible portion of rendered lines with horizontal scrolling
		visibleLines := m.renderedLines
		if m.scrollOffset < len(visibleLines) {
			endIdx := m.scrollOffset + contentHeight
			if endIdx > len(visibleLines) {
				endIdx = len(visibleLines)
			}
			for _, line := range visibleLines[m.scrollOffset:endIdx] {
				// Apply horizontal scrolling
				displayLine := line
				if m.horizOffset < len(line) {
					displayLine = line[m.horizOffset:]
				} else {
					displayLine = ""
				}
				b.WriteString(displayLine)
				b.WriteString("\n")
			}
		}
	}

	// Status line
	statusStyle := lipgloss.NewStyle().
		Reverse(true).
		Width(m.width)

	statusText := ""
	if m.loading {
		statusText = "Loading..."
	} else if m.hasMore {
		statusText = ":"
	} else {
		statusText = "(END)"
	}

	// Add horizontal scroll indicator if scrolled
	if m.horizOffset > 0 {
		statusText = fmt.Sprintf("%s [→%d]", statusText, m.horizOffset)
	}

	b.WriteString(statusStyle.Render(statusText))

	return b.String()
}

// padOrTruncate pads or truncates a string to the specified width
// If truncate is true, strings longer than width will be truncated with "..."
// If truncate is false, strings can extend beyond width (for horizontal scrolling)
func padOrTruncate(s string, width int, truncate bool) string {
	if len(s) > width {
		if truncate {
			// Truncate with ellipsis
			if width <= 3 {
				return s[:width]
			}
			return s[:width-3] + "..."
		}
		// Don't truncate - allow horizontal scrolling
		return s
	}
	// Pad to width
	return s + strings.Repeat(" ", width-len(s))
}

// RunPager runs the interactive pager
func RunPager[T any](config PagerConfig[T]) error {
	// Check if terminal is interactive
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		// Non-interactive mode - just dump all data
		return runNonInteractive(config)
	}

	// Run Bubble Tea pager
	initialModel := pagerModel[T]{
		config:       config,
		items:        []T{},
		cursor:       config.InitialCursor,
		scrollOffset: 0,
		horizOffset:  0,
		loading:      true,
		hasMore:      true,
		isFirstPage:  true,
	}

	p := tea.NewProgram(
		initialModel,
		tea.WithAltScreen(),
	)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Check if the pager model has an error (from fetching data)
	if m, ok := finalModel.(pagerModel[T]); ok && m.err != nil {
		return m.err
	}

	return nil
}

// runNonInteractive dumps all data without pagination
func runNonInteractive[T any](config PagerConfig[T]) error {
	cursor := config.InitialCursor
	limit := 100
	allItems := []T{}

	for {
		items, nextCursor, err := config.FetchFunc(config.Ctx, cursor, limit)
		if err != nil {
			return fmt.Errorf("failed to list %s: %w", config.ItemName, err)
		}

		if len(items) == 0 {
			break
		}

		allItems = append(allItems, items...)

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	if len(allItems) == 0 {
		fmt.Println(config.NoItemsMessage)
		return nil
	}

	columns, rows := config.TableRenderFunc(allItems)

	// Calculate actual widths (use fixed width or compute max from data)
	actualWidths := make([]int, len(columns))
	for i, col := range columns {
		if col.Width > 0 {
			// Fixed width column
			actualWidths[i] = col.Width
		} else {
			// Dynamic width - compute from header and all row values
			maxWidth := len(col.Header)
			for _, row := range rows {
				if i < len(row) {
					if len(row[i]) > maxWidth {
						maxWidth = len(row[i])
					}
				}
			}
			actualWidths[i] = maxWidth
		}
	}

	headerParts := make([]string, len(columns))
	for i, col := range columns {
		headerParts[i] = padOrTruncate(col.Header, actualWidths[i], false)
	}
	fmt.Println(strings.Join(headerParts, "   "))

	for _, row := range rows {
		rowParts := make([]string, len(columns))
		for i := range columns {
			cellValue := ""
			if i < len(row) {
				cellValue = row[i]
			}
			// Don't truncate in non-interactive mode - let terminal handle it
			rowParts[i] = padOrTruncate(cellValue, actualWidths[i], false)
		}
		fmt.Println(strings.Join(rowParts, "   "))
	}

	return nil
}
