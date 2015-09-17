package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/br0xen/termbox-util"
	"github.com/nsf/termbox-go"
)

/*
ViewPort helps keep track of what's being displayed on the screen
*/
type ViewPort struct {
	bytesPerRow  int
	numberOfRows int
	firstRow     int
}

/*
BrowserScreen holds all that's going on :D
*/
type BrowserScreen struct {
	db             *BoltDB
	viewPort       ViewPort
	queuedCommand  string
	currentPath    []string
	currentType    int
	message        string
	mode           BrowserMode
	inputModal     *termbox_util.InputModal
	confirmModal   *termbox_util.ConfirmModal
	messageTimeout time.Duration
	messageTime    time.Time
}

/*
BrowserMode is just for designating the mode that we're in
*/
type BrowserMode int

const (
	modeBrowse        = 16  // 0000 0001 0000
	modeChange        = 32  // 0000 0010 0000
	modeChangeKey     = 33  // 0000 0010 0001
	modeChangeVal     = 34  // 0000 0010 0010
	modeInsert        = 64  // 0000 0100 0000
	modeInsertBucket  = 65  // 0000 0100 0001
	modeInsertPair    = 68  // 0000 0100 0100
	modeInsertPairKey = 69  // 0000 0100 0101
	modeInserPairVal  = 70  // 0000 0100 0110
	modeDelete        = 256 // 0001 0000 0000
	modeModToParent   = 8   // 0000 0000 1000
)

/*
BoltType is just for tracking what type of db item we're looking at
*/
type BoltType int

const (
	typeBucket = iota
	typePair
)

func (screen *BrowserScreen) handleKeyEvent(event termbox.Event) int {
	if screen.mode == 0 {
		screen.mode = modeBrowse
	}
	if screen.mode == modeBrowse {
		return screen.handleBrowseKeyEvent(event)
	} else if screen.mode&modeChange == modeChange {
		return screen.handleInputKeyEvent(event)
	} else if screen.mode&modeInsert == modeInsert {
		return screen.handleInsertKeyEvent(event)
	} else if screen.mode == modeDelete {
		return screen.handleDeleteKeyEvent(event)
	}
	return BrowserScreenIndex
}

