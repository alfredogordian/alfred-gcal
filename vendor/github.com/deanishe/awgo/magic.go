//
// Copyright (c) 2016 Dean Jackson <deanishe@deanishe.net>
//
// MIT Licence. See http://opensource.org/licenses/MIT
//
// Created on 2016-11-05
//

package aw

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// DefaultMagicPrefix is the default prefix for "magic" arguments.
// This can be overriden with the MagicPrefix value in Options.
const DefaultMagicPrefix = "workflow:"

// Magic actions registered by default.
var (
	DefaultMagicActions = []MagicAction{
		openLogMagic{},    // Opens log file
		openCacheMagic{},  // Opens cache directory
		clearCacheMagic{}, // Clears cache directory
		openDataMagic{},   // Opens data directory
		clearDataMagic{},  // Clears data directory
		resetMagic{},      // Clears cache and data directories
	}
)

// MagicActions contains the registered magic actions. See the MagicAction
// interface for full documentation.
type MagicActions map[string]MagicAction

// Register adds a MagicAction to the mapping. Previous entries are overwritten.
func (ma MagicActions) Register(actions ...MagicAction) {
	for _, action := range actions {
		ma[action.Keyword()] = action
	}
}

// Unregister removes a MagicAction from the mapping (based on its keyword).
func (ma MagicActions) Unregister(actions ...MagicAction) {
	for _, action := range actions {
		delete(ma, action.Keyword())
	}
}

// Args runs a magic action or returns command-line arguments.
// It parses args for magic actions. If it finds one, it takes
// control of your workflow and runs the action.
//
// If not magic actions are found, it returns args.
func (ma MagicActions) Args(args []string, prefix string) []string {
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if strings.HasPrefix(arg, prefix) {
			query := arg[len(prefix):]
			action := ma[query]
			if action != nil {
				log.Printf(action.RunText())
				NewItem(action.RunText()).
					Icon(IconInfo).
					Valid(false)
				SendFeedback()
				if err := action.Run(); err != nil {
					log.Printf("Error running magic arg `%s`: %s", action.Description(), err)
					finishLog(true)
				}
				finishLog(false)
				os.Exit(0)
			} else {
				for kw, action := range ma {
					NewItem(action.Keyword()).
						Subtitle(action.Description()).
						Valid(false).
						Icon(IconInfo).
						UID(action.Description()).
						Autocomplete(prefix + kw).
						Match(fmt.Sprintf("%s %s", action.Keyword(), action.Description()))
				}
				Filter(query)
				WarnEmpty("No matching action", "Try another query?")
				SendFeedback()
				os.Exit(0)
			}
		}
	}
	return args
}

// MagicAction is a command that is called directly by AwGo (i.e. your workflow
// code is not run) if its keyword is passed in a user query. Magic Actions are
// mainly aimed at making debugging and supporting users easier (via the
// built-in actions), but it also provides a simple way to integrate your own
// commands that don't need a "real" UI (via Item.Autocomplete("<prefix>:XYZ")
// + Item.Valid(false)).
//
// The "update" sub-package registers a Magic Action to check for and install
// an update, for example.
//
// The built-in Magic Actions provide useful functions for debugging problems
// with workflows, so you, the developer, don't have to implement them yourself
// and don't have to hand-hold users through the process of digging out files
// buried somewhere deep in ~/Library. For example, you can simply request that
// a user enter "workflow:log" to open the log file or "workflow:delcache" to
// delete any cached data, instead of asking them to root around somewhere in
// ~/Library.
//
// To use Magic Actions, it's imperative that your workflow retrieves
// command-line arguments via Args()/Workflow.Args() instead of accessing
// os.Args directly (or at least calls Args()/Workflow.Args()).
//
// These functions return os.Args[1:], but first check if any argument starts
// with the "magic" prefix ("workflow:" by default).
//
// If so, AwGo will take control of the workflow (i.e. your code will no
// longer be run) and run its own "magic" mode. In this mode, it checks
// if the rest of the user query matches the keyword for a registered
// MagicAction, and if so, it runs that action, displaying RunText() in
// Alfred (if it's a Script Filter) and the log file & debugger.
//
// If no keyword matches, AwGo sends a list of available magic actions
// to Alfred, filtered by the user's query. Hitting TAB or RETURN on
// an item will run it.
//
//
// The built-in magic actions are:
//
//    Keyword           | Action
//    --------------------------------------------------------------------------------------
//    <prefix>log       | Open workflow's log file in the default app (usually Console).
//    <prefix>data      | Open workflow's data directory in the default app (usually Finder).
//    <prefix>cache     | Open workflow's data directory in the default app (usually Finder).
//    <prefix>deldata   | Delete everything in the workflow's data directory.
//    <prefix>delcache  | Delete everything in the workflow's cache directory.
//    <prefix>reset     | Delete everything in the workflow's data and cache directories.
//    <prefix>help      | Open help URL in default browser.
//                      | Only registered if you have set a HelpURL.
//    <prefix>update    | Check for updates and install a newer version of the workflow
//                      | if available.
//                      | Only registered if you have set an Updater.
//
type MagicAction interface {
	// Keyword is what the user must enter to run the action after
	// AwGo has recognised the magic prefix. So if the prefix is "workflow:"
	// (the default), a user must enter the query "workflow:<keyword>" to
	// execute this action.
	Keyword() string
	// Description is shown when a user has entered "magic" mode, but
	// the query does not yet match a keyword.
	Description() string
	// RunText is sent to Alfred and written to the log file & debugger when
	// the action is run.
	RunText() string
	// Run is called when the Magic Action is triggered.
	Run() error
}

