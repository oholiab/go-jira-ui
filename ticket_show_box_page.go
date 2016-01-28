package jiraui

import (
	"fmt"
	ui "github.com/gizak/termui"
	"regexp"
)

const (
	defaultBoxMaxWrapWidth = 50
)

type TicketShowBoxPage struct {
	BaseListPage
	CommandBarFragment
	StatusBarFragment
	MaxWrapWidth uint
	TicketId     string
	Template     string
	apiBody      interface{}
	TicketTrail  []*TicketShowBoxPage // previously viewed tickets in drill-down
	WrapWidth    uint
	opts         map[string]interface{}
}

func (p *TicketShowBoxPage) Search() {
	s := p.ActiveSearch
	n := len(p.cachedResults)
	if s.command == "" {
		return
	}
	increment := 1
	if s.directionUp {
		increment = -1
	}
	// we use modulo here so we can loop through every line.
	// adding 'n' means we never have '-1 % n'.
	startLine := (p.selectedLine + n + increment) % n
	for i := startLine; i != p.selectedLine; i = (i + increment + n) % n {
		if s.re.MatchString(p.cachedResults[i]) {
			p.SetSelectedLine(i)
			p.Update()
			break
		}
	}
}

func (p *TicketShowBoxPage) SelectItem() {
	selected := p.cachedResults[p.selectedLine]
	if ok, _ := regexp.MatchString(`^epic_links:`, selected); ok {
		q := new(TicketListPage)
		q.ActiveQuery.Name = fmt.Sprintf("Open Tasks in Epic %s", p.TicketId)
		q.ActiveQuery.JQL = fmt.Sprintf("\"Epic Link\" = %s AND resolution = Unresolved", p.TicketId)
		currentPage = q
	} else {
		newTicketId := findTicketIdInString(selected)
		if newTicketId == "" {
			return
		} else if newTicketId == p.TicketId {
			return
		}
		q := new(TicketShowBoxPage)
		q.TicketId = newTicketId
		q.TicketTrail = append(p.TicketTrail, p)
		currentPage = q
	}
	changePage()
}

func (p *TicketShowBoxPage) Id() string {
	return p.TicketId
}

func (p *TicketShowBoxPage) PreviousPara() {
	newDisplayLine := 0
	if p.selectedLine == 0 {
		return
	}
	for i := p.selectedLine - 1; i > 0; i-- {
		if ok, _ := regexp.MatchString(`^\s*$`, p.cachedResults[i]); ok {
			newDisplayLine = i
			break
		}
	}
	p.PreviousLine(p.selectedLine - newDisplayLine)
}

func (p *TicketShowBoxPage) NextPara() {
	newDisplayLine := len(p.cachedResults) - 1
	if p.selectedLine == newDisplayLine {
		return
	}
	for i := p.selectedLine + 1; i < len(p.cachedResults); i++ {
		if ok, _ := regexp.MatchString(`^\s*$`, p.cachedResults[i]); ok {
			newDisplayLine = i
			break
		}
	}
	p.NextLine(newDisplayLine - p.selectedLine)
}

func (p *TicketShowBoxPage) GoBack() {
	if len(p.TicketTrail) == 0 {
		if ticketListPage != nil {
			currentPage = ticketListPage
		} else {
			currentPage = ticketQueryPage
		}
	} else {
		last := len(p.TicketTrail) - 1
		currentPage = p.TicketTrail[last]
	}
	changePage()
}

func (p *TicketShowBoxPage) EditTicket() {
	runJiraCmdEdit(p.TicketId)
}

func (p *TicketShowBoxPage) ActiveTicketId() string {
	return p.TicketId
}

func (p *TicketShowBoxPage) ticketTrailAsString() (trail string) {
	for i := len(p.TicketTrail) - 1; i >= 0; i-- {
		q := *p.TicketTrail[i]
		trail = trail + " <- " + q.Id()
	}
	return trail
}