func (screen *BrowserScreen) handleBrowseKeyEvent(event termbox.Event) int {
	if event.Ch == '?' {
		// About
		return AboutScreenIndex

	} else if event.Ch == 'q' || event.Key == termbox.KeyEsc || event.Key == termbox.KeyCtrlC {
		// Quit
		return ExitScreenIndex

	} else if event.Ch == 'g' {
		// Jump to Beginning
		screen.currentPath = screen.db.getNextVisiblePath(nil)

	} else if event.Ch == 'G' {
		// Jump to End
		screen.currentPath = screen.db.getPrevVisiblePath(nil)

	} else if event.Key == termbox.KeyCtrlR {
		screen.refreshDatabase()

	} else if event.Key == termbox.KeyCtrlF {
		// Jump forward half a screen
		_, h := termbox.Size()
		half := h / 2
		screen.jumpCursorDown(half)

	} else if event.Key == termbox.KeyCtrlB {
		_, h := termbox.Size()
		half := h / 2
		screen.jumpCursorUp(half)

	} else if event.Ch == 'j' || event.Key == termbox.KeyArrowDown {
		screen.moveCursorDown()

	} else if event.Ch == 'k' || event.Key == termbox.KeyArrowUp {
		screen.moveCursorUp()

	} else if event.Ch == 'p' {
		// p creates a new pair at the current level
		screen.startInsertItem(typePair)
	} else if event.Ch == 'P' {
		// P creates a new pair at the parent level
		screen.startInsertItemAtParent(typePair)

	} else if event.Ch == 'b' {
		// b creates a new bucket at the current level
		screen.startInsertItem(typeBucket)
	} else if event.Ch == 'B' {
		// B creates a new bucket at the parent level
		screen.startInsertItemAtParent(typeBucket)

	} else if event.Ch == 'e' {
		b, p, _ := screen.db.getGenericFromPath(screen.currentPath)
		if b != nil {
			screen.setMessage("Cannot edit a bucket, did you mean to (r)ename?")
		} else if p != nil {
			screen.startEditItem()
		}

	} else if event.Ch == 'r' {
		screen.startRenameItem()

	} else if event.Key == termbox.KeyEnter {
		b, p, _ := screen.db.getGenericFromPath(screen.currentPath)
		if b != nil {
			screen.db.toggleOpenBucket(screen.currentPath)
		} else if p != nil {
			screen.startEditItem()
		}

	} else if event.Ch == 'l' || event.Key == termbox.KeyArrowRight {
		b, p, _ := screen.db.getGenericFromPath(screen.currentPath)
		// Select the current item
		if b != nil {
			screen.db.toggleOpenBucket(screen.currentPath)
		} else if p != nil {
			screen.startEditItem()
		} else {
			screen.setMessage("Not sure what to do here...")
		}

	} else if event.Ch == 'h' || event.Key == termbox.KeyArrowLeft {
		// If we are _on_ a bucket that's open, close it
		b, _, e := screen.db.getGenericFromPath(screen.currentPath)
		if e == nil && b != nil && b.expanded {
			screen.db.closeBucket(screen.currentPath)
		} else {
			if len(screen.currentPath) > 1 {
				parentBucket, err := screen.db.getBucketFromPath(screen.currentPath[:len(screen.currentPath)-1])
				if err == nil {
					screen.db.closeBucket(parentBucket.GetPath())
					// Figure out how far up we need to move the cursor
					screen.currentPath = parentBucket.GetPath()
				}
			} else {
				screen.db.closeBucket(screen.currentPath)
			}
		}

	} else if event.Ch == 'D' {
		screen.startDeleteItem()
	}
	return BrowserScreenIndex
}

func (screen *BrowserScreen) handleInputKeyEvent(event termbox.Event) int {
	if event.Key == termbox.KeyEsc {
		screen.mode = modeBrowse
		screen.inputModal.Clear()
	} else {
		screen.inputModal.HandleKeyPress(event)
		if screen.inputModal.IsDone() {
			b, p, _ := screen.db.getGenericFromPath(screen.currentPath)
			if b != nil {
				if screen.mode == modeChangeKey {
					newName := screen.inputModal.GetValue()
					if renameBucket(screen.currentPath, newName) != nil {
						screen.setMessage("Error renaming bucket.")
					} else {
						b.name = newName
						screen.currentPath[len(screen.currentPath)-1] = b.name
						screen.setMessage("Bucket Renamed!")
						screen.refreshDatabase()
					}
				}
			} else if p != nil {
				if screen.mode == modeChangeKey {
					newKey := screen.inputModal.GetValue()
					if updatePairKey(screen.currentPath, newKey) != nil {
						screen.setMessage("Error occurred updating Pair.")
					} else {
						p.key = newKey
						screen.currentPath[len(screen.currentPath)-1] = p.key
						screen.setMessage("Pair updated!")
						screen.refreshDatabase()
					}
				} else if screen.mode == modeChangeVal {
					newVal := screen.inputModal.GetValue()
					if updatePairValue(screen.currentPath, newVal) != nil {
						screen.setMessage("Error occurred updating Pair.")
					} else {
						p.val = newVal
						screen.setMessage("Pair updated!")
						screen.refreshDatabase()
					}
				}
			}
			screen.mode = modeBrowse
			screen.inputModal.Clear()
		}
	}
	return BrowserScreenIndex
}

