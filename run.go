package jiraui

import (
	"fmt"
	"github.com/coryb/optigo"
	ui "github.com/gizak/termui"
	"github.com/op/go-logging"
	"os"
)

var exitNow = false

type EditPager interface {
	DeleteRuneBackward()
	InsertRune(r rune)
	Update()
	Create()
}

type TicketCommander interface {
	ActiveTicketId() string
	Refresh()
}

type Searcher interface {
	SetSearch(string)
	Search()
}

type CommandBoxer interface {
	SetCommandMode(bool)
	ExecuteCommand()
	CommandMode() bool
	CommandBar() *CommandBar
	Update()
}

type GoBacker interface {
	GoBack()
}

type Refresher interface {
	Refresh()
}

type ItemSelecter interface {
	SelectItem()
}

type TicketEditer interface {
	EditTicket()
}

type TicketCommenter interface {
	CommentTicket()
}

type PagePager interface {
	NextLine(int)
	PreviousLine(int)
	NextPara()
	PreviousPara()
	NextPage()
	PreviousPage()
	TopOfPage()
	BottomOfPage()
	IsPopulated() bool
	Update()
}

type Navigable interface {
	Create()
	Update()
	Id() string
}

var currentPage Navigable
var previousPage Navigable

var ticketQueryPage *QueryPage
var helpPage *HelpPage
var ticketListPage *TicketListPage
var labelListPage *LabelListPage
var sortOrderPage *SortOrderPage
var passwordInputBox *PasswordInputBox

func changePage() {
	switch currentPage.(type) {
	case *QueryPage:
		log.Debugf("changePage: QueryPage %s (%p)", currentPage.Id(), currentPage)
		currentPage.Create()
	case *TicketListPage:
		log.Debugf("changePage: TicketListPage %s (%p)", currentPage.Id(), currentPage)
		currentPage.Create()
	case *SortOrderPage:
		log.Debugf("changePage: SortOrderPage %s (%p)", currentPage.Id(), currentPage)
		currentPage.Create()
	case *LabelListPage:
		log.Debugf("changePage: LabelListPage %s (%p)", currentPage.Id(), currentPage)
		currentPage.Create()
	case *TicketShowPage:
		log.Debugf("changePage: TicketShowPage %s (%p)", currentPage.Id(), currentPage)
		currentPage.Create()
	case *TicketShowBoxPage:
		log.Debugf("changePage: TicketShowBoxPage %s (%p)", currentPage.Id(), currentPage)
		currentPage.Create()
	case *HelpPage:
		log.Debugf("changePage: HelpPage %s (%p)", currentPage.Id(), currentPage)
		currentPage.Create()
	}
}

var (
	log    = logging.MustGetLogger("jiraui")
	format = "%{color}%{time:2006-01-02T15:04:05.000Z07:00} %{level:-5s} [%{shortfile}]%{color:reset} %{message}"
)

var cliOpts map[string]interface{}

func Run() {

	var err error
	logging.SetLevel(logging.NOTICE, "")

	usage := func(ok bool) {
		printer := fmt.Printf
		if !ok {
			printer = func(format string, args ...interface{}) (int, error) {
				return fmt.Fprintf(os.Stderr, format, args...)
			}
			defer func() {
				os.Exit(1)
			}()
		} else {
			defer func() {
				os.Exit(0)
			}()
		}
		output := fmt.Sprintf(`
Usage:
  jira-ui ls <Query Options> 
  jira-ui ISSUE
  jira-ui

General Options:
  -e --endpoint=URI   URI to use for jira
  -h --help           Show this usage
  -u --user=USER      Username to use for authenticaion
  -v --verbose        Increase output logging
  --skiplogin         Skip the login check. You must have a valid session token (eg via 'jira login')
  --version           Print version

Ticket View Options:
  -t --template=FILE  Template file to use for viewing tickets
  -m --max_wrap=VAL   Maximum word-wrap width when viewing ticket text (0 disables)

Query Options:
  -q --query=JQL            Jira Query Language expression for the search
  -f --queryfields=FIELDS   Fields that are used in "list" view

`)
		printer(output)
	}

	jiraCommands := map[string]string{
		"list":     "list",
		"ls":       "list",
		"password": "password",
		"passwd":   "password",
	}

	cliOpts = make(map[string]interface{})
	setopt := func(name string, value interface{}) {
		cliOpts[name] = value
	}

	op := optigo.NewDirectAssignParser(map[string]interface{}{
		"h|help": usage,
		"version": func() {
			fmt.Println(fmt.Sprintf("version: %s", VERSION))
			os.Exit(0)
		},
		"v|verbose+": func() {
			logging.SetLevel(logging.GetLevel("")+1, "")
		},
		"u|user=s":        setopt,
		"endpoint=s":      setopt,
		"q|query=s":       setopt,
		"f|queryfields=s": setopt,
		"t|template=s":    setopt,
		"m|max_wrap=i":    setopt,
		"skip_login":      setopt,
	})

	if err := op.ProcessAll(os.Args[1:]); err != nil {
		log.Error("%s", err)
		usage(false)
	}
	args := op.Args

	var command string
	if len(args) > 0 {
		if alias, ok := jiraCommands[args[0]]; ok {
			command = alias
			args = args[1:]
		} else {
			command = "view"
			args = args[0:]
		}
	} else {
		command = "toplevel"
	}

	requireArgs := func(count int) {
		if len(args) < count {
			log.Error("Not enough arguments. %d required, %d provided", count, len(args))
			usage(false)
		}
	}

	if val, ok := cliOpts["skip_login"]; !ok || !val.(bool) {
		err = ensureLoggedIntoJira()
		if err != nil {
			log.Error("Login failed. Aborting")
			os.Exit(2)
		}
	}

	err = ui.Init()
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	registerKeyboardHandlers()

	ticketQueryPage = new(QueryPage)
	passwordInputBox = new(PasswordInputBox)
	helpPage = new(HelpPage)

	switch command {
	case "list":
		ticketListPage = new(TicketListPage)
		if query := cliOpts["query"]; query == nil {
			log.Error("Must supply a --query option to %q", command)
			os.Exit(1)
		} else {
			ticketListPage.ActiveQuery.JQL = query.(string)
			ticketListPage.ActiveQuery.Name = "adhoc"
			currentPage = ticketListPage
		}
	case "view":
		requireArgs(1)
		p := new(TicketShowPage)
		p.TicketId = args[0]
		currentPage = p
	case "toplevel":
		currentPage = ticketQueryPage
	case "password":
		currentPage = passwordInputBox
	default:
		log.Error("Unknown command %s", command)
		os.Exit(1)
	}

	for exitNow != true {

		currentPage.Create()
		ui.Loop()

	}

}