func (p *TicketShowBoxPage) Refresh() {
	pDeref := &p
	q := *pDeref
	q.cachedResults = make([]string, 0)
	q.apiBody = nil
	currentPage = q
	changePage()
	q.Create()
}

func (p *TicketShowBoxPage) Update() {
	ls := p.uiList
	log.Debugf("TicketShowBoxPage.Update(): self:        %s (%p), ls: (%p)", p.Id(), p, ls)
	p.markActiveLine()
	log.Debugf("someline: %s", p.displayLines[p.firstDisplayLine:][0])
	blockTexts := SplitBlocks(p.displayLines[p.firstDisplayLine:])
	blocks := []*ui.List{}

	for _, blockText := range blockTexts {
		block := ui.NewList()
		block.BorderLabel = "List"
		block.Items = blockText
		block.Height = len(blockText) + 2
		ui.Body.AddRows(
			ui.NewRow(
				//ui.NewCol(6, 0, ls),
				ui.NewCol(12, 0, block)))
		blocks = append(blocks, block)
	}
	ui.Body.Align()
	ui.Render(ui.Body)
	//	for _, block := range blocks {
	//		log.Debugf("someblock")
	//		ui.Render(block)
	//	}
	p.statusBar.Update()
	p.commandBar.Update()
}

func SplitBlocks(lines []string) [][]string {
	splitLines := [][]string{}
	cacheLines := []string{}
	for _, line := range lines {
		if ok, _ := regexp.MatchString(`^\(break\)`, line); ok {
			splitLines = append(splitLines, cacheLines)
			cacheLines = []string{}
		} else {
			cacheLines = append(cacheLines, line)
		}
	}
	if len(cacheLines) != 0 {
		splitLines = append(splitLines, cacheLines)
	}
	return splitLines
}

func (p *TicketShowBoxPage) Create() {
	log.Debugf("TicketShowBoxPage.Create(): self:        %s (%p)", p.Id(), p)
	log.Debugf("TicketShowBoxPage.Create(): currentPage: %s (%p)", currentPage.Id(), currentPage)
	p.opts = getJiraOpts()
	if p.TicketId == "" {
		p.TicketId = ticketListPage.GetSelectedTicketId()
	}
	if p.MaxWrapWidth == 0 {
		if m := p.opts["max_wrap"]; m != nil {
			p.MaxWrapWidth = uint(m.(int64))
		} else {
			p.MaxWrapWidth = defaultBoxMaxWrapWidth
		}
	}
	ui.Clear()
	ls := ui.NewList()
	if p.statusBar == nil {
		p.statusBar = new(StatusBar)
	}
	if p.commandBar == nil {
		p.commandBar = new(CommandBar)
	}
	p.uiList = ls
	if p.Template == "" {
		if templateOpt := p.opts["template"]; templateOpt == nil {
			p.Template = "jira_ui_view"
		} else {
			p.Template = templateOpt.(string)
		}
	}
	innerWidth := uint(ui.TermWidth()) - 3
	if innerWidth < p.MaxWrapWidth {
		p.WrapWidth = innerWidth
	} else {
		p.WrapWidth = p.MaxWrapWidth
	}
	if p.apiBody == nil {
		p.apiBody, _ = FetchJiraTicket(p.TicketId)
	}
	p.cachedResults = WrapText(JiraTicketAsStrings(p.apiBody, p.Template), p.WrapWidth)
	p.displayLines = make([]string, len(p.cachedResults))
	if p.selectedLine >= len(p.cachedResults) {
		p.selectedLine = len(p.cachedResults) - 1
	}
	ls.ItemFgColor = ui.ColorYellow
	ls.Height = ui.TermHeight() - 2
	ls.Width = ui.TermWidth()
	ls.Border = true
	ls.BorderLabel = fmt.Sprintf("%s %s", p.TicketId, p.ticketTrailAsString())
	ls.Y = 0
	p.statusBar.Create()
	p.commandBar.Create()
	p.Update()
}