func (screen *BrowserScreen) handleDeleteKeyEvent(event termbox.Event) int {
	screen.confirmModal.HandleKeyPress(event)
	if screen.confirmModal.IsDone() {
		if screen.confirmModal.IsAccepted() {
			holdNextPath := screen.db.getNextVisiblePath(screen.currentPath)
			holdPrevPath := screen.db.getPrevVisiblePath(screen.currentPath)
			if deleteKey(screen.currentPath) == nil {
				screen.refreshDatabase()
				// Move the current path endpoint appropriately
				//found_new_path := false
				if holdNextPath != nil {
					if len(holdNextPath) > 2 {
						if holdNextPath[len(holdNextPath)-2] == screen.currentPath[len(screen.currentPath)-2] {
							screen.currentPath = holdNextPath
						} else if holdPrevPath != nil {
							screen.currentPath = holdPrevPath
						} else {
							// Otherwise, go to the parent
							screen.currentPath = screen.currentPath[:(len(holdNextPath) - 2)]
						}
					} else {
						// Root bucket deleted, set to next
						screen.currentPath = holdNextPath
					}
				} else if holdPrevPath != nil {
					screen.currentPath = holdPrevPath
				} else {
					screen.currentPath = screen.currentPath[:0]
				}
			}
		}
		screen.mode = modeBrowse
		screen.confirmModal.Clear()
	}
	return BrowserScreenIndex
}

func (screen *BrowserScreen) handleInsertKeyEvent(event termbox.Event) int {
	if event.Key == termbox.KeyEsc {
		if len(screen.db.buckets) == 0 {
			return ExitScreenIndex
		}
		screen.mode = modeBrowse
		screen.inputModal.Clear()
	} else {
		screen.inputModal.HandleKeyPress(event)
		if screen.inputModal.IsDone() {
			newVal := screen.inputModal.GetValue()
			screen.inputModal.Clear()
			var insertPath []string
			if len(screen.currentPath) > 0 {
				_, p, e := screen.db.getGenericFromPath(screen.currentPath)
				if e != nil {
					screen.setMessage("Error Inserting new item. Invalid Path.")
				}
				insertPath = screen.currentPath
				// where are we inserting?
				if p != nil {
					// If we're sitting on a pair, we have to go to it's parent
					screen.mode = screen.mode | modeModToParent
				}
				if screen.mode&modeModToParent == modeModToParent {
					if len(screen.currentPath) > 1 {
						insertPath = screen.currentPath[:len(screen.currentPath)-1]
					} else {
						insertPath = make([]string, 0)
					}
				}
			}

			parentB, _, _ := screen.db.getGenericFromPath(insertPath)
			if screen.mode&modeInsertBucket == modeInsertBucket {
				err := insertBucket(insertPath, newVal)
				if err != nil {
					screen.setMessage(fmt.Sprintf("%s => %s", err, insertPath))
				} else {
					if parentB != nil {
						parentB.expanded = true
					}
				}
				screen.currentPath = append(insertPath, newVal)

				screen.refreshDatabase()
				screen.mode = modeBrowse
				screen.inputModal.Clear()
			} else if screen.mode&modeInsertPair == modeInsertPair {
				err := insertPair(insertPath, newVal, "")
				if err != nil {
					screen.setMessage(fmt.Sprintf("%s => %s", err, insertPath))
					screen.refreshDatabase()
					screen.mode = modeBrowse
					screen.inputModal.Clear()
				} else {
					if parentB != nil {
						parentB.expanded = true
					}
					screen.currentPath = append(insertPath, newVal)
					screen.refreshDatabase()
					screen.startEditItem()
				}
			}
		}
	}
	return BrowserScreenIndex
}

func (screen *BrowserScreen) jumpCursorUp(distance int) bool {
	// Jump up 'distance' lines
	visPaths, err := screen.db.buildVisiblePathSlice()
	if err == nil {
		findPath := strings.Join(screen.currentPath, "/")
		startJump := false
		for i := range visPaths {
			if visPaths[len(visPaths)-1-i] == findPath {
				startJump = true
			}
			if startJump {
				distance--
				if distance == 0 {
					screen.currentPath = strings.Split(visPaths[len(visPaths)-1-i], "/")
					break
				}
			}
		}
		if strings.Join(screen.currentPath, "/") == findPath {
			screen.currentPath = screen.db.getNextVisiblePath(nil)
		}
	}
	return true
}
func (screen *BrowserScreen) jumpCursorDown(distance int) bool {
	visPaths, err := screen.db.buildVisiblePathSlice()
	if err == nil {
		findPath := strings.Join(screen.currentPath, "/")
		startJump := false
		for i := range visPaths {
			if visPaths[i] == findPath {
				startJump = true
			}
			if startJump {
				distance--
				if distance == 0 {
					screen.currentPath = strings.Split(visPaths[i], "/")
					break
				}
			}
		}
		if strings.Join(screen.currentPath, "/") == findPath {
			screen.currentPath = screen.db.getPrevVisiblePath(nil)
		}
	}
	return true
}

