package display

import (
	"strconv"
	"strings"
	"sync"

	"github.com/gdamore/tcell"
	"github.com/makeworld-the-better-one/amfora/structs"
	"gitlab.com/tslocum/cview"
)

type tabMode int

const (
	modeOff        tabMode = iota // Regular mode
	modeLinkSelect                // When the enter key is pressed, allow for tab-based link navigation
)

// tabHist holds the history for a tab.
type tabHistory struct {
	urls []string
	pos  int // Position: where in the list of URLs we are
}

// tab hold the information needed for each browser tab.
type tab struct {
	page        *structs.Page
	view        *cview.TextView
	mode        tabMode
	history     *tabHistory
	reformatMut *sync.Mutex // Mutex for reformatting, so there's only one reformat job at once
	selected    string      // The current text or link selected
	selectedID  string      // The cview region ID for the selected text/link
	barLabel    string      // The bottomBar label for the tab
	barText     string      // The bottomBar text for the tab
}

// makeNewTab initializes an tab struct with no content.
func makeNewTab() *tab {
	t := tab{
		page: &structs.Page{},
		view: cview.NewTextView().
			SetDynamicColors(true).
			SetRegions(true).
			SetScrollable(true).
			SetWrap(false).
			SetChangedFunc(func() {
				App.Draw()
			}),
		mode:        modeOff,
		history:     &tabHistory{},
		reformatMut: &sync.Mutex{},
	}
	t.view.SetDoneFunc(func(key tcell.Key) {
		// Altered from: https://gitlab.com/tslocum/cview/-/blob/master/demos/textview/main.go
		// Handles being able to select and "click" links with the enter and tab keys

		tab := curTab // Don't let it change in the middle of the code

		defer tabs[tab].saveBottomBar()

		if key == tcell.KeyEsc {
			// Stop highlighting
			tabs[tab].view.Highlight("")
			bottomBar.SetLabel("")
			bottomBar.SetText(tabs[tab].page.Url)
			tabs[tab].mode = modeOff
		}

		currentSelection := tabs[tab].view.GetHighlights()
		numSelections := len(tabs[tab].page.Links)

		if key == tcell.KeyEnter {
			if len(currentSelection) > 0 && len(tabs[tab].page.Links) > 0 {
				// A link was selected, "click" it and load the page it's for
				bottomBar.SetLabel("")
				tabs[tab].mode = modeOff
				linkN, _ := strconv.Atoi(currentSelection[0])
				followLink(tab, tabs[tab].page.Url, tabs[tab].page.Links[linkN])
				return
			} else {
				tabs[tab].view.Highlight("0").ScrollToHighlight()
				// Display link URL in bottomBar
				bottomBar.SetLabel("[::b]Link: [::-]")
				bottomBar.SetText(tabs[tab].page.Links[0])
				tabs[tab].selected = tabs[tab].page.Links[0]
				tabs[tab].selectedID = "0"
			}
		} else if len(currentSelection) > 0 {
			// There's still a selection, but a different key was pressed, not Enter

			index, _ := strconv.Atoi(currentSelection[0])
			if key == tcell.KeyTab {
				index = (index + 1) % numSelections
			} else if key == tcell.KeyBacktab {
				index = (index - 1 + numSelections) % numSelections
			} else {
				return
			}
			tabs[tab].view.Highlight(strconv.Itoa(index)).ScrollToHighlight()
			// Display link URL in bottomBar
			bottomBar.SetLabel("[::b]Link: [::-]")
			bottomBar.SetText(tabs[tab].page.Links[index])
			tabs[tab].selected = tabs[tab].page.Links[index]
			tabs[tab].selectedID = currentSelection[0]
		}
	})

	return &t
}

// addToHistory adds the given URL to history.
// It assumes the URL is currently being loaded and displayed on the page.
func (t *tab) addToHistory(u string) {
	if t.history.pos < len(t.history.urls)-1 {
		// We're somewhere in the middle of the history instead, with URLs ahead and behind.
		// The URLs ahead need to be removed so this new URL is the most recent item in the history
		t.history.urls = t.history.urls[:t.history.pos+1]
	}
	t.history.urls = append(t.history.urls, u)
	t.history.pos++
}

// pageUp scrolls up 75% of the height of the terminal, like Bombadillo.
func (t *tab) pageUp() {
	row, col := t.view.GetScrollOffset()
	t.view.ScrollTo(row-(termH/4)*3, col)
}

// pageDown scrolls down 75% of the height of the terminal, like Bombadillo.
func (t *tab) pageDown() {
	row, col := t.view.GetScrollOffset()
	t.view.ScrollTo(row+(termH/4)*3, col)
}

// hasContent returns true when the tab has a page that could be displayed.
// The most likely situation where false would be returned is when the default
// new tab content is being displayed.
func (t *tab) hasContent() bool {
	if t.page == nil || t.view == nil {
		return false
	}
	if t.page.Url == "" {
		return false
	}
	if strings.HasPrefix(t.page.Url, "about:") {
		return false
	}
	if t.page.Content == "" {
		return false
	}
	return true
}

// saveScroll saves where in the page the user was.
// It should be used whenever moving from one page to another.
func (t *tab) saveScroll() {
	// It will also be saved in the cache because the cache uses the same pointer
	row, col := t.view.GetScrollOffset()
	t.page.Row = row
	t.page.Column = col
}

// applyScroll applies the saved scroll values to the page and tab.
// It should only be used when going backward and forward.
func (t *tab) applyScroll() {
	t.view.ScrollTo(t.page.Row, t.page.Column)
}

// saveBottomBar saves the current bottomBar values in the tab.
func (t *tab) saveBottomBar() {
	t.barLabel = bottomBar.GetLabel()
	t.barText = bottomBar.GetText()
}

// applyBottomBar sets the bottomBar using the stored tab values
func (t *tab) applyBottomBar() {
	bottomBar.SetLabel(t.barLabel)
	bottomBar.SetText(t.barText)
}