// Opens workflow's log file.
type openLogMagic struct{}

func (a openLogMagic) Keyword() string     { return "log" }
func (a openLogMagic) Description() string { return "Open workflow's log file" }
func (a openLogMagic) RunText() string     { return "Opening log file…" }
func (a openLogMagic) Run() error          { return OpenLog() }

// Opens workflow's data directory.
type openDataMagic struct{}

func (a openDataMagic) Keyword() string     { return "data" }
func (a openDataMagic) Description() string { return "Open workflow's data directory" }
func (a openDataMagic) RunText() string     { return "Opening data directory…" }
func (a openDataMagic) Run() error          { return OpenData() }

// Opens workflow's cache directory.
type openCacheMagic struct{}

func (a openCacheMagic) Keyword() string     { return "cache" }
func (a openCacheMagic) Description() string { return "Open workflow's cache directory" }
func (a openCacheMagic) RunText() string     { return "Opening cache directory…" }
func (a openCacheMagic) Run() error          { return OpenCache() }

// Deletes the contents of the workflow's cache directory.
type clearCacheMagic struct{}

func (a clearCacheMagic) Keyword() string     { return "delcache" }
func (a clearCacheMagic) Description() string { return "Delete workflow's cached data" }
func (a clearCacheMagic) RunText() string     { return "Deleted workflow's cached data" }
func (a clearCacheMagic) Run() error          { return ClearCache() }

// Deletes the contents of the workflow's data directory.
type clearDataMagic struct{}

func (a clearDataMagic) Keyword() string     { return "deldata" }
func (a clearDataMagic) Description() string { return "Delete workflow's saved data" }
func (a clearDataMagic) RunText() string     { return "Deleted workflow's saved data" }
func (a clearDataMagic) Run() error          { return ClearData() }

// Deletes the contents of the workflow's cache & data directories.
type resetMagic struct{}

func (a resetMagic) Keyword() string     { return "reset" }
func (a resetMagic) Description() string { return "Delete all saved and cached workflow data" }
func (a resetMagic) RunText() string     { return "Deleted workflow saved and cached data" }
func (a resetMagic) Run() error          { return Reset() }

// Opens URL in default browser.
type helpMagic struct {
	URL string
}

func (a helpMagic) Keyword() string     { return "help" }
func (a helpMagic) Description() string { return "Open workflow help URL in default browser" }
func (a helpMagic) RunText() string     { return "Opening help in your browser…" }
func (a helpMagic) Run() error {
	cmd := exec.Command("open", a.URL)
	return cmd.Run()
}

// Updates the workflow if a newer release is available.
type updateMagic struct {
	updater Updater
}

func (a updateMagic) Keyword() string     { return "update" }
func (a updateMagic) Description() string { return "Check for updates, and install if one is available" }
func (a updateMagic) RunText() string     { return "Fetching update…" }
func (a updateMagic) Run() error {
	if err := a.updater.CheckForUpdate(); err != nil {
		return err
	}
	if a.updater.UpdateAvailable() {
		return a.updater.Install()
	}
	log.Println("No update available")
	return nil
}