func (screen *BrowserScreen) moveCursorUp() bool {
	newPath := screen.db.getPrevVisiblePath(screen.currentPath)
	if newPath != nil {
		screen.currentPath = newPath
		return true
	}
	return false
}
func (screen *BrowserScreen) moveCursorDown() bool {
	newPath := screen.db.getNextVisiblePath(screen.currentPath)
	if newPath != nil {
		screen.currentPath = newPath
		return true
	}
	return false
}

func (screen *BrowserScreen) performLayout() {}

func (screen *BrowserScreen) drawScreen(style Style) {
	if len(screen.db.buckets) == 0 && screen.mode&modeInsertBucket != modeInsertBucket {
		// Force a bucket insert
		screen.startInsertItemAtParent(typeBucket)
	}
	if screen.message == "" {
		screen.setMessageWithTimeout("Press '?' for help", -1)
	}
	screen.drawLeftPane(style)
	screen.drawRightPane(style)
	screen.drawHeader(style)
	screen.drawFooter(style)

	if screen.inputModal != nil {
		screen.inputModal.Draw()
	}
	if screen.mode == modeDelete {
		screen.confirmModal.Draw()
	}
}

func (screen *BrowserScreen) drawHeader(style Style) {
	width, _ := termbox.Size()
	spaces := strings.Repeat(" ", (width / 2))
	termbox_util.DrawStringAtPoint(fmt.Sprintf("%s%s%s", spaces, ProgramName, spaces), 0, 0, style.titleFg, style.titleBg)
}
func (screen *BrowserScreen) drawFooter(style Style) {
	if screen.messageTimeout > 0 && time.Since(screen.messageTime) > screen.messageTimeout {
		screen.clearMessage()
	}
	_, height := termbox.Size()
	termbox_util.DrawStringAtPoint(screen.message, 0, height-1, style.defaultFg, style.defaultBg)
}
func (screen *BrowserScreen) drawLeftPane(style Style) {
	w, h := termbox.Size()
	if w > 80 {
		w = w / 2
	}
	screen.viewPort.numberOfRows = h - 2

	termbox_util.FillWithChar('=', 0, 1, w, 1, style.defaultFg, style.defaultBg)
	y := 2
	screen.viewPort.firstRow = y
	if len(screen.currentPath) == 0 {
		screen.currentPath = screen.db.getNextVisiblePath(nil)
	}

	// So we know how much of the tree _wants_ to be visible
	// we only have screen.viewPort.numberOfRows of space though
	curPathSpot := 0
	visSlice, err := screen.db.buildVisiblePathSlice()
	if err == nil {
		for i := range visSlice {
			if strings.Join(screen.currentPath, "/") == visSlice[i] {
				curPathSpot = i
			}
		}
	}

	treeOffset := 0
	maxCursor := screen.viewPort.numberOfRows * 2 / 3
	if curPathSpot > maxCursor {
		treeOffset = curPathSpot - maxCursor
	}

	for i := range screen.db.buckets {
		// The drawBucket function returns how many lines it took up
		bktH := screen.drawBucket(&screen.db.buckets[i], style, (y - treeOffset))
		y += bktH
	}
}
func (screen *BrowserScreen) drawRightPane(style Style) {
	w, h := termbox.Size()
	if w > 80 {
		// Screen is wide enough, split it
		termbox_util.FillWithChar('=', 0, 1, w, 1, style.defaultFg, style.defaultBg)
		termbox_util.FillWithChar('|', (w / 2), screen.viewPort.firstRow-1, (w / 2), h, style.defaultFg, style.defaultBg)

		b, p, err := screen.db.getGenericFromPath(screen.currentPath)
		if err == nil {
			startX := (w / 2) + 2
			startY := 2
			if b != nil {
				termbox_util.DrawStringAtPoint(fmt.Sprintf("Path: %s", strings.Join(b.GetPath(), "/")), startX, startY, style.defaultFg, style.defaultBg)
				termbox_util.DrawStringAtPoint(fmt.Sprintf("Buckets: %d", len(b.buckets)), startX, startY+1, style.defaultFg, style.defaultBg)
				termbox_util.DrawStringAtPoint(fmt.Sprintf("Pairs: %d", len(b.pairs)), startX, startY+2, style.defaultFg, style.defaultBg)
			} else if p != nil {
				termbox_util.DrawStringAtPoint(fmt.Sprintf("Path: %s", strings.Join(p.GetPath(), "/")), startX, startY, style.defaultFg, style.defaultBg)
				termbox_util.DrawStringAtPoint(fmt.Sprintf("Key: %s", p.key), startX, startY+1, style.defaultFg, style.defaultBg)
				termbox_util.DrawStringAtPoint(fmt.Sprintf("Value: %s", p.val), startX, startY+2, style.defaultFg, style.defaultBg)
			}
		}
	}
}

