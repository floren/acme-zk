# zk client for acme

This is a really basic [zk](https://github.com/floren/zk) client for the [acme](https://en.wikipedia.org/wiki/Acme_(text_editor)) text editor. It's written in Go and should work on both Plan 9 and Unix systems.

You'll need to have zk set up already (see the repo for docs).

When you run acme-zk, it creates a window showing your note hierarchy, initially in a fully collapsed state. Notes with sub-notes have a '+' sign next to them. Middle-click on a note to expand it and show the subnotes.

Right-click a note to open it. Make your changes, then execute Put to save.

This client does not yet support:

* Note creation
* Linking
* File operations
* Grepping

## some weird philosophical crap

It's a weird balance when writing an acme tool to decide where you manage custom windows vs. just opening something as a regular file. Because each note is really just a text file on the disk somewhere, I *could* just plumb that filename when you right-click a note. I chose to instead "manage" note editing myself because it makes it easier to implement more advanced stuff later, like linking, subnote creation, and viewing the files associated with the note.
