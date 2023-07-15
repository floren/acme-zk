package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"sync"

	"9fans.net/go/acme"
	libzk "github.com/floren/zk/libzk"
)

var (
	expandState = map[int]bool{}

	zk *libzk.ZK

	mtx sync.Mutex
)

func main() {
	var err error

	// Initialize the zk object
	zk, err = libzk.NewZK("/home/john/zk")
	if err != nil {
		log.Fatal(err)
	}

	// Now get an Acme window
	w, err := acme.New()
	if err != nil {
		log.Fatal(err)
	}
	w.Fprintf("tag", "Update")
	w.Name("zk")

	updateDisplay(w)

	for e := range w.EventChan() {
		switch e.C2 {
		case 'x':
			// execute in the tag
			switch string(e.Text) {
			case "Update":
				// Check the ZK state and refresh
				updateDisplay(w)
			default:
				// just run the command
				w.WriteEvent(e)
			}
		case 'X':
			// execute in the body
			// If they middle-clicked on a number or a plus sign, expand it
			// Figure out where they clicked
			id, err := getIdForEvent(w, e)
			if err != nil {
				continue
			}
			// now expand it
			mtx.Lock()
			expandState[id] = !isExpanded(id)
			mtx.Unlock()
			updateDisplay(w)
		case 'l', 'L':
			// Open something
			id, err := getIdForEvent(w, e)
			if err != nil {
				continue
			}
			go openNote(id)
		}
		if err != nil {
			fmt.Println(err)
		}
	}
}

// getIdForEvent takes an acme Event structure and finds out where in the main window it happened.
func getIdForEvent(w *acme.Win, e *acme.Event) (id int, err error) {
	var body, b []byte
	// The body will never be too massive, so just read it all
	body, err = w.ReadAll("body")
	if err != nil {
		return
	}
	buf := bytes.NewBuffer(body)
	line := 1
	for i := 0; i != e.Q0; {
		var r rune
		var sz int
		r, sz, err = buf.ReadRune()
		if err != nil {
			return
		}
		i += sz
		if r == '\n' {
			line++
		}
	}
	if err = w.Addr("%d", line); err != nil {
		return
	}
	if b, err = w.ReadAll("xdata"); err != nil {
		return
	}
	re := regexp.MustCompile(`\[(\d+)\]`)
	var matches []string
	if matches = re.FindStringSubmatch(string(b)); len(matches) < 2 {
		err = errors.New("Cannot find an ID")
		return
	}
	id, err = strconv.Atoi(matches[1])
	return
}

func isExpanded(id int) bool {
	expanded, _ := expandState[id]
	return expanded
}

func updateDisplay(w *acme.Win) {
	mtx.Lock()
	defer mtx.Unlock()
	w.Clear()
	w.Ctl("noscroll")
	w.Ctl("addr=dot")

	// Loop through the ZK tree and print it, with appropriate decorations
	var printer func(id, depth int) error
	printer = func(id, depth int) error {
		note, err := zk.GetNote(id)
		if err != nil {
			return err
		}
		indent := ""
		for i := 0; i < depth; i++ {
			indent += "	"
		}
		prefix := " "
		if len(note.Subnotes) > 0 || len(note.Files) > 0 {
			if isExpanded(id) {
				prefix = "-"
			} else {
				prefix = "+"
			}
		}
		fileCount := ""
		if len(note.Files) > 1 {
			fileCount = fmt.Sprintf(" (%d files)", len(note.Files))
		} else if len(note.Files) == 1 {
			fileCount = " (1 file)"
		}
		w.Fprintf("body", "%s%s [%d] %s%s\n", indent, prefix, id, note.Title, fileCount)
		if isExpanded(id) {
			for _, sub := range note.Subnotes {
				if err := printer(sub, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}

	// TODO: do something about the error
	printer(0, 0)

	w.Ctl("dot=addr\nshow")
}

func openNote(id int) {
	// dirty is true if the buffer has been changed
	// dirtyWarned is true if the user executed "Get" and has been warned that the buffer is dirty
	var dirty, dirtyWarned bool

	w, err := acme.New()
	if err != nil {
		fmt.Printf("couldn't create new acme window: %v\n", err)
		return
	}
	w.Name("zk/%d", id)
	// Tag
	w.Fprintf("tag", "Undo Redo Put")

	// Populate the body
	getBody := func() {
		note, err := zk.GetNote(id)
		if err != nil {
			w.Errf("Can't open note %d: %v", id, err)
			return
		}
		dirty = false
		w.Clear()
		w.Fprintf("body", "%s", note.Body)

		w.Addr("0")
		w.Ctl("dot=addr")
		w.Ctl("show")
		w.Ctl("clean")
	}

	getBody()

	for e := range w.EventChan() {
		switch e.C2 {
		case 'x':
			// execute in the tag
			switch string(e.Text) {
			case "Put":
				// First read what's in the window
				body, err := w.ReadAll("body")
				if err != nil {
					w.Errf("Can't read body: %v", err)
					continue
				}
				// TODO: implement change detection so we can warn
				// before we overwrite if it's been changed elsewhere
				zk.UpdateNote(id, string(body))
				dirty = false
				w.Ctl("clean")
			case "Get":
				// Re-read the note
				// throw a warning if body is dirty
				if dirty && !dirtyWarned {
					w.Err(fmt.Sprintf("Note %d has been changed, Get again to discard changes", id))
					dirtyWarned = true
					continue
				}
				dirtyWarned = false
				getBody()
			default:
				// just run the command
				w.WriteEvent(e)
			}
		case 'D':
			// text deleted from body
			fallthrough
		case 'I':
			// text inserted to the body
			if !dirty {
				w.Ctl("dirty")
				dirty = true
			}
		}
		if err != nil {
			w.Err(err.Error())
		}
	}
}