/* drawBucket
 * @bkt *BoltBucket - The bucket to draw
 * @style Style - The style to use
 * @w int - The Width of the lines
 * @y int - The Y position to start drawing
 * return - The number of lines used
 */
func (screen *BrowserScreen) drawBucket(bkt *BoltBucket, style Style, y int) int {
	w, _ := termbox.Size()
	if w > 80 {
		w = w / 2
	}
	usedLines := 0
	bucketFg := style.defaultFg
	bucketBg := style.defaultBg
	if comparePaths(screen.currentPath, bkt.GetPath()) {
		bucketFg = style.cursorFg
		bucketBg = style.cursorBg
	}

	bktString := strings.Repeat(" ", len(bkt.GetPath())*2) //screen.db.getDepthFromPath(bkt.GetPath())*2)
	if bkt.expanded {
		bktString = bktString + "- " + bkt.name + " "
		bktString = fmt.Sprintf("%s%s", bktString, strings.Repeat(" ", (w-len(bktString))))

		termbox_util.DrawStringAtPoint(bktString, 0, (y + usedLines), bucketFg, bucketBg)
		usedLines++

		for i := range bkt.buckets {
			usedLines += screen.drawBucket(&bkt.buckets[i], style, y+usedLines)
		}
		for i := range bkt.pairs {
			usedLines += screen.drawPair(&bkt.pairs[i], style, y+usedLines)
		}
	} else {
		bktString = bktString + "+ " + bkt.name
		bktString = fmt.Sprintf("%s%s", bktString, strings.Repeat(" ", (w-len(bktString))))
		termbox_util.DrawStringAtPoint(bktString, 0, (y + usedLines), bucketFg, bucketBg)
		usedLines++
	}
	return usedLines
}

func (screen *BrowserScreen) drawPair(bp *BoltPair, style Style, y int) int {
	w, _ := termbox.Size()
	if w > 80 {
		w = w / 2
	}
	bucketFg := style.defaultFg
	bucketBg := style.defaultBg
	if comparePaths(screen.currentPath, bp.GetPath()) {
		bucketFg = style.cursorFg
		bucketBg = style.cursorBg
	}

	pairString := strings.Repeat(" ", len(bp.GetPath())*2) //screen.db.getDepthFromPath(bp.GetPath())*2)
	pairString = fmt.Sprintf("%s%s: %s", pairString, bp.key, bp.val)
	if w-len(pairString) > 0 {
		pairString = fmt.Sprintf("%s%s", pairString, strings.Repeat(" ", (w-len(pairString))))
	}
	termbox_util.DrawStringAtPoint(pairString, 0, y, bucketFg, bucketBg)
	return 1
}

