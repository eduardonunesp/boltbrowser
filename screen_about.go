package main

import (
	"fmt"

	"github.com/br0xen/termbox-util"
	"github.com/nsf/termbox-go"
)

/*
Command is a struct for associating descriptions to keys
*/
type Command struct {
	key         string
	description string
}

/*
AboutScreen is just a basic 'int' type that we can extend to make this screen
*/
type AboutScreen int

func drawCommandAtPoint(cmd Command, xPos int, yPos int, style Style) {
	termboxUtil.DrawStringAtPoint(fmt.Sprintf("%6s", cmd.key), xPos, yPos, style.defaultFg, style.defaultBg)
	termboxUtil.DrawStringAtPoint(cmd.description, xPos+8, yPos, style.defaultFg, style.defaultBg)
}

func (screen *AboutScreen) handleKeyEvent(event termbox.Event) int {
	return BrowserScreenIndex
}

func (screen *AboutScreen) performLayout() {}

func (screen *AboutScreen) drawScreen(style Style) {
	defaultFg := style.defaultFg
	defaultBg := style.defaultBg
	width, height := termbox.Size()
	template := [...]string{
		" _______  _______  ___    _______  _______  ______    _______  _     _  _______  _______  ______   ",
		"|  _    ||       ||   |  |       ||  _    ||    _ |  |       || | _ | ||       ||       ||    _ |  ",
		"| |_|   ||   _   ||   |  |_     _|| |_|   ||   | ||  |   _   || || || ||  _____||    ___||   | ||  ",
		"|       ||  | |  ||   |    |   |  |       ||   |_||_ |  | |  ||       || |_____ |   |___ |   |_||_ ",
		"|  _   | |  |_|  ||   |___ |   |  |  _   | |    __  ||  |_|  ||       ||_____  ||    ___||    __  |",
		"| |_|   ||       ||       ||   |  | |_|   ||   |  | ||       ||   _   | _____| ||   |___ |   |  | |",
		"|_______||_______||_______||___|  |_______||___|  |_||_______||__| |__||_______||_______||___|  |_|",
	}
	if width < 100 {
		template = [...]string{
			" ____  ____  _   _____  ____  ____   ____  _     _  ____  ___  ____   ",
			"|  _ ||    || | |     ||  _ ||  _ | |    || | _ | ||    ||   ||  _ |  ",
			"| |_||| _  || | |_   _|| |_||| | || | _  || || || || ___||  _|| | ||  ",
			"|    ||| | || |   | |  |    || |_|| || | ||       |||___ | |_ | |_||_ ",
			"|  _ |||_| || |___| |  |  _ ||  _  |||_| ||       ||__  ||  _||  __  |",
			"| |_|||    ||     | |  | |_||| | | ||    ||   _   | __| || |_ | |  | |",
			"|____||____||_____|_|  |____||_| |_||____||__| |__||____||___||_|  |_|",
		}
	}
	firstLine := template[0]
	startX := (width - len(firstLine)) / 2
	//startX := (width - len(firstLine)) / 2
	startY := ((height - 2*len(template)) / 2) - 2
	xPos := startX
	yPos := startY
	if height <= 20 {
		title := "bboltBrowser"
		startY = 0
		yPos = 0
		termboxUtil.DrawStringAtPoint(title, (width-len(title))/2, startY, style.titleFg, style.titleBg)
	} else {
		if height < 25 {
			startY = 0
			yPos = 0
		}
		for _, line := range template {
			xPos = startX
			for _, runeValue := range line {
				bg := defaultBg
				displayRune := ' '
				if runeValue != ' ' {
					//bg = termbox.Attribute(125)
					displayRune = runeValue
					termbox.SetCell(xPos, yPos, displayRune, defaultFg, bg)
				}
				xPos++
			}
			yPos++
		}
	}
	yPos++
	versionString := fmt.Sprintf("Version: %0.1f", VersionNum)
	termboxUtil.DrawStringAtPoint(versionString, (width-len(versionString))/2, yPos, style.defaultFg, style.defaultBg)

	commands1 := []Command{
		{"h,←", "close parent"},
		{"j,↓", "down"},
		{"k,↑", "up"},
		{"l,→", "open item"},
		{"J", "scroll right pane down"},
		{"K", "scroll right pane up"},
		{"", ""},
		{"g", "goto top"},
		{"G", "goto bottom"},
		{"", ""},
		{"ctrl+f", "jump down"},
		{"ctrl+b", "jump up"},
	}

	commands2 := []Command{
		{"p,P", "create pair/at parent"},
		{"b,B", "create bucket/at parent"},
		{"e", "edit value of pair"},
		{"r", "rename pair/bucket"},
		{"", ""},
		{"", ""},
		{"D", "delete item"},
		{"x,X", "export as string/json to file"},
		{"", ""},
		{"?", "this screen"},
		{"q", "quit program"},
	}
	var maxCmd1 int
	for k := range commands1 {
		tst := len(commands1[k].key) + 1 + len(commands1[k].description)
		if tst > maxCmd1 {
			maxCmd1 = tst
		}
	}
	var maxCmd2 int
	for k := range commands2 {
		tst := len(commands2[k].key) + 1 + len(commands2[k].description)
		if tst > maxCmd2 {
			maxCmd2 = tst
		}
	}
	xPos = (width / 2) - ((maxCmd1 + maxCmd2) / 2)
	yPos++

	for k := range commands1 {
		drawCommandAtPoint(commands1[k], xPos, yPos+1+k, style)
	}
	for k := range commands2 {
		drawCommandAtPoint(commands2[k], xPos+40, yPos+1+k, style)
	}
	exitTxt := "Press any key to return to browser"
	termboxUtil.DrawStringAtPoint(exitTxt, (width-len(exitTxt))/2, height-1, style.titleFg, style.titleBg)
}