func (screen *BrowserScreen) startDeleteItem() bool {
	b, p, e := screen.db.getGenericFromPath(screen.currentPath)
	if e == nil {
		w, h := termbox.Size()
		inpW, inpH := (w / 2), 6
		inpX, inpY := ((w / 2) - (inpW / 2)), ((h / 2) - inpH)
		mod := termbox_util.CreateConfirmModal("", inpX, inpY, inpW, inpH, termbox.ColorWhite, termbox.ColorBlack)
		if b != nil {
			mod.SetTitle(termbox_util.AlignText(fmt.Sprintf("Delete Bucket '%s'?", b.name), inpW-1, termbox_util.ALIGN_CENTER))
		} else if p != nil {
			mod.SetTitle(termbox_util.AlignText(fmt.Sprintf("Delete Pair '%s'?", p.key), inpW-1, termbox_util.ALIGN_CENTER))
		}
		mod.Show()
		mod.SetText(termbox_util.AlignText("This cannot be undone!", inpW-1, termbox_util.ALIGN_CENTER))
		screen.confirmModal = mod
		screen.mode = modeDelete
		return true
	}
	return false
}

func (screen *BrowserScreen) startEditItem() bool {
	_, p, e := screen.db.getGenericFromPath(screen.currentPath)
	if e == nil {
		w, h := termbox.Size()
		inpW, inpH := (w / 2), 6
		inpX, inpY := ((w / 2) - (inpW / 2)), ((h / 2) - inpH)
		mod := termbox_util.CreateInputModal("", inpX, inpY, inpW, inpH, termbox.ColorWhite, termbox.ColorBlack)
		if p != nil {
			mod.SetTitle(termbox_util.AlignText(fmt.Sprintf("Input new value for '%s'", p.key), inpW, termbox_util.ALIGN_CENTER))
			mod.SetValue(p.val)
		}
		mod.Show()
		screen.inputModal = mod
		screen.mode = modeChangeVal
		return true
	}
	return false
}

func (screen *BrowserScreen) startRenameItem() bool {
	b, p, e := screen.db.getGenericFromPath(screen.currentPath)
	if e == nil {
		w, h := termbox.Size()
		inpW, inpH := (w / 2), 6
		inpX, inpY := ((w / 2) - (inpW / 2)), ((h / 2) - inpH)
		mod := termbox_util.CreateInputModal("", inpX, inpY, inpW, inpH, termbox.ColorWhite, termbox.ColorBlack)
		if b != nil {
			mod.SetTitle(termbox_util.AlignText(fmt.Sprintf("Rename Bucket '%s' to:", b.name), inpW, termbox_util.ALIGN_CENTER))
			mod.SetValue(b.name)
		} else if p != nil {
			mod.SetTitle(termbox_util.AlignText(fmt.Sprintf("Rename Key '%s' to:", p.key), inpW, termbox_util.ALIGN_CENTER))
			mod.SetValue(p.key)
		}
		mod.Show()
		screen.inputModal = mod
		screen.mode = modeChangeKey
		return true
	}
	return false
}

func (screen *BrowserScreen) startInsertItemAtParent(tp BoltType) bool {
	w, h := termbox.Size()
	inpW, inpH := w-1, 7
	if w > 80 {
		inpW, inpH = (w / 2), 7
	}
	inpX, inpY := ((w / 2) - (inpW / 2)), ((h / 2) - inpH)
	mod := termbox_util.CreateInputModal("", inpX, inpY, inpW, inpH, termbox.ColorWhite, termbox.ColorBlack)
	screen.inputModal = mod
	if len(screen.currentPath) <= 0 {
		// in the root directory
		if tp == typeBucket {
			mod.SetTitle(termbox_util.AlignText("Create Root Bucket", inpW, termbox_util.ALIGN_CENTER))
			screen.mode = modeInsertBucket | modeModToParent
			mod.Show()
			return true
		}
	} else {
		var insPath string
		_, p, e := screen.db.getGenericFromPath(screen.currentPath[:len(screen.currentPath)-1])
		if e == nil && p != nil {
			insPath = strings.Join(screen.currentPath[:len(screen.currentPath)-2], "/") + "/"
		} else {
			insPath = strings.Join(screen.currentPath[:len(screen.currentPath)-1], "/") + "/"
		}
		titlePrfx := ""
		if tp == typeBucket {
			titlePrfx = "New Bucket: "
		} else if tp == typePair {
			titlePrfx = "New Pair: "
		}
		titleText := titlePrfx + insPath
		if len(titleText) > inpW {
			truncW := len(titleText) - inpW
			titleText = titlePrfx + "..." + insPath[truncW+3:]
		}
		if tp == typeBucket {
			mod.SetTitle(termbox_util.AlignText(titleText, inpW, termbox_util.ALIGN_CENTER))
			screen.mode = modeInsertBucket | modeModToParent
			mod.Show()
			return true
		} else if tp == typePair {
			mod.SetTitle(termbox_util.AlignText(titleText, inpW, termbox_util.ALIGN_CENTER))
			mod.Show()
			screen.mode = modeInsertPair | modeModToParent
			return true
		}
	}
	return false
}

func (screen *BrowserScreen) startInsertItem(tp BoltType) bool {
	w, h := termbox.Size()
	inpW, inpH := w-1, 7
	if w > 80 {
		inpW, inpH = (w / 2), 7
	}
	inpX, inpY := ((w / 2) - (inpW / 2)), ((h / 2) - inpH)
	mod := termbox_util.CreateInputModal("", inpX, inpY, inpW, inpH, termbox.ColorWhite, termbox.ColorBlack)
	screen.inputModal = mod
	var insPath string
	_, p, e := screen.db.getGenericFromPath(screen.currentPath)
	if e == nil && p != nil {
		insPath = strings.Join(screen.currentPath[:len(screen.currentPath)-1], "/") + "/"
	} else {
		insPath = strings.Join(screen.currentPath, "/") + "/"
	}
	titlePrfx := ""
	if tp == typeBucket {
		titlePrfx = "New Bucket: "
	} else if tp == typePair {
		titlePrfx = "New Pair: "
	}
	titleText := titlePrfx + insPath
	if len(titleText) > inpW {
		truncW := len(titleText) - inpW
		titleText = titlePrfx + "..." + insPath[truncW+3:]
	}
	if tp == typeBucket {
		mod.SetTitle(termbox_util.AlignText(titleText, inpW, termbox_util.ALIGN_CENTER))
		screen.mode = modeInsertBucket
		mod.Show()
		return true
	} else if tp == typePair {
		mod.SetTitle(termbox_util.AlignText(titleText, inpW, termbox_util.ALIGN_CENTER))
		mod.Show()
		screen.mode = modeInsertPair
		return true
	}
	return false
}

func (screen *BrowserScreen) setMessage(msg string) {
	screen.message = msg
	screen.messageTime = time.Now()
	screen.messageTimeout = time.Second * 2
}

/* setMessageWithTimeout lets you specify the timeout for the message
 * setting it to -1 means it won't timeout
 */
func (screen *BrowserScreen) setMessageWithTimeout(msg string, timeout time.Duration) {
	screen.message = msg
	screen.messageTime = time.Now()
	screen.messageTimeout = timeout
}

func (screen *BrowserScreen) clearMessage() {
	screen.message = ""
	screen.messageTimeout = -1
}

func (screen *BrowserScreen) refreshDatabase() {
	shadowDB := screen.db
	screen.db = screen.db.refreshDatabase()
	screen.db.syncOpenBuckets(shadowDB)
}

func comparePaths(p1, p2 []string) bool {
	return strings.Join(p1, "/") == strings.Join(p2, "/")
